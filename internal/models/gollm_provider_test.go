package models

import (
	"testing"
)

func TestNewGollmProvider(t *testing.T) {
	provider := NewGollmProvider()
	if provider == nil {
		t.Fatal("Expected NewGollmProvider to return a non-nil provider")
	}

	if provider.llms == nil {
		t.Error("Expected llms map to be initialized")
	}

	if provider.costCalculator == nil {
		t.Error("Expected costCalculator to be initialized")
	}
}

func TestGollmProviderName(t *testing.T) {
	provider := NewGollmProvider()
	name := provider.Name()

	if name != "GollmProvider" {
		t.Errorf("Expected provider name 'GollmProvider', got '%s'", name)
	}
}

func TestGollmProviderListModels(t *testing.T) {
	provider := NewGollmProvider()
	models, err := provider.ListModels()

	if err != nil {
		t.Errorf("Expected no error from ListModels, got: %v", err)
	}

	// Currently returns empty list
	if models == nil {
		t.Error("Expected non-nil models slice")
	}
}

func TestGollmProviderInitialize(t *testing.T) {
	provider := NewGollmProvider()

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name:    "Valid config",
			config:  map[string]interface{}{"api_key": "test-key"},
			wantErr: false,
		},
		{
			name:    "Empty config",
			config:  map[string]interface{}{},
			wantErr: false,
		},
		{
			name:    "Nil config",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.Initialize(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGollmProviderClose(t *testing.T) {
	provider := NewGollmProvider()
	provider.llms["test-model"] = nil // Add a dummy entry

	err := provider.Close()
	if err != nil {
		t.Errorf("Expected no error from Close, got: %v", err)
	}

	if len(provider.llms) != 0 {
		t.Error("Expected llms map to be cleared after Close")
	}
}

func TestGollmProviderGetTokenCount(t *testing.T) {
	provider := NewGollmProvider()

	tests := []struct {
		name        string
		text        string
		expectedMin int
		expectedMax int
	}{
		{
			name:        "Empty text",
			text:        "",
			expectedMin: 0,
			expectedMax: 0,
		},
		{
			name:        "Short text",
			text:        "Hello",
			expectedMin: 0,
			expectedMax: 2,
		},
		{
			name:        "Medium text",
			text:        "This is a test sentence for token counting.",
			expectedMin: 8,
			expectedMax: 15,
		},
		{
			name:        "Long text",
			text:        "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
			expectedMin: 20,
			expectedMax: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := provider.GetTokenCount(tt.text)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if count < tt.expectedMin || count > tt.expectedMax {
				t.Errorf("Token count %d not in expected range [%d, %d] for text length %d",
					count, tt.expectedMin, tt.expectedMax, len(tt.text))
			}
		})
	}
}

func TestGollmProviderEstimateCost(t *testing.T) {
	provider := NewGollmProvider()
	provider.Initialize(map[string]interface{}{})

	tests := []struct {
		name         string
		model        string
		inputTokens  int
		outputTokens int
		wantErr      bool
		checkCost    func(cost float64) bool
	}{
		{
			name:         "GPT-4 cost estimation",
			model:        "gpt-4",
			inputTokens:  1000,
			outputTokens: 500,
			wantErr:      false,
			checkCost: func(cost float64) bool {
				return cost > 0
			},
		},
		{
			name:         "Zero tokens",
			model:        "gpt-4",
			inputTokens:  0,
			outputTokens: 0,
			wantErr:      false,
			checkCost: func(cost float64) bool {
				return cost == 0
			},
		},
		{
			name:         "Empty model name",
			model:        "",
			inputTokens:  100,
			outputTokens: 100,
			wantErr:      true,
			checkCost:    nil,
		},
		{
			name:         "Negative input tokens",
			model:        "gpt-4",
			inputTokens:  -100,
			outputTokens: 100,
			wantErr:      true,
			checkCost:    nil,
		},
		{
			name:         "Negative output tokens",
			model:        "gpt-4",
			inputTokens:  100,
			outputTokens: -100,
			wantErr:      true,
			checkCost:    nil,
		},
		{
			name:         "Claude model",
			model:        "claude-3-opus-20240229",
			inputTokens:  1000,
			outputTokens: 500,
			wantErr:      false,
			checkCost: func(cost float64) bool {
				return cost > 0
			},
		},
		{
			name:         "Ollama model (free)",
			model:        "ollama/llama3",
			inputTokens:  1000,
			outputTokens: 500,
			wantErr:      false,
			checkCost: func(cost float64) bool {
				return cost == 0 // Ollama is free
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, err := provider.EstimateCost(tt.model, tt.inputTokens, tt.outputTokens)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.checkCost != nil && !tt.checkCost(cost) {
				t.Errorf("Cost %f did not pass validation for model %s", cost, tt.model)
			}
		})
	}
}

func TestGollmProviderDetectProviderType(t *testing.T) {
	provider := NewGollmProvider()
	// Clear cost calculator to use deterministic heuristic-based detection
	// This avoids flaky tests due to varying LiteLLM API data
	provider.costCalculator = nil

	// Test that provider detection works for common models using heuristics
	tests := []struct {
		modelName string
		expected  string
	}{
		{"gpt-4", "openai"},
		{"gpt-3.5-turbo", "openai"},
		{"o1-preview", "openai"},
		{"claude-3-opus", "anthropic"},
		{"claude-3-5-sonnet", "anthropic"},
		{"gemini-pro", "google"},
		{"gemini-1.5-pro", "google"},
		{"command-r", "cohere"},
		{"llama3", "ollama"},
		{"codellama", "ollama"},
		{"huggingface-model", "huggingface"},
		{"hf-model", "huggingface"},
	}

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			result := provider.detectProviderType(tt.modelName)

			if result != tt.expected {
				t.Errorf("detectProviderType(%q) = %q, want %q",
					tt.modelName, result, tt.expected)
			}
		})
	}
}

