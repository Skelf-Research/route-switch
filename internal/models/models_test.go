package models

import (
	"errors"
	"testing"
)

func TestModelError(t *testing.T) {
	t.Run("Error with model and provider", func(t *testing.T) {
		err := NewModelError("GetModel", "gpt-4", "openai", ErrModelNotFound)
		expected := "GetModel: model=gpt-4 provider=openai: model not found"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Error with only model", func(t *testing.T) {
		err := &ModelError{Op: "GetModel", Model: "gpt-4", Err: ErrModelNotFound}
		expected := "GetModel: model=gpt-4: model not found"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Error with only provider", func(t *testing.T) {
		err := &ModelError{Op: "Initialize", Provider: "openai", Err: ErrAPIKeyMissing}
		expected := "Initialize: provider=openai: API key is missing"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Error with only op", func(t *testing.T) {
		err := &ModelError{Op: "DoSomething", Err: ErrInvalidInput}
		expected := "DoSomething: invalid input"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		baseErr := ErrModelNotFound
		err := NewModelError("GetModel", "gpt-4", "openai", baseErr)
		if !errors.Is(err, baseErr) {
			t.Error("expected error to unwrap to base error")
		}
	})
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"ErrNotFound", ErrNotFound, true},
		{"ErrModelNotFound", ErrModelNotFound, true},
		{"wrapped ErrNotFound", NewModelError("Op", "", "", ErrNotFound), true},
		{"ErrInvalidInput", ErrInvalidInput, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFoundError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"ErrInvalidInput", ErrInvalidInput, true},
		{"ErrEmptyPrompt", ErrEmptyPrompt, true},
		{"ErrEmptyModelName", ErrEmptyModelName, true},
		{"ErrNegativeTokens", ErrNegativeTokens, true},
		{"wrapped ErrEmptyPrompt", NewModelError("Op", "", "", ErrEmptyPrompt), true},
		{"ErrNotFound", ErrNotFound, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidationError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsProviderError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"ErrProviderNotAvailable", ErrProviderNotAvailable, true},
		{"ErrProviderNotInitialized", ErrProviderNotInitialized, true},
		{"ErrAPIKeyMissing", ErrAPIKeyMissing, true},
		{"ErrRateLimited", ErrRateLimited, true},
		{"wrapped ErrRateLimited", NewModelError("Op", "", "openai", ErrRateLimited), true},
		{"ErrNotFound", ErrNotFound, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsProviderError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestValidateModel(t *testing.T) {
	tests := []struct {
		name    string
		model   Model
		wantErr error
	}{
		{
			name:    "valid model",
			model:   Model{Name: "gpt-4", Provider: "openai", CostPerToken: 0.001, MaxTokens: 8192},
			wantErr: nil,
		},
		{
			name:    "empty name",
			model:   Model{Name: "", Provider: "openai"},
			wantErr: ErrEmptyModelName,
		},
		{
			name:    "negative max tokens",
			model:   Model{Name: "gpt-4", MaxTokens: -1},
			wantErr: ErrInvalidInput,
		},
		{
			name:    "negative cost per token",
			model:   Model{Name: "gpt-4", CostPerToken: -0.001},
			wantErr: ErrInvalidInput,
		},
		{
			name:    "zero max tokens is valid",
			model:   Model{Name: "gpt-4", MaxTokens: 0},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModel(tt.model)
			if tt.wantErr == nil && err != nil {
				t.Errorf("unexpected error: %v", err)
			} else if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidateExample(t *testing.T) {
	tests := []struct {
		name    string
		example Example
		wantErr bool
	}{
		{
			name:    "valid example",
			example: Example{Input: "What is 2+2?", Output: "4"},
			wantErr: false,
		},
		{
			name:    "empty input",
			example: Example{Input: "", Output: "4"},
			wantErr: true,
		},
		{
			name:    "empty output",
			example: Example{Input: "What is 2+2?", Output: ""},
			wantErr: true,
		},
		{
			name:    "both empty",
			example: Example{Input: "", Output: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExample(tt.example)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
