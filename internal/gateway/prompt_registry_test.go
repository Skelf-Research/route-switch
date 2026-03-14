package gateway

import (
	"testing"
	"time"
)

func TestPromptRegistry_AddCombination(t *testing.T) {
	registry := NewPromptRegistry()

	combination := &PromptCombination{
		ID:          "test-id",
		Name:        "test-name",
		Prompt:      "test prompt",
		Model:       "gpt-4",
		Provider:    "openai",
		Weight:      10,
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
		Metadata:    map[string]interface{}{"test": true},
	}

	err := registry.AddCombination(combination)
	if err != nil {
		t.Errorf("AddCombination failed: %v", err)
	}

	retrieved, exists := registry.GetCombination("test-id")
	if !exists {
		t.Error("Combination not found after adding")
	}

	if retrieved.Name != "test-name" {
		t.Errorf("Expected name 'test-name', got '%s'", retrieved.Name)
	}
}

func TestPromptRegistry_GetCombinationByName(t *testing.T) {
	registry := NewPromptRegistry()

	combination := &PromptCombination{
		ID:          "test-id",
		Name:        "test-name",
		Prompt:      "test prompt",
		Model:       "gpt-4",
		Provider:    "openai",
		Weight:      10,
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
		Metadata:    map[string]interface{}{"test": true},
	}

	registry.AddCombination(combination)

	retrieved, exists := registry.GetCombinationByName("test-name")
	if !exists {
		t.Error("Combination not found by name")
	}

	if retrieved.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", retrieved.ID)
	}
}

func TestPromptRegistry_GetAllCombinations(t *testing.T) {
	registry := NewPromptRegistry()

	combination1 := &PromptCombination{
		ID:          "test-id-1",
		Name:        "test-name-1",
		Prompt:      "test prompt 1",
		Model:       "gpt-4",
		Provider:    "openai",
		Weight:      10,
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
		Metadata:    map[string]interface{}{"test": true},
	}

	combination2 := &PromptCombination{
		ID:          "test-id-2",
		Name:        "test-name-2",
		Prompt:      "test prompt 2",
		Model:       "gpt-3.5-turbo",
		Provider:    "openai",
		Weight:      5,
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
		Metadata:    map[string]interface{}{"test": true},
	}

	registry.AddCombination(combination1)
	registry.AddCombination(combination2)

	combinations := registry.GetAllCombinations()

	if len(combinations) != 2 {
		t.Errorf("Expected 2 combinations, got %d", len(combinations))
	}
}

func TestPromptRegistry_UpdatePerformance(t *testing.T) {
	registry := NewPromptRegistry()

	combination := &PromptCombination{
		ID:          "test-id",
		Name:        "test-name",
		Prompt:      "test prompt",
		Model:       "gpt-4",
		Provider:    "openai",
		Weight:      10,
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
		Metadata:    map[string]interface{}{"test": true},
	}

	registry.AddCombination(combination)

	// Update performance
	err := registry.UpdatePerformance("test-id", 1*time.Second, true, 0.01)
	if err != nil {
		t.Errorf("UpdatePerformance failed: %v", err)
	}

	updated, exists := registry.GetCombination("test-id")
	if !exists {
		t.Fatal("Combination not found after updating performance")
	}

	if updated.Performance.ResponseTimeAvg <= 0 {
		t.Error("Response time average not updated")
	}

	if updated.Performance.SuccessRate != 1.0 {
		t.Errorf("Expected success rate 1.0, got %f", updated.Performance.SuccessRate)
	}

	if updated.Performance.CostPerRequest <= 0 {
		t.Error("Cost per request not updated")
	}
}

func TestPromptRegistry_GetActiveCombinations(t *testing.T) {
	registry := NewPromptRegistry()

	// Active combination
	activeCombination := &PromptCombination{
		ID:          "active-id",
		Name:        "active-name",
		Prompt:      "test prompt",
		Model:       "gpt-4",
		Provider:    "openai",
		Weight:      10, // Weight > 0 means active
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
		Metadata:    map[string]interface{}{"test": true},
	}

	// Inactive combination
	inactiveCombination := &PromptCombination{
		ID:          "inactive-id",
		Name:        "inactive-name",
		Prompt:      "test prompt",
		Model:       "gpt-3.5-turbo",
		Provider:    "openai",
		Weight:      0, // Weight = 0 means inactive
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
		Metadata:    map[string]interface{}{"test": true},
	}

	registry.AddCombination(activeCombination)
	registry.AddCombination(inactiveCombination)

	activeList := registry.GetActiveCombinations()

	if len(activeList) != 1 {
		t.Errorf("Expected 1 active combination, got %d", len(activeList))
	}

	if activeList[0].ID != "active-id" {
		t.Errorf("Expected active combination with ID 'active-id', got '%s'", activeList[0].ID)
	}
}

func TestPromptRegistry_GetCombinationsByProvider(t *testing.T) {
	registry := NewPromptRegistry()

	combination1 := &PromptCombination{
		ID:          "test-id-1",
		Name:        "test-name-1",
		Prompt:      "test prompt 1",
		Model:       "gpt-4",
		Provider:    "openai",
		Weight:      10,
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
		Metadata:    map[string]interface{}{"test": true},
	}

	combination2 := &PromptCombination{
		ID:          "test-id-2",
		Name:        "test-name-2",
		Prompt:      "test prompt 2",
		Model:       "claude-2",
		Provider:    "anthropic", // Different provider
		Weight:      5,
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
		Metadata:    map[string]interface{}{"test": true},
	}

	registry.AddCombination(combination1)
	registry.AddCombination(combination2)

	openaiCombinations := registry.GetCombinationsByProvider("openai")

	if len(openaiCombinations) != 1 {
		t.Errorf("Expected 1 OpenAI combination, got %d", len(openaiCombinations))
	}

	if openaiCombinations[0].ID != "test-id-1" {
		t.Errorf("Expected OpenAI combination with ID 'test-id-1', got '%s'", openaiCombinations[0].ID)
	}
}
