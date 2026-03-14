package core

import (
	"testing"

	"github.com/skelf-research/route-switch/internal/config"
	"github.com/skelf-research/route-switch/internal/models"
	"github.com/skelf-research/route-switch/internal/optimizer"
)

func TestServiceWithDependencies(t *testing.T) {
	// Create mock dependencies
	provider := models.NewMockModelProvider()
	evaluator := models.NewSimilarityEvaluationStrategy()

	// Create a mock optimizer for testing
	mockOptimizer := &mockOptimizer{}

	serviceConfig := &ServiceConfig{
		ModelProvider: provider,
		Evaluator:     evaluator,
		Optimizer:     mockOptimizer,
		Config:        &config.Config{},
	}

	service := NewService(serviceConfig)

	// Test that dependencies are properly initialized
	if service.modelProvider == nil {
		t.Error("Expected modelProvider to be initialized")
	}

	if service.optimizer == nil {
		t.Error("Expected optimizer to be initialized")
	}
}

func TestServiceOptimizePromptWithRealDependencies(t *testing.T) {
	// Create mock dependencies
	provider := models.NewMockModelProvider()
	evaluator := models.NewSimilarityEvaluationStrategy()

	// Create a mock optimizer for testing
	mockOptimizer := &mockOptimizer{}

	serviceConfig := &ServiceConfig{
		ModelProvider: provider,
		Evaluator:     evaluator,
		Optimizer:     mockOptimizer,
		Config:        &config.Config{},
	}

	service := NewService(serviceConfig)

	result, err := service.OptimizePrompt("Write a poem about technology", "gpt-4")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result.OriginalPrompt != "Write a poem about technology" {
		t.Errorf("Expected original prompt to match input, got %s", result.OriginalPrompt)
	}

	if result.OptimizedPrompt == "" {
		t.Error("Expected optimized prompt to not be empty")
	}

	if result.Model != "gpt-4" {
		t.Errorf("Expected model to be gpt-4, got %s", result.Model)
	}

	// Check that the improvement score is within valid range
	if result.ImprovementScore < 0 || result.ImprovementScore > 1 {
		t.Errorf("Expected improvement score between 0 and 1, got %f", result.ImprovementScore)
	}
}

func TestServiceFindBestModelWithRealDependencies(t *testing.T) {
	// Create mock dependencies
	provider := models.NewMockModelProvider()
	evaluator := models.NewSimilarityEvaluationStrategy()

	// Create a mock optimizer for testing
	mockOptimizer := &mockOptimizer{}

	serviceConfig := &ServiceConfig{
		ModelProvider: provider,
		Evaluator:     evaluator,
		Optimizer:     mockOptimizer,
		Config:        &config.Config{},
	}

	service := NewService(serviceConfig)

	result, err := service.FindBestModel("Write a poem about technology", "gpt-4")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result.OriginalPrompt != "Write a poem about technology" {
		t.Errorf("Expected original prompt to match input, got %s", result.OriginalPrompt)
	}

	if result.OptimizedPrompt == "" {
		t.Error("Expected optimized prompt to not be empty")
	}

	if result.Model == "" {
		t.Error("Expected model to not be empty")
	}

	// Check that the improvement score is within valid range
	if result.ImprovementScore < 0 || result.ImprovementScore > 1 {
		t.Errorf("Expected improvement score between 0 and 1, got %f", result.ImprovementScore)
	}
}

// mockOptimizer is a test double for the optimizer interface
type mockOptimizer struct{}

func (m *mockOptimizer) OptimizePrompt(basePrompt string, model models.Model, examples []models.Example) (*optimizer.PromptOptimizationResult, error) {
	return &optimizer.PromptOptimizationResult{
		Prompt: "Optimized: " + basePrompt,
		Score:  0.8,
		Metadata: map[string]interface{}{"test": true},
	}, nil
}

func (m *mockOptimizer) EvaluatePrompt(prompt string, model models.Model, examples []models.Example) (float64, error) {
	return 0.8, nil
}

func (m *mockOptimizer) GetBestCandidate() *optimizer.PromptOptimizationResult {
	return nil
}