package models

import (
	"errors"
	"fmt"
)

// Model represents a language model with its properties
type Model struct {
	Name         string
	Provider     string
	CostPerToken float64
	MaxTokens    int
	Description  string
}

// Common errors with sentinel values for easy comparison
var (
	ErrNotFound             = errors.New("not found")
	ErrInvalidInput         = errors.New("invalid input")
	ErrProviderNotAvailable = errors.New("provider not available")
	ErrModelNotFound        = errors.New("model not found")
	ErrEmptyPrompt          = errors.New("prompt cannot be empty")
	ErrEmptyModelName       = errors.New("model name cannot be empty")
	ErrNegativeTokens       = errors.New("token count cannot be negative")
	ErrProviderNotInitialized = errors.New("provider not initialized")
	ErrAPIKeyMissing        = errors.New("API key is missing")
	ErrRateLimited          = errors.New("rate limited by provider")
	ErrContextLengthExceeded = errors.New("context length exceeded")
	ErrInvalidResponse      = errors.New("invalid response from provider")
)

// ModelError wraps errors with additional context
type ModelError struct {
	Op       string // Operation that failed (e.g., "GetModel", "CallModel")
	Model    string // Model name involved
	Provider string // Provider name involved
	Err      error  // Underlying error
}

func (e *ModelError) Error() string {
	if e.Model != "" && e.Provider != "" {
		return fmt.Sprintf("%s: model=%s provider=%s: %v", e.Op, e.Model, e.Provider, e.Err)
	}
	if e.Model != "" {
		return fmt.Sprintf("%s: model=%s: %v", e.Op, e.Model, e.Err)
	}
	if e.Provider != "" {
		return fmt.Sprintf("%s: provider=%s: %v", e.Op, e.Provider, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *ModelError) Unwrap() error {
	return e.Err
}

// NewModelError creates a new ModelError
func NewModelError(op, model, provider string, err error) *ModelError {
	return &ModelError{
		Op:       op,
		Model:    model,
		Provider: provider,
		Err:      err,
	}
}

// IsNotFoundError returns true if the error indicates a resource was not found
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, ErrModelNotFound)
}

// IsValidationError returns true if the error is due to invalid input
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidInput) ||
		errors.Is(err, ErrEmptyPrompt) ||
		errors.Is(err, ErrEmptyModelName) ||
		errors.Is(err, ErrNegativeTokens)
}

// IsProviderError returns true if the error is from the provider
func IsProviderError(err error) bool {
	return errors.Is(err, ErrProviderNotAvailable) ||
		errors.Is(err, ErrProviderNotInitialized) ||
		errors.Is(err, ErrAPIKeyMissing) ||
		errors.Is(err, ErrRateLimited)
}

// ModelProvider is an interface for interacting with different model providers
type ModelProvider interface {
	Name() string
	ListModels() ([]Model, error)
	GetModel(name string) (Model, error)
	CallModel(modelName, prompt string) (string, error)
	EstimateCost(modelName string, inputTokens, outputTokens int) (float64, error)
	GetTokenCount(text string) (int, error)
	Initialize(config map[string]interface{}) error
	Close() error
}

// Example represents a few-shot example
type Example struct {
	Input  string
	Output string
}

// EvaluationResult holds the result of evaluating a prompt
type EvaluationResult struct {
	Score   float64
	Correct bool
	Details map[string]interface{}
}

// EvaluationStrategy defines how to evaluate model outputs
type EvaluationStrategy interface {
	Evaluate(prompt string, expectedOutput string, actualOutput string, model Model) (*EvaluationResult, error)
	Name() string
}

// ValidateModel validates a Model struct
func ValidateModel(m Model) error {
	if m.Name == "" {
		return ErrEmptyModelName
	}
	if m.MaxTokens < 0 {
		return fmt.Errorf("%w: max tokens cannot be negative", ErrInvalidInput)
	}
	if m.CostPerToken < 0 {
		return fmt.Errorf("%w: cost per token cannot be negative", ErrInvalidInput)
	}
	return nil
}

// ValidateExample validates an Example struct
func ValidateExample(e Example) error {
	if e.Input == "" {
		return fmt.Errorf("%w: example input cannot be empty", ErrInvalidInput)
	}
	if e.Output == "" {
		return fmt.Errorf("%w: example output cannot be empty", ErrInvalidInput)
	}
	return nil
}
