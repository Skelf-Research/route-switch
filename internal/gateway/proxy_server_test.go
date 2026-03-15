package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/skelf-research/route-switch/internal/analytics"
	"github.com/skelf-research/route-switch/internal/models"
)

type TestModelProvider struct {
	models map[string]models.Model
}

func (p *TestModelProvider) Name() string { return "TestProvider" }
func (p *TestModelProvider) ListModels() ([]models.Model, error) {
	var result []models.Model
	for _, m := range p.models {
		result = append(result, m)
	}
	return result, nil
}
func (p *TestModelProvider) GetModel(name string) (models.Model, error) {
	model, exists := p.models[name]
	if !exists {
		return models.Model{}, models.ErrNotFound
	}
	return model, nil
}
func (p *TestModelProvider) CallModel(modelName, prompt string) (string, error) {
	return "Response to: " + prompt, nil
}
func (p *TestModelProvider) EstimateCost(modelName string, inputTokens, outputTokens int) (float64, error) {
	return float64(inputTokens+outputTokens) * 0.00001, nil
}
func (p *TestModelProvider) GetTokenCount(text string) (int, error) {
	return len([]rune(text)), nil
}
func (p *TestModelProvider) Initialize(config map[string]interface{}) error { return nil }
func (p *TestModelProvider) Close() error                                   { return nil }

func TestProxyServer_BasicFunctionality(t *testing.T) {
	// Set up registry with a combination
	registry := NewPromptRegistry()

	combination := &PromptCombination{
		ID:          "test-combo",
		Name:        "test-combo",
		Prompt:      "Test prompt: ",
		Model:       "gpt-4",
		Provider:    "test",
		Weight:      10,
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
		Metadata:    map[string]interface{}{"test": true},
	}

	registry.AddCombination(combination)

	// Set up load balancer that always returns the test combination
	// For this test, we'll create a custom balancer
	loadBalancer := NewLoadBalancer(registry, RoundRobinStrategy)

	proxy := NewProxyServer(registry, loadBalancer, ":0", nil, nil) // Use port 0 for automatic assignment

	// Register a test provider
	testProvider := &TestModelProvider{
		models: map[string]models.Model{
			"gpt-4": {
				Name:         "gpt-4",
				Provider:     "OpenAI",
				CostPerToken: 0.00003,
				MaxTokens:    8192,
				Description:  "Most capable GPT-4 model",
			},
		},
	}
	proxy.RegisterProvider("test", testProvider)

	// Create a test HTTP request
	reqBody := `{
		"model": "gpt-4",
		"messages": [
			{
				"role": "user",
				"content": "Hello!"
			}
		]
	}`

	req, err := http.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler method directly
	proxy.handleChatCompletions(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check if the response body contains expected fields
	var response Response
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Object != "chat.completion" {
		t.Errorf("Expected response object 'chat.completion', got '%s'", response.Object)
	}

	if len(response.Choices) == 0 {
		t.Error("Expected at least one choice in response")
	}

	if response.Choices[0].Message.Content == "" {
		t.Error("Expected non-empty message content in response")
	}
}

func TestProxyServer_HealthCheck(t *testing.T) {
	registry := NewPromptRegistry()
	loadBalancer := NewLoadBalancer(registry, RoundRobinStrategy)
	proxy := NewProxyServer(registry, loadBalancer, ":0", nil, nil)

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	// Call the health check handler
	proxy.handleHealth(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check if the response contains expected health status
	var healthResponse map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &healthResponse)
	if err != nil {
		t.Fatalf("Failed to unmarshal health response: %v", err)
	}

	if healthResponse["status"] != "healthy" {
		t.Errorf("Expected health status 'healthy', got '%s'", healthResponse["status"])
	}
}

func TestProxyServer_InvalidMethod(t *testing.T) {
	registry := NewPromptRegistry()
	loadBalancer := NewLoadBalancer(registry, RoundRobinStrategy)
	proxy := NewProxyServer(registry, loadBalancer, ":0", nil, nil)

	req, err := http.NewRequest("GET", "/v1/chat/completions", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	// Call the handler method directly
	proxy.handleChatCompletions(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusMethodNotAllowed)
	}
}

func TestProxyServer_InvalidJSON(t *testing.T) {
	registry := NewPromptRegistry()
	loadBalancer := NewLoadBalancer(registry, RoundRobinStrategy)
	proxy := NewProxyServer(registry, loadBalancer, ":0", nil, nil)

	// Create a test HTTP request with invalid JSON
	req, err := http.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString("{invalid json}"))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	// Call the handler method directly
	proxy.handleChatCompletions(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func TestRenderTemplate(t *testing.T) {
	template := "Hello {name}, ticket {id} is {status}."
	vars := map[string]interface{}{"name": "Alex", "id": 42, "status": "open"}
	got := renderTemplate(template, vars)
	want := "Hello Alex, ticket 42 is open."
	if got != want {
		t.Fatalf("renderTemplate mismatch: got %q want %q", got, want)
	}

	noVars := renderTemplate("No vars", nil)
	if noVars != "No vars" {
		t.Fatalf("renderTemplate should return original when no vars: got %q", noVars)
	}
}

func TestProxyServer_NoAvailableCombinations(t *testing.T) {
	// Registry with no combinations
	registry := NewPromptRegistry()
	loadBalancer := NewLoadBalancer(registry, RoundRobinStrategy)
	proxy := NewProxyServer(registry, loadBalancer, ":0", nil, nil)

	// Register a provider
	testProvider := &TestModelProvider{
		models: map[string]models.Model{
			"gpt-4": {
				Name:         "gpt-4",
				Provider:     "OpenAI",
				CostPerToken: 0.00003,
				MaxTokens:    8192,
				Description:  "Most capable GPT-4 model",
			},
		},
	}
	proxy.RegisterProvider("test", testProvider)

	// Create a test HTTP request
	reqBody := `{
		"model": "gpt-4",
		"messages": [
			{
				"role": "user",
				"content": "Hello!"
			}
		]
	}`

	req, err := http.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	// Call the handler method directly
	proxy.handleChatCompletions(rr, req)

	// Should return 503 Service Unavailable when no combinations available
	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusServiceUnavailable)
	}
}

