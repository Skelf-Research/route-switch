package models

import (
	"testing"
)

func TestMockModelProvider(t *testing.T) {
	provider := NewMockModelProvider()

	// Test Name
	if provider.Name() != "MockProvider" {
		t.Errorf("Expected provider name to be MockProvider, got %s", provider.Name())
	}

	// Test ListModels
	models, err := provider.ListModels()
	if err != nil {
		t.Errorf("Expected no error from ListModels, got %v", err)
	}
	if len(models) == 0 {
		t.Error("Expected ListModels to return at least one model")
	}

	// Test GetModel
	model, err := provider.GetModel("gpt-4")
	if err != nil {
		t.Errorf("Expected no error from GetModel, got %v", err)
	}
	if model.Name != "gpt-4" {
		t.Errorf("Expected model name to be gpt-4, got %s", model.Name)
	}

	// Test GetModel with non-existent model
	_, err = provider.GetModel("non-existent-model")
	if err == nil {
		t.Error("Expected error from GetModel with non-existent model, got nil")
	}

	// Test CallModel
	response, err := provider.CallModel("gpt-4", "test prompt")
	if err != nil {
		t.Errorf("Expected no error from CallModel, got %v", err)
	}
	if response == "" {
		t.Error("Expected CallModel to return a non-empty response")
	}

	// Test CallModel with non-existent model
	_, err = provider.CallModel("non-existent-model", "test prompt")
	if err == nil {
		t.Error("Expected error from CallModel with non-existent model, got nil")
	}
}