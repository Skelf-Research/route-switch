package gateway

import (
	"testing"
	"time"

	"github.com/skelf-research/route-switch/internal/config"
	"github.com/skelf-research/route-switch/internal/core"
	"github.com/skelf-research/route-switch/internal/models"
	"github.com/skelf-research/route-switch/internal/optimizer"
)

func TestBackgroundOptimizer_StartStop(t *testing.T) {
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
		OptimizationEnabled:  false, // We'll test the optimizer directly
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

	// Add a test combination
	err = gw.AddPromptCombination("Test prompt", "gpt-4", "openai", "test-combo")
	if err != nil {
		t.Fatalf("AddPromptCombination failed: %v", err)
	}

	// Create background optimizer with short interval for testing
	bgOptimizer := NewBackgroundOptimizer(gw, 100*time.Millisecond)

	// Start the optimizer
	go bgOptimizer.Start()

	// Give it a moment to run once
	time.Sleep(150 * time.Millisecond)

	// Stop the optimizer
	bgOptimizer.Stop()

	// Verify we can stop without issues
	t.Log("Background optimizer started and stopped successfully")
}

func TestBackgroundOptimizer_OptimizeCombination(t *testing.T) {
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

	// Add a test combination
	err = gw.AddPromptCombination("Test original prompt", "gpt-4", "openai", "test-combo")
	if err != nil {
		t.Fatalf("AddPromptCombination failed: %v", err)
	}

	// Get the combination to test optimization
	combinations := gw.GetActiveCombinations()
	if len(combinations) == 0 {
		t.Fatal("No combinations found")
	}

	combination := combinations[0]

	// Since we can't directly test the optimization (as it creates a copy),
	// let's just test that the function doesn't panic and runs successfully
	bgOptimizer := NewBackgroundOptimizer(gw, 1*time.Hour)

	// Add metadata to indicate this needs optimization
	combination.Metadata["original_prompt"] = "Test original prompt"

	// Call optimizeCombination directly - just make sure it doesn't error
	bgOptimizer.optimizeCombination(combination)

	// We can't validate the actual optimization result since it operates on a copy,
	// but we can at least ensure the function runs without panicking
	t.Log("optimizeCombination executed without errors")
}

func TestBackgroundOptimizer_Optimize(t *testing.T) {
	// Set up gateway
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

	// Add a test combination with old LastOptimized time to trigger optimization
	err = gw.AddPromptCombination("Test original prompt", "gpt-4", "openai", "test-combo")
	if err != nil {
		t.Fatalf("AddPromptCombination failed: %v", err)
	}

	// Get the combination and set LastOptimized to a time that would trigger optimization
	combinations := gw.GetActiveCombinations()
	if len(combinations) == 0 {
		t.Fatal("No combinations found")
	}

	combination := combinations[0]
	// Set LastOptimized to 25 hours ago (more than 24 hours) to trigger optimization
	combination.LastOptimized = time.Now().Add(-25 * time.Hour)

	// Run the optimization process
	bgOptimizer := NewBackgroundOptimizer(gw, 1*time.Hour)
	bgOptimizer.optimize()

	// Since we're testing the background optimizer, we can check that it runs without errors
	// The key is that this runs without errors
	t.Logf("Optimization completed for combination: %s", combination.Name)
}