func TestTransformRequest(t *testing.T) {
	registry := NewPromptRegistry()
	loadBalancer := NewLoadBalancer(registry, RoundRobinStrategy)
	proxy := NewProxyServer(registry, loadBalancer, ":0", nil, nil)

	combination := &PromptCombination{
		ID:          "test-combo",
		Name:        "test-combo",
		Prompt:      "Be concise and helpful: ",
		Model:       "gpt-4",
		Provider:    "test",
		Weight:      10,
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
	}

	originalReq := Request{
		Model: "gpt-4",
		Messages: []Message{
			{Role: "user", Content: "How are you?"},
		},
	}

	transformedReq := proxy.transformRequest(originalReq, combination)

	// Check that the transformation added a system message
	if len(transformedReq.Messages) < 2 {
		t.Fatalf("Expected at least 2 messages (system + user), got %d", len(transformedReq.Messages))
	}

	// First message should be the system prompt
	systemMessage := transformedReq.Messages[0]
	if systemMessage.Role != "system" {
		t.Errorf("Expected first message to be system role, got %s", systemMessage.Role)
	}
	if systemMessage.Content != combination.Prompt {
		t.Errorf("Expected system message content to be combination prompt, got: %s", systemMessage.Content)
	}

	// Last message should be the original user message
	lastMessage := transformedReq.Messages[len(transformedReq.Messages)-1]
	if lastMessage.Role != "user" {
		t.Errorf("Expected last message to be user role, got %s", lastMessage.Role)
	}
	if lastMessage.Content != "How are you?" {
		t.Errorf("Expected original user content preserved, got: %s", lastMessage.Content)
	}

	if transformedReq.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got %s", transformedReq.Model)
	}
}

func TestProxyServer_StatusEndpoint(t *testing.T) {
	registry := NewPromptRegistry()
	loadBalancer := NewLoadBalancer(registry, RoundRobinStrategy)
	fakeStore := &fakeAnalyticsStore{
		promptStats: &analytics.PromptStats{
			PromptID:      "template-1",
			TemplateID:    "template-1",
			TotalRequests: 2,
			SuccessRate:   0.5,
		},
		systemStats: &analytics.SystemStats{
			TotalPrompts:  1,
			TotalRequests: 2,
			SuccessRate:   0.5,
		},
	}
	combination := &PromptCombination{
		ID:          "combo-1",
		Name:        "combo",
		TemplateID:  "template-1",
		Prompt:      "Hi",
		Model:       "gpt-4",
		Provider:    "test",
		Weight:      1,
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
	}
	registry.AddCombination(combination)

	proxy := NewProxyServer(registry, loadBalancer, ":0", nil, fakeStore)

	req := httptest.NewRequest("GET", "/status", nil)
	rr := httptest.NewRecorder()
	proxy.handleStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status handler returned %d", rr.Code)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal status response: %v", err)
	}
	if payload["count"].(float64) != 1 {
		t.Fatalf("expected 1 combination, got %v", payload["count"])
	}

	statsReq := httptest.NewRequest("GET", "/v1/system/analytics", nil)
	statsRec := httptest.NewRecorder()
	proxy.handleSystemAnalytics(statsRec, statsReq)
	if statsRec.Code != http.StatusOK {
		t.Fatalf("system analytics returned %d", statsRec.Code)
	}

	promptReq := httptest.NewRequest("GET", "/v1/prompts/template-1/stats", nil)
	promptRec := httptest.NewRecorder()
	proxy.handlePromptStats(promptRec, promptReq)
	if promptRec.Code != http.StatusOK {
		t.Fatalf("prompt stats returned %d", promptRec.Code)
	}
}

type fakeAnalyticsStore struct {
	promptStats *analytics.PromptStats
	systemStats *analytics.SystemStats
}

func (f *fakeAnalyticsStore) RecordInvocation(ctx context.Context, record *analytics.InvocationRecord) error {
	return nil
}

func (f *fakeAnalyticsStore) QueryPromptStats(ctx context.Context, filter analytics.StatsFilter) (*analytics.PromptStats, error) {
	return f.promptStats, nil
}

func (f *fakeAnalyticsStore) QuerySystemStats(ctx context.Context) (*analytics.SystemStats, error) {
	return f.systemStats, nil
}

func (f *fakeAnalyticsStore) Close() error { return nil }