func TestGollmProviderDetectProviderTypeFallback(t *testing.T) {
	// Test the fallback heuristic when LiteLLM data is not available
	// These tests verify the heuristic fallback logic
	heuristicTests := []struct {
		modelName string
		expected  string
	}{
		{"gpt-4-custom-suffix", "openai"},          // Contains "gpt-"
		{"o1-mini-custom", "openai"},               // Contains "o1-"
		{"claude-instant-v1", "anthropic"},         // Contains "claude-"
		{"gemini-flash-exp", "google"},             // Contains "gemini-"
		{"command-light-custom", "cohere"},         // Contains "command-"
		{"unknown-random-model", "openai"},         // Default fallback
	}

	for _, tt := range heuristicTests {
		t.Run(tt.modelName+"_heuristic", func(t *testing.T) {
			// Use a provider with no pricing data to force heuristic
			freshProvider := NewGollmProvider()
			// Clear any cached pricing to force heuristic path
			freshProvider.costCalculator = nil

			// Since costCalculator is nil, it will use the heuristic
			result := freshProvider.detectProviderType(tt.modelName)

			if result != tt.expected {
				t.Errorf("detectProviderType heuristic(%q) = %q, want %q",
					tt.modelName, result, tt.expected)
			}
		})
	}
}

func TestGollmProviderDetectProviderTypeHeuristic(t *testing.T) {
	provider := NewGollmProvider()

	// Test the heuristic method directly
	tests := []struct {
		modelName string
		expected  string
	}{
		{"gpt-4", "openai"},
		{"gpt-3.5-turbo", "openai"},
		{"o1-preview", "openai"},
		{"claude-3-opus", "anthropic"},
		{"claude-2", "anthropic"},
		{"gemini-pro", "google"},
		{"gemini-1.5-flash", "google"},
		{"command-r", "cohere"},
		{"command-light", "cohere"},
		{"llama3", "ollama"},
		{"llama-70b", "ollama"},
		{"mistral-7b", "ollama"},
		{"mixtral-8x7b", "ollama"},
		{"codellama", "ollama"},
		{"phi-3", "ollama"},
		{"gemma-7b", "ollama"},
		{"qwen-72b", "ollama"},
		{"deepseek-v2", "ollama"},
		{"huggingface-model", "huggingface"},
		{"hf-model", "huggingface"},
		{"unknown-model", "openai"}, // Default fallback
	}

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			result := provider.detectProviderTypeHeuristic(tt.modelName)
			if result != tt.expected {
				t.Errorf("detectProviderTypeHeuristic(%q) = %q, want %q",
					tt.modelName, result, tt.expected)
			}
		})
	}
}

