package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/skelf-research/route-switch/internal/analytics"
	"github.com/skelf-research/route-switch/internal/models"
	"github.com/skelf-research/route-switch/internal/storage/dataset"
)

// Request represents an incoming request to the gateway
type Request struct {
	Model            string                 `json:"model"`
	Messages         []Message              `json:"messages"`
	Stream           bool                   `json:"stream,omitempty"`
	MaxTokens        int                    `json:"max_tokens,omitempty"`
	Temperature      float64                `json:"temperature,omitempty"`
	TopP             float64                `json:"top_p,omitempty"`
	Stop             []string               `json:"stop,omitempty"`
	PresencePenalty  float64                `json:"presence_penalty,omitempty"`
	FrequencyPenalty float64                `json:"frequency_penalty,omitempty"`
	User             string                 `json:"user,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Variables        map[string]interface{} `json:"variables,omitempty"`
}

// Message represents a message in a conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Response represents a response from a model provider
type Response struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
}

// Choice represents a choice in the response
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	Logprobs     *any    `json:"logprobs,omitempty"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ProxyServer handles incoming requests and routes them to appropriate model providers
type ProxyServer struct {
	registry       *PromptRegistry
	loadBalancer   *LoadBalancer
	providers      map[string]models.ModelProvider
	httpClient     *http.Client
	addr           string
	datasetStore   dataset.DatasetStore
	analyticsStore analytics.AnalyticsStore
	server         *http.Server
	mux            *http.ServeMux
}

// NewProxyServer creates a new proxy server instance
func NewProxyServer(registry *PromptRegistry, loadBalancer *LoadBalancer, addr string, datasetStore dataset.DatasetStore, analyticsStore analytics.AnalyticsStore) *ProxyServer {
	mux := http.NewServeMux()
	ps := &ProxyServer{
		registry:     registry,
		loadBalancer: loadBalancer,
		providers:    make(map[string]models.ModelProvider),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		addr:           addr,
		datasetStore:   datasetStore,
		analyticsStore: analyticsStore,
		mux:            mux,
	}

	ps.registerHandlers()
	ps.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  65 * time.Second,
		WriteTimeout: 65 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return ps
}

func (ps *ProxyServer) registerHandlers() {
	ps.mux.HandleFunc("/v1/chat/completions", ps.handleChatCompletions)
	ps.mux.HandleFunc("/health", ps.handleHealth)
	ps.mux.HandleFunc("/health/storage", ps.handleStorageHealth)
	ps.mux.HandleFunc("/status", ps.handleStatus)
	ps.mux.HandleFunc("/v1/system/analytics", ps.handleSystemAnalytics)
	ps.mux.HandleFunc("/v1/prompts/", ps.handlePromptStats)
}

// RegisterProvider registers a model provider with the proxy
func (ps *ProxyServer) RegisterProvider(name string, provider models.ModelProvider) {
	ps.providers[name] = provider
}

// Start starts the proxy server
func (ps *ProxyServer) Start() error {
	return ps.server.ListenAndServe()
}

// handleChatCompletions handles chat completion requests
func (ps *ProxyServer) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	// Parse the incoming request
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Select the appropriate prompt+model combination (respecting overrides)
	combination, err := ps.resolveCombination(req)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			http.Error(w, "no prompt combinations available", http.StatusServiceUnavailable)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	// Apply request-scoped overrides without mutating the registry entry
	effectiveCombination := *combination
	if req.Model != "" {
		effectiveCombination.Model = req.Model
	}
	if overrideModel := metadataString(req.Metadata, "model"); overrideModel != "" {
		effectiveCombination.Model = overrideModel
	}
	if overrideProvider := metadataString(req.Metadata, "provider"); overrideProvider != "" {
		effectiveCombination.Provider = overrideProvider
	}

	// Transform the request using the optimized prompt
	transformedReq := ps.transformRequest(req, &effectiveCombination)

	// Get the appropriate provider
	provider, exists := ps.providers[effectiveCombination.Provider]
	if !exists {
		http.Error(w, fmt.Sprintf("Provider %s not available", effectiveCombination.Provider), http.StatusServiceUnavailable)
		return
	}

	// Call the model provider
	startTime := time.Now()
	response, renderedPrompt, err := ps.callProvider(provider, effectiveCombination.Model, transformedReq)
	duration := time.Since(startTime)

	if err != nil {
		// Update performance metrics with failure
		ps.registry.UpdatePerformance(combination.ID, duration, false, 0)
		ps.recordDatasetEntry(ctx, &effectiveCombination, renderedPrompt, "", false, 0, map[string]interface{}{
			"error":            err.Error(),
			"request_metadata": transformedReq.Metadata,
		}, transformedReq.Variables)
		ps.recordAnalyticsEntry(ctx, &effectiveCombination, duration, false, 0, 0, 0, map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, fmt.Sprintf("Provider error: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate cost
	inputTokens := response.Usage.PromptTokens
	outputTokens := response.Usage.CompletionTokens
	cost, _ := provider.EstimateCost(effectiveCombination.Model, inputTokens, outputTokens)

	// Update performance metrics
	ps.registry.UpdatePerformance(combination.ID, duration, true, cost)
	ps.recordDatasetEntry(ctx, &effectiveCombination, renderedPrompt, response.Choices[0].Message.Content, true, cost, map[string]interface{}{
		"request_metadata": transformedReq.Metadata,
		"input_tokens":     inputTokens,
		"output_tokens":    outputTokens,
	}, transformedReq.Variables)
	ps.recordAnalyticsEntry(ctx, &effectiveCombination, duration, true, cost, inputTokens, outputTokens, transformedReq.Metadata)

	// Send response (support streaming)
	if transformedReq.Stream {
		ps.writeStreamResponse(w, response)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// transformRequest transforms the incoming request using the optimized prompt
func (ps *ProxyServer) transformRequest(req Request, combination *PromptCombination) Request {
	renderedPrompt := renderTemplate(combination.Prompt, req.Variables)

	systemMessage := Message{Role: "system", Content: renderedPrompt}
	newMessages := []Message{systemMessage}
	newMessages = append(newMessages, req.Messages...)
	req.Messages = newMessages

	return req
}

// callProvider calls the appropriate model provider with the request
func (ps *ProxyServer) callProvider(provider models.ModelProvider, modelName string, req Request) (*Response, string, error) {
	var promptBuilder strings.Builder
	for _, msg := range req.Messages {
		role := strings.ToUpper(msg.Role)
		if role == "" {
			role = "USER"
		}
		promptBuilder.WriteString(role)
		promptBuilder.WriteString(": ")
		promptBuilder.WriteString(msg.Content)
		promptBuilder.WriteString("\n")
	}
	prompt := promptBuilder.String()

	// Call the provider
	responseText, err := provider.CallModel(modelName, prompt)
	if err != nil {
		return nil, prompt, err
	}

	// Estimate token counts for the response
	inputTokens, _ := provider.GetTokenCount(prompt)
	outputTokens, _ := provider.GetTokenCount(responseText)

	// Create a mock response structure
	response := &Response{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   modelName,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: responseText,
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     inputTokens,
			CompletionTokens: outputTokens,
			TotalTokens:      inputTokens + outputTokens,
		},
	}

	return response, prompt, nil
}

func (ps *ProxyServer) recordDatasetEntry(ctx context.Context, combination *PromptCombination, inputPrompt, outputText string, success bool, cost float64, extraMetadata map[string]interface{}, variables map[string]interface{}) {
	if ps.datasetStore == nil {
		return
	}
	key := datasetKey(combination)
	metadata := map[string]interface{}{
		"provider":       combination.Provider,
		"combination_id": combination.ID,
	}
	if combination.TemplateID != "" {
		metadata["template_id"] = combination.TemplateID
	}
	for k, v := range extraMetadata {
		metadata[k] = v
	}
	record := &dataset.Record{
		PromptID:  key,
		Model:     combination.Model,
		Input:     inputPrompt,
		Output:    outputText,
		Variables: copyMap(variables),
		Success:   success,
		Cost:      cost,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
	_ = ps.datasetStore.AddRecord(ctx, key, record)
}

func (ps *ProxyServer) recordAnalyticsEntry(ctx context.Context, combination *PromptCombination, duration time.Duration, success bool, cost float64, inputTokens, outputTokens int, metadata map[string]interface{}) {
	if ps.analyticsStore == nil {
		return
	}
	record := &analytics.InvocationRecord{
		PromptID:      datasetKey(combination),
		TemplateID:    combination.TemplateID,
		CombinationID: combination.ID,
		Provider:      combination.Provider,
		Model:         combination.Model,
		Duration:      duration,
		Success:       success,
		Cost:          cost,
		InputTokens:   inputTokens,
		OutputTokens:  outputTokens,
		Metadata:      copyMap(metadata),
		CreatedAt:     time.Now(),
	}
	_ = ps.analyticsStore.RecordInvocation(ctx, record)
}

func datasetKey(combination *PromptCombination) string {
	if combination.TemplateID != "" {
		return combination.TemplateID
	}
	return combination.ID
}

func copyMap(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string]interface{}, len(src))
	for k, v := range src {
		cloned[k] = v
	}
	return cloned
}

func renderTemplate(promptTemplate string, variables map[string]interface{}) string {
	if len(variables) == 0 {
		return promptTemplate
	}
	rendered := promptTemplate
	for key, value := range variables {
		placeholder := fmt.Sprintf("{%s}", key)
		rendered = strings.ReplaceAll(rendered, placeholder, fmt.Sprint(value))
	}
	return rendered
}

// handleHealth handles health check requests
func (ps *ProxyServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"message": "Route-Switch Gateway is running",
	})
}

func (ps *ProxyServer) handleStorageHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	datasetStatus := "unavailable"
	if ps.datasetStore != nil {
		if _, err := ps.datasetStore.TotalCount(ctx, "__health__"); err == nil {
			datasetStatus = "ok"
		}
	}

	analyticsStatus := "unavailable"
	if ps.analyticsStore != nil {
		if _, err := ps.analyticsStore.QuerySystemStats(ctx); err == nil {
			analyticsStatus = "ok"
		}
	}

	json.NewEncoder(w).Encode(map[string]string{
		"dataset":   datasetStatus,
		"analytics": analyticsStatus,
	})
}

