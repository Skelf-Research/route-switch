package models

import (
	"fmt"
)

// MockModelProvider implements ModelProvider interface for testing
type MockModelProvider struct {
	models map[string]Model
	config map[string]interface{}
}

// NewMockModelProvider creates a new mock model provider
func NewMockModelProvider() *MockModelProvider {
	// Initialize with some sample models
	models := map[string]Model{
		"gpt-4": {
			Name:         "gpt-4",
			Provider:     "OpenAI",
			CostPerToken: 0.00003,
			MaxTokens:    8192,
			Description:  "Most capable GPT-4 model",
		},
		"gpt-3.5-turbo": {
			Name:         "gpt-3.5-turbo",
			Provider:     "OpenAI",
			CostPerToken: 0.000002,
			MaxTokens:    4096,
			Description:  "Fast and cheap GPT-3.5 model",
		},
		"claude-2": {
			Name:         "claude-2",
			Provider:     "Anthropic",
			CostPerToken: 0.000015,
			MaxTokens:    100000,
			Description:  "Anthropic's Claude 2 model",
		},
	}

	return &MockModelProvider{
		models: models,
		config: make(map[string]interface{}),
	}
}

// Name returns the name of the provider
func (m *MockModelProvider) Name() string {
	return "MockProvider"
}

// ListModels returns all available models
func (m *MockModelProvider) ListModels() ([]Model, error) {
	models := make([]Model, 0, len(m.models))
	for _, model := range m.models {
		models = append(models, model)
	}
	return models, nil
}

// GetModel returns a specific model by name
func (m *MockModelProvider) GetModel(name string) (Model, error) {
	model, exists := m.models[name]
	if !exists {
		return Model{}, fmt.Errorf("model %s not found", name)
	}
	return model, nil
}

// CallModel simulates calling a model with a prompt
func (m *MockModelProvider) CallModel(modelName, prompt string) (string, error) {
	_, exists := m.models[modelName]
	if !exists {
		return "", fmt.Errorf("model %s not found", modelName)
	}

	// In a real implementation, this would call the actual model API
	// For now, we'll just return a mock response
	return fmt.Sprintf("Response from %s to prompt: %s", modelName, prompt), nil
}

// EstimateCost estimates the cost of using a model for a given number of tokens
func (m *MockModelProvider) EstimateCost(modelName string, inputTokens, outputTokens int) (float64, error) {
	model, exists := m.models[modelName]
	if !exists {
		return 0, fmt.Errorf("model %s not found", modelName)
	}

	// Calculate estimated cost
	inputCost := float64(inputTokens) * model.CostPerToken
	outputCost := float64(outputTokens) * model.CostPerToken
	return inputCost + outputCost, nil
}

// GetTokenCount returns an estimated token count for the given text
func (m *MockModelProvider) GetTokenCount(text string) (int, error) {
	// A very rough approximation: divide character count by 4 to estimate tokens
	if len(text) == 0 {
		return 0, nil
	}
	return len([]rune(text)) / 4, nil
}

// Initialize sets up the provider with configuration
func (m *MockModelProvider) Initialize(config map[string]interface{}) error {
	m.config = config
	return nil
}

// Close performs cleanup operations for the provider
func (m *MockModelProvider) Close() error {
	// For mock provider, no cleanup needed
	return nil
}