func TestGollmProviderGetAPIKey(t *testing.T) {
	provider := NewGollmProvider()

	tests := []struct {
		name        string
		config      map[string]interface{}
		primaryKey  string
		fallbackKey string
		wantKey     string
		wantFound   bool
	}{
		{
			name:        "Primary key exists",
			config:      map[string]interface{}{"openai_api_key": "primary-key", "api_key": "fallback-key"},
			primaryKey:  "openai_api_key",
			fallbackKey: "api_key",
			wantKey:     "primary-key",
			wantFound:   true,
		},
		{
			name:        "Only fallback key exists",
			config:      map[string]interface{}{"api_key": "fallback-key"},
			primaryKey:  "openai_api_key",
			fallbackKey: "api_key",
			wantKey:     "fallback-key",
			wantFound:   true,
		},
		{
			name:        "No key exists",
			config:      map[string]interface{}{},
			primaryKey:  "openai_api_key",
			fallbackKey: "api_key",
			wantKey:     "",
			wantFound:   false,
		},
		{
			name:        "Nil config",
			config:      nil,
			primaryKey:  "openai_api_key",
			fallbackKey: "api_key",
			wantKey:     "",
			wantFound:   false,
		},
		{
			name:        "Empty primary key value",
			config:      map[string]interface{}{"openai_api_key": "", "api_key": "fallback-key"},
			primaryKey:  "openai_api_key",
			fallbackKey: "api_key",
			wantKey:     "fallback-key",
			wantFound:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, found := provider.getAPIKey(tt.config, tt.primaryKey, tt.fallbackKey)

			if found != tt.wantFound {
				t.Errorf("getAPIKey() found = %v, want %v", found, tt.wantFound)
			}

			if key != tt.wantKey {
				t.Errorf("getAPIKey() key = %q, want %q", key, tt.wantKey)
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s          string
		substrings []string
		expected   bool
	}{
		{"gpt-4-turbo", []string{"gpt-"}, true},
		{"GPT-4-TURBO", []string{"gpt-"}, true}, // Case insensitive
		{"claude-3", []string{"gpt-", "claude-"}, true},
		{"mistral-7b", []string{"gpt-", "claude-"}, false},
		{"", []string{"test"}, false},
		{"test", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			result := containsAny(tt.s, tt.substrings)
			if result != tt.expected {
				t.Errorf("containsAny(%q, %v) = %v, want %v", tt.s, tt.substrings, result, tt.expected)
			}
		})
	}
}

func TestGollmProviderGetCostCalculator(t *testing.T) {
	provider := NewGollmProvider()

	calculator := provider.GetCostCalculator()
	if calculator == nil {
		t.Error("Expected non-nil cost calculator")
	}
}

func TestGollmProviderGetCostBreakdown(t *testing.T) {
	provider := NewGollmProvider()

	breakdown, err := provider.GetCostBreakdown("gpt-4", 1000, 500)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if breakdown == nil {
		t.Fatal("Expected non-nil breakdown")
	}

	if breakdown.InputTokens != 1000 {
		t.Errorf("Expected input tokens 1000, got %d", breakdown.InputTokens)
	}

	if breakdown.OutputTokens != 500 {
		t.Errorf("Expected output tokens 500, got %d", breakdown.OutputTokens)
	}

	if breakdown.TotalCost <= 0 {
		t.Error("Expected positive total cost")
	}

	if breakdown.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", breakdown.Model)
	}
}