// handleStatus returns current combination status and latest analytics.
func (ps *ProxyServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	combinations := ps.registry.GetAllCombinations()
	var summaries []map[string]interface{}
	for _, combo := range combinations {
		summary := map[string]interface{}{
			"id":             combo.ID,
			"name":           combo.Name,
			"template_id":    combo.TemplateID,
			"model":          combo.Model,
			"provider":       combo.Provider,
			"weight":         combo.Weight,
			"performance":    combo.Performance,
			"metadata":       combo.Metadata,
			"last_optimized": combo.LastOptimized,
		}
		if stats := ps.fetchPromptStats(r.Context(), combo.TemplateID, combo.ID); stats != nil {
			summary["analytics"] = stats
		}
		summaries = append(summaries, summary)
	}

	resp := map[string]interface{}{
		"status":       "ok",
		"combinations": summaries,
		"count":        len(summaries),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (ps *ProxyServer) handleSystemAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if ps.analyticsStore == nil {
		http.Error(w, "analytics store not configured", http.StatusServiceUnavailable)
		return
	}
	stats, err := ps.analyticsStore.QuerySystemStats(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("query analytics: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (ps *ProxyServer) handlePromptStats(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/stats") {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if ps.analyticsStore == nil {
		http.Error(w, "analytics store not configured", http.StatusServiceUnavailable)
		return
	}
	segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(segments) != 4 || segments[0] != "v1" || segments[1] != "prompts" || segments[3] != "stats" {
		http.NotFound(w, r)
		return
	}

	id := segments[2]
	var stats *analytics.PromptStats
	if combo, ok := ps.registry.GetCombination(id); ok {
		stats = ps.fetchPromptStats(r.Context(), combo.TemplateID, combo.ID)
	} else {
		stats = ps.fetchPromptStats(r.Context(), id, id)
	}
	if stats == nil {
		http.Error(w, "stats not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (ps *ProxyServer) fetchPromptStats(ctx context.Context, templateID, fallbackID string) *analytics.PromptStats {
	if ps.analyticsStore == nil {
		return nil
	}

	filter := analytics.StatsFilter{}
	if templateID != "" {
		filter.TemplateID = templateID
	} else if fallbackID != "" {
		filter.PromptID = fallbackID
	} else {
		return nil
	}

	stats, err := ps.analyticsStore.QueryPromptStats(ctx, filter)
	if err != nil || stats == nil {
		return nil
	}
	return stats
}

func (ps *ProxyServer) resolveCombination(req Request) (*PromptCombination, error) {
	if id := metadataString(req.Metadata, "combination_id"); id != "" {
		if combo, ok := ps.registry.GetCombination(id); ok {
			return combo, nil
		}
		return nil, fmt.Errorf("combination %s not found", id)
	}

	if templateID := metadataString(req.Metadata, "template_id"); templateID != "" {
		for _, combo := range ps.registry.GetAllCombinations() {
			if combo.TemplateID == templateID {
				return combo, nil
			}
		}
		return nil, fmt.Errorf("template %s not registered", templateID)
	}

	if req.Model != "" {
		for _, combo := range ps.registry.GetAllCombinations() {
			if combo.Model == req.Model {
				return combo, nil
			}
		}
	}

	return ps.loadBalancer.SelectCombination()
}

func metadataString(meta map[string]interface{}, key string) string {
	if meta == nil {
		return ""
	}
	if value, ok := meta[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func (ps *ProxyServer) writeStreamResponse(w http.ResponseWriter, resp *Response) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	chunks := chunkContent(resp.Choices[0].Message.Content)
	for _, chunk := range chunks {
		event := map[string]interface{}{
			"id":     resp.ID,
			"object": "chat.completion.chunk",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]interface{}{
						"role":    "assistant",
						"content": chunk,
					},
					"finish_reason": nil,
				},
			},
		}
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	done := map[string]interface{}{
		"id":     resp.ID,
		"object": "chat.completion.chunk",
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"delta":         map[string]interface{}{},
				"finish_reason": resp.Choices[0].FinishReason,
			},
		},
		"usage": resp.Usage,
	}
	data, _ := json.Marshal(done)
	fmt.Fprintf(w, "data: %s\n\n", data)
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func chunkContent(content string) []string {
	const chunkSize = 200
	if len(content) <= chunkSize {
		return []string{content}
	}

	var chunks []string
	runes := []rune(content)
	for start := 0; start < len(runes); start += chunkSize {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}
	return chunks
}

// AddCombination adds a prompt+model combination to the registry through the proxy
func (ps *ProxyServer) AddCombination(combination *PromptCombination) error {
	return ps.registry.AddCombination(combination)
}

// GetCombination retrieves a combination by ID
func (ps *ProxyServer) GetCombination(id string) (*PromptCombination, bool) {
	return ps.registry.GetCombination(id)
}
