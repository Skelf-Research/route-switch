package models

// Model represents a language model with its properties
type Model struct {
	Name     string
	Provider string
	CostPerToken float64
	MaxTokens    int
	Description  string
}

// ModelProvider is an interface for interacting with different model providers
type ModelProvider interface {
	Name() string
	ListModels() ([]Model, error)
	GetModel(name string) (Model, error)
	CallModel(modelName, prompt string) (string, error)
}