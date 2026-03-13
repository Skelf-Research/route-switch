package core

import (
	"testing"
)

func TestNewService(t *testing.T) {
	service := NewService()
	if service == nil {
		t.Error("Expected NewService to return a non-nil service")
	}
}

func TestOptimizePrompt(t *testing.T) {
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
}

func TestFindBestModel(t *testing.T) {
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
}