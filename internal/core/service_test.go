package core

import (
	"testing"

	"github.com/skelf-research/route-switch/internal/config"
	"github.com/skelf-research/route-switch/internal/models"
)

func TestNewService(t *testing.T) {
	// Create mock dependencies
	provider := models.NewMockModelProvider()
	evaluator := models.NewSimilarityEvaluationStrategy()

	// Create a mock optimizer (using a minimal implementation for testing)
	mockOptimizer := &mockOptimizer{}

	serviceConfig := &ServiceConfig{
		ModelProvider: provider,
		Evaluator:     evaluator,
		Optimizer:     mockOptimizer,
		Config:        &config.Config{},
	}

	service := NewService(serviceConfig)
	if service == nil {
		t.Error("Expected NewService to return a non-nil service")
	}
}

func TestOptimizePrompt(t *testing.T) {
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
}

func TestFindBestModel(t *testing.T) {
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

	// For the mock implementation, we don't expect a specific cost pattern
}

