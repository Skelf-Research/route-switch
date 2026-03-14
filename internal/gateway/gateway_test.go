package gateway

import (
	"testing"
	"time"

	"github.com/skelf-research/route-switch/internal/config"
	"github.com/skelf-research/route-switch/internal/core"
	"github.com/skelf-research/route-switch/internal/models"
	"github.com/skelf-research/route-switch/internal/optimizer"
)

// MockModelProvider for testing
type MockModelProvider struct {
	models map[string]models.Model
}

func (m *MockModelProvider) Name() string { return "MockProvider" }
func (m *MockModelProvider) ListModels() ([]models.Model, error) {
	return []models.Model{}, nil
}
func (m *MockModelProvider) GetModel(name string) (models.Model, error) {
	model, exists := m.models[name]
	if !exists {
		return models.Model{}, models.ErrNotFound
	}
	return model, nil
}
func (m *MockModelProvider) CallModel(modelName, prompt string) (string, error) {
	return "Mock response to: " + prompt, nil
}
func (m *MockModelProvider) EstimateCost(modelName string, inputTokens, outputTokens int) (float64, error) {
	return 0.01, nil
}
func (m *MockModelProvider) GetTokenCount(text string) (int, error) {
	return len([]rune(text)), nil
}
func (m *MockModelProvider) Initialize(config map[string]interface{}) error { return nil }
func (m *MockModelProvider) Close() error                                   { return nil }

