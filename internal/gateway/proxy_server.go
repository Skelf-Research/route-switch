package gateway

import (
	"context"
	"encoding/json"
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
	datasetStore   dataset.Store
	analyticsStore analytics.Store
	server         *http.Server
	mux            *http.ServeMux
}

// NewProxyServer creates a new proxy server instance
func NewProxyServer(registry *PromptRegistry, loadBalancer *LoadBalancer, addr string, datasetStore dataset.Store, analyticsStore analytics.Store) *ProxyServer {
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

	// Select the appropriate prompt+model combination
	combination, err := ps.loadBalancer.SelectCombination()
	if err != nil {
		http.Error(w, "No available model combinations", http.StatusServiceUnavailable)
		return
	}

	// Transform the request using the optimized prompt
	transformedReq := ps.transformRequest(req, combination)

	// Get the appropriate provider
	provider, exists := ps.providers[combination.Provider]
	if !exists {
		http.Error(w, fmt.Sprintf("Provider %s not available", combination.Provider), http.StatusServiceUnavailable)
		return
	}

	// Call the model provider
	startTime := time.Now()
	response, renderedPrompt, err := ps.callProvider(provider, combination.Model, transformedReq)
	duration := time.Since(startTime)

	if err != nil {
		// Update performance metrics with failure
		ps.registry.UpdatePerformance(combination.ID, duration, false, 0)
		ps.recordDatasetEntry(ctx, combination, renderedPrompt, "", false, 0, map[string]interface{}{
			"error":            err.Error(),
			"request_metadata": transformedReq.Metadata,
		}, transformedReq.Variables)
		ps.recordAnalyticsEntry(ctx, combination, duration, false, 0, 0, 0, map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, fmt.Sprintf("Provider error: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate cost
	inputTokens := response.Usage.PromptTokens
	outputTokens := response.Usage.CompletionTokens
	cost, _ := provider.EstimateCost(combination.Model, inputTokens, outputTokens)

	// Update performance metrics
	ps.registry.UpdatePerformance(combination.ID, duration, true, cost)
	ps.recordDatasetEntry(ctx, combination, renderedPrompt, response.Choices[0].Message.Content, true, cost, map[string]interface{}{
		"request_metadata": transformedReq.Metadata,
		"input_tokens":     inputTokens,
		"output_tokens":    outputTokens,
	}, transformedReq.Variables)
	ps.recordAnalyticsEntry(ctx, combination, duration, true, cost, inputTokens, outputTokens, transformedReq.Metadata)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// transformRequest transforms the incoming request using the optimized prompt
func (ps *ProxyServer) transformRequest(req Request, combination *PromptCombination) Request {
	// For now, we'll replace the first user message with our optimized prompt
	// In a real implementation, we'd intelligently integrate the optimized prompt
	// with the user's actual request
	renderedPrompt := renderTemplate(combination.Prompt, req.Variables)

	if len(req.Messages) > 0 {
		// Create a new message that combines the optimized prompt with the user's input
		combinedContent := fmt.Sprintf("%s\n\nUser Request: %s", renderedPrompt, req.Messages[len(req.Messages)-1].Content)
		req.Messages[len(req.Messages)-1] = Message{
			Role:    "user",
			Content: combinedContent,
		}
	} else {
		// If no messages, create one with the optimized prompt
		req.Messages = []Message{
			{Role: "user", Content: renderedPrompt},
		}
	}

	return req
}

// callProvider calls the appropriate model provider with the request
func (ps *ProxyServer) callProvider(provider models.ModelProvider, modelName string, req Request) (*Response, string, error) {
	// Convert our request structure to the format expected by the model provider
	// For now, we'll just concatenate messages to form a single prompt
	var promptBuilder strings.Builder
	for _, msg := range req.Messages {
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

// AddCombination adds a prompt+model combination to the registry through the proxy
func (ps *ProxyServer) AddCombination(combination *PromptCombination) error {
	return ps.registry.AddCombination(combination)
}

// GetCombination retrieves a combination by ID
func (ps *ProxyServer) GetCombination(id string) (*PromptCombination, bool) {
	return ps.registry.GetCombination(id)
}
