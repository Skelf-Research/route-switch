package utils

import (
	"testing"
)

func TestNewCostCalculator(t *testing.T) {
	calculator := NewCostCalculator()
	if calculator == nil {
		t.Error("Expected NewCostCalculator to return a non-nil calculator")
	}
}

func TestCalculateCost(t *testing.T) {
	calculator := NewCostCalculator()
	
	// Test with sample values
	cost, err := calculator.CalculateCost("gpt-4", 100, 200)
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if cost <= 0 {
		t.Errorf("Expected cost to be greater than 0, got %f", cost)
	}
}

func TestFindCheapestModel(t *testing.T) {
	calculator := NewCostCalculator()
	
	models := []string{"gpt-4", "gpt-3.5-turbo", "claude-2"}
	prompt := "Write a poem about technology"
	
	model, cost, err := calculator.FindCheapestModel(models, prompt)
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if model == "" {
		t.Error("Expected model name to not be empty")
	}
	
	if cost <= 0 {
		t.Errorf("Expected cost to be greater than 0, got %f", cost)
	}
}