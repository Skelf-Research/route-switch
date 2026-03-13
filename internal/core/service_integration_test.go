package core

import (
	"testing"
)

func TestServiceWithDependencies(t *testing.T) {
	service := NewService()
	
	// Test that dependencies are properly initialized
	if service.modelProvider == nil {
		t.Error("Expected modelProvider to be initialized")
	}
	
	if service.optimizer == nil {
		t.Error("Expected optimizer to be initialized")
	}
	
	if service.costCalc == nil {
		t.Error("Expected costCalc to be initialized")
	}
}

func TestServiceOptimizePromptWithRealDependencies(t *testing.T) {
	service := NewService()
	
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
	service := NewService()
	
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
	
	if result.Cost <= 0 {
		t.Errorf("Expected cost to be greater than 0, got %f", result.Cost)
	}
	
	// Check that the improvement score is within valid range
	if result.ImprovementScore < 0 || result.ImprovementScore > 1 {
		t.Errorf("Expected improvement score between 0 and 1, got %f", result.ImprovementScore)
	}
}