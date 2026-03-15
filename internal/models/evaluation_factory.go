package models

import (
	"fmt"
	"strings"
)

// NewEvaluationStrategy returns an evaluation strategy by name.
func NewEvaluationStrategy(name string) (EvaluationStrategy, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "similarity":
		return NewSimilarityEvaluationStrategy(), nil
	case "keyword":
		return NewKeywordMatchEvaluationStrategy(nil), nil
	case "exact", "exact_match":
		return NewExactMatchEvaluationStrategy(), nil
	default:
		return nil, fmt.Errorf("unknown evaluation strategy %q", name)
	}
}