func TestNewGateway(t *testing.T) {
	// Create mock service config
	mockProvider := &MockModelProvider{
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

	evaluator := models.NewSimilarityEvaluationStrategy()
	bayesianOpt, err := optimizer.NewGoptunaBayesianOptimizer(map[string]interface{}{"num_trials": 5})
	if err != nil {
		t.Fatalf("Failed to create Bayesian optimizer: %v", err)
	}

	opt := optimizer.NewMIPROv2(mockProvider, evaluator, bayesianOpt, config.MiproV2Config{
		NumCandidates:            5,
		MaxBootstrappedDemos:     3,
		MaxLabeledDemos:          2,
		NumTrials:                10,
		MinibatchSize:            5,
		MinibatchFullEvalSteps:   3,
		NumInstructionCandidates: 3,
	})

	serviceConfig := &core.ServiceConfig{
		ModelProvider: mockProvider,
		Evaluator:     evaluator,
		Optimizer:     opt,
		Config:        &config.Config{},
	}

	gatewayConfig := &GatewayConfig{
		Addr:                 ":8080",
		LoadBalancerStrategy: RoundRobinStrategy,
		OptimizationEnabled:  false,
		OptimizationInterval: 1 * time.Hour,
	}

	appConfig := &config.Config{
		Gateway: config.GatewayConfig{
			Addr:         ":8080",
			Strategy:     "round_robin",
			Combinations: []config.PromptCombinationConfig{},
			Optimization: config.OptimizationConfig{
				Enabled: false,
			},
		},
	}

	gw, err := NewGateway(serviceConfig, gatewayConfig, appConfig, nil, nil)
	if err != nil {
		t.Fatalf("NewGateway failed: %v", err)
	}

	if gw == nil {
		t.Fatal("NewGateway returned nil")
	}

	if gw.registry == nil {
		t.Error("Gateway registry is nil")
	}

	if gw.loadBalancer == nil {
		t.Error("Gateway loadBalancer is nil")
	}

	if gw.service == nil {
		t.Error("Gateway service is nil")
	}
}

func TestGateway_AddPromptCombination(t *testing.T) {
	// Set up gateway with minimal configuration
	mockProvider := &MockModelProvider{
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

	evaluator := models.NewSimilarityEvaluationStrategy()
	bayesianOpt, err := optimizer.NewGoptunaBayesianOptimizer(map[string]interface{}{"num_trials": 5})
	if err != nil {
		t.Fatalf("Failed to create Bayesian optimizer: %v", err)
	}

	opt := optimizer.NewMIPROv2(mockProvider, evaluator, bayesianOpt, config.MiproV2Config{
		NumCandidates:            5,
		MaxBootstrappedDemos:     3,
		MaxLabeledDemos:          2,
		NumTrials:                10,
		MinibatchSize:            5,
		MinibatchFullEvalSteps:   3,
		NumInstructionCandidates: 3,
	})

	serviceConfig := &core.ServiceConfig{
		ModelProvider: mockProvider,
		Evaluator:     evaluator,
		Optimizer:     opt,
		Config:        &config.Config{},
	}

	gatewayConfig := &GatewayConfig{
		Addr:                 ":8080",
		LoadBalancerStrategy: RoundRobinStrategy,
		OptimizationEnabled:  false,
		OptimizationInterval: 1 * time.Hour,
	}

	appConfig := &config.Config{
		Gateway: config.GatewayConfig{
			Addr:         ":8080",
			Strategy:     "round_robin",
			Combinations: []config.PromptCombinationConfig{},
			Optimization: config.OptimizationConfig{
				Enabled: false,
			},
		},
	}

	gw, err := NewGateway(serviceConfig, gatewayConfig, appConfig, nil, nil)
	if err != nil {
		t.Fatalf("NewGateway failed: %v", err)
	}

	// Add a prompt combination
	err = gw.AddPromptCombination("Test prompt", "gpt-4", "openai", "test-combo")
	if err != nil {
		t.Fatalf("AddPromptCombination failed: %v", err)
	}

	// Verify the combination was added
	combinations := gw.GetActiveCombinations()
	if len(combinations) != 1 {
		t.Errorf("Expected 1 combination, got %d", len(combinations))
	}

	if combinations[0].Name != "test-combo" {
		t.Errorf("Expected combination name 'test-combo', got '%s'", combinations[0].Name)
	}
}

func TestGateway_UpdateCombinationWeight(t *testing.T) {
	// Set up minimal gateway
	mockProvider := &MockModelProvider{
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

	evaluator := models.NewSimilarityEvaluationStrategy()
	bayesianOpt, err := optimizer.NewGoptunaBayesianOptimizer(map[string]interface{}{"num_trials": 5})
	if err != nil {
		t.Fatalf("Failed to create Bayesian optimizer: %v", err)
	}

	opt := optimizer.NewMIPROv2(mockProvider, evaluator, bayesianOpt, config.MiproV2Config{
		NumCandidates:            5,
		MaxBootstrappedDemos:     3,
		MaxLabeledDemos:          2,
		NumTrials:                10,
		MinibatchSize:            5,
		MinibatchFullEvalSteps:   3,
		NumInstructionCandidates: 3,
	})

	serviceConfig := &core.ServiceConfig{
		ModelProvider: mockProvider,
		Evaluator:     evaluator,
		Optimizer:     opt,
		Config:        &config.Config{},
	}

	gatewayConfig := &GatewayConfig{
		Addr:                 ":8080",
		LoadBalancerStrategy: RoundRobinStrategy,
		OptimizationEnabled:  false,
		OptimizationInterval: 1 * time.Hour,
	}

	appConfig := &config.Config{
		Gateway: config.GatewayConfig{
			Addr:         ":8080",
			Strategy:     "round_robin",
			Combinations: []config.PromptCombinationConfig{},
			Optimization: config.OptimizationConfig{
				Enabled: false,
			},
		},
	}

	gw, err := NewGateway(serviceConfig, gatewayConfig, appConfig, nil, nil)
	if err != nil {
		t.Fatalf("NewGateway failed: %v", err)
	}

	// Add a prompt combination
	err = gw.AddPromptCombination("Test prompt", "gpt-4", "openai", "test-combo")
	if err != nil {
		t.Fatalf("AddPromptCombination failed: %v", err)
	}

	// Get the combination ID
	combinations := gw.GetActiveCombinations()
	if len(combinations) == 0 {
		t.Fatal("No combinations found")
	}
	id := combinations[0].ID

	// Update the weight
	err = gw.UpdateCombinationWeight(id, 50)
	if err != nil {
		t.Fatalf("UpdateCombinationWeight failed: %v", err)
	}

	// Verify the weight was updated
	combination, exists := gw.registry.GetCombination(id)
	if !exists {
		t.Fatal("Combination not found after weight update")
	}

	if combination.Weight != 50 {
		t.Errorf("Expected weight 50, got %d", combination.Weight)
	}
}

func TestGateway_LoadCombinationsFromConfig(t *testing.T) {
	// Set up minimal gateway
	mockProvider := &MockModelProvider{
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

	evaluator := models.NewSimilarityEvaluationStrategy()
	bayesianOpt, err := optimizer.NewGoptunaBayesianOptimizer(map[string]interface{}{"num_trials": 5})
	if err != nil {
		t.Fatalf("Failed to create Bayesian optimizer: %v", err)
	}

	opt := optimizer.NewMIPROv2(mockProvider, evaluator, bayesianOpt, config.MiproV2Config{
		NumCandidates:            5,
		MaxBootstrappedDemos:     3,
		MaxLabeledDemos:          2,
		NumTrials:                10,
		MinibatchSize:            5,
		MinibatchFullEvalSteps:   3,
		NumInstructionCandidates: 3,
	})

	serviceConfig := &core.ServiceConfig{
		ModelProvider: mockProvider,
		Evaluator:     evaluator,
		Optimizer:     opt,
		Config:        &config.Config{},
	}

	gatewayConfig := &GatewayConfig{
		Addr:                 ":8080",
		LoadBalancerStrategy: RoundRobinStrategy,
		OptimizationEnabled:  false,
		OptimizationInterval: 1 * time.Hour,
	}

	// Create app config with predefined combinations
	appConfig := &config.Config{
		Gateway: config.GatewayConfig{
			Addr:     ":8080",
			Strategy: "round_robin",
			Combinations: []config.PromptCombinationConfig{
				{
					ID:        "test-1",
					Name:      "test-combo-1",
					Prompt:    "Test prompt 1",
					Model:     "gpt-4",
					Provider:  "openai",
					IsPrimary: true,
					Weight:    70,
					Enabled:   true,
					Metadata:  map[string]interface{}{"test": true},
				},
				{
					ID:        "test-2",
					Name:      "test-combo-2",
					Prompt:    "Test prompt 2",
					Model:     "gpt-4",
					Provider:  "openai",
					IsPrimary: false,
					Weight:    30,
					Enabled:   true,
					Metadata:  map[string]interface{}{"test": true},
				},
			},
			Optimization: config.OptimizationConfig{
				Enabled: false,
			},
		},
	}

	gw, err := NewGateway(serviceConfig, gatewayConfig, appConfig, nil, nil)
	if err != nil {
		t.Fatalf("NewGateway failed: %v", err)
	}

	// Check that combinations were loaded
	combinations := gw.GetActiveCombinations()
	if len(combinations) != 2 {
		t.Errorf("Expected 2 combinations from config, got %d", len(combinations))
	}

	// Check that primary combination exists
	primaryFound := false
	for _, combo := range combinations {
		if combo.Name == "test-combo-1" { // This was marked as primary
			primaryFound = true
			break
		}
	}

	if !primaryFound {
		t.Error("Primary combination not found in registry")
	}
}
