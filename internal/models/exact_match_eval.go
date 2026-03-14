package models

import (
	"strings"
)

// ExactMatchEvaluationStrategy checks if the output exactly matches the expected output
type ExactMatchEvaluationStrategy struct{}

// NewExactMatchEvaluationStrategy creates a new ExactMatchEvaluationStrategy
func NewExactMatchEvaluationStrategy() *ExactMatchEvaluationStrategy {
	return &ExactMatchEvaluationStrategy{}
}

// Evaluate checks if the actual output exactly matches the expected output
func (e *ExactMatchEvaluationStrategy) Evaluate(prompt string, expectedOutput string, actualOutput string, model Model) (*EvaluationResult, error) {
	score := 0.0
	correct := false

	if strings.TrimSpace(actualOutput) == strings.TrimSpace(expectedOutput) {
		score = 1.0
		correct = true
	}

	return &EvaluationResult{
		Score:   score,
		Correct: correct,
		Details: map[string]interface{}{
			"strategy": "exact_match",
		},
	}, nil
}

// Name returns the name of this strategy
func (e *ExactMatchEvaluationStrategy) Name() string {
	return "ExactMatch"
}