func TestGollmProviderGetMaxTokens(t *testing.T) {
	provider := NewGollmProvider()

	tests := []struct {
		model        string
		minInputMax  int
		minOutputMax int
		wantErr      bool
	}{
		{"gpt-4", 8000, 4000, false},
		{"gpt-4o", 100000, 10000, false},
		{"claude-3-opus-20240229", 100000, 4000, false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			inputMax, outputMax, err := provider.GetMaxTokens(tt.model)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if inputMax < tt.minInputMax {
				t.Errorf("Expected input max >= %d, got %d", tt.minInputMax, inputMax)
			}

			if outputMax < tt.minOutputMax {
				t.Errorf("Expected output max >= %d, got %d", tt.minOutputMax, outputMax)
			}
		})
	}
}

func TestGollmProviderCallModelValidation(t *testing.T) {
	provider := NewGollmProvider()
	provider.Initialize(map[string]interface{}{})

	tests := []struct {
		name      string
		modelName string
		prompt    string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "Empty model name",
			modelName: "",
			prompt:    "test prompt",
			wantErr:   true,
			errMsg:    "model name cannot be empty",
		},
		{
			name:      "Empty prompt",
			modelName: "gpt-4",
			prompt:    "",
			wantErr:   true,
			errMsg:    "prompt cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := provider.CallModel(tt.modelName, tt.prompt)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			}
		})
	}
}

func TestGollmProviderGetModelValidation(t *testing.T) {
	provider := NewGollmProvider()
	provider.Initialize(map[string]interface{}{})

	_, err := provider.GetModel("")
	if err == nil {
		t.Error("Expected error for empty model name")
	}
}

func TestGollmProviderCreateLLMInstanceValidation(t *testing.T) {
	provider := NewGollmProvider()

	tests := []struct {
		name         string
		providerType string
		modelName    string
		wantErr      bool
	}{
		{
			name:         "Empty provider type",
			providerType: "",
			modelName:    "gpt-4",
			wantErr:      true,
		},
		{
			name:         "Empty model name",
			providerType: "openai",
			modelName:    "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := provider.createLLMInstance(tt.providerType, tt.modelName, nil)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			}
		})
	}
}

func TestGollmProviderCostComparisonBetweenModels(t *testing.T) {
	provider := NewGollmProvider()
	provider.Initialize(map[string]interface{}{})

	// GPT-4 should be more expensive than GPT-3.5-turbo
	gpt4Cost, err := provider.EstimateCost("gpt-4", 1000, 500)
	if err != nil {
		t.Fatalf("Failed to get GPT-4 cost: %v", err)
	}

	gpt35Cost, err := provider.EstimateCost("gpt-3.5-turbo", 1000, 500)
	if err != nil {
		t.Fatalf("Failed to get GPT-3.5-turbo cost: %v", err)
	}

	if gpt4Cost <= gpt35Cost {
		t.Errorf("Expected GPT-4 cost (%f) to be greater than GPT-3.5-turbo cost (%f)",
			gpt4Cost, gpt35Cost)
	}
}

func TestGollmProviderDifferentProviderConfigs(t *testing.T) {
	tests := []struct {
		name         string
		providerType string
		config       map[string]interface{}
	}{
		{
			name:         "OpenAI config",
			providerType: "openai",
			config: map[string]interface{}{
				"api_key": "test-openai-key",
			},
		},
		{
			name:         "Anthropic config",
			providerType: "anthropic",
			config: map[string]interface{}{
				"anthropic_api_key": "test-anthropic-key",
			},
		},
		{
			name:         "Ollama config with host",
			providerType: "ollama",
			config: map[string]interface{}{
				"ollama_host": "http://localhost:11434",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewGollmProvider()
			err := provider.Initialize(tt.config)
			// We don't expect errors for configuration, only for actual API calls
			if err != nil {
				t.Logf("Note: Initialize returned error (expected without real credentials): %v", err)
			}
		})
	}
}