package models

import (
	"math"
	"strings"
)

// SimilarityEvaluationStrategy uses string similarity to evaluate outputs
type SimilarityEvaluationStrategy struct{}

// NewSimilarityEvaluationStrategy creates a new SimilarityEvaluationStrategy
func NewSimilarityEvaluationStrategy() *SimilarityEvaluationStrategy {
	return &SimilarityEvaluationStrategy{}
}

// Evaluate calculates similarity between expected and actual output
func (s *SimilarityEvaluationStrategy) Evaluate(prompt string, expectedOutput string, actualOutput string, model Model) (*EvaluationResult, error) {
	score := calculateSimilarity(expectedOutput, actualOutput)
	
	return &EvaluationResult{
		Score:   score,
		Correct: score >= 0.8, // Consider correct if similarity is 80% or higher
		Details: map[string]interface{}{
			"strategy": "similarity",
			"similarity_score": score,
		},
	}, nil
}

// Name returns the name of this strategy
func (s *SimilarityEvaluationStrategy) Name() string {
	return "Similarity"
}

// calculateSimilarity calculates the similarity between two strings using a simple approach
func calculateSimilarity(str1, str2 string) float64 {
	s1 := strings.Fields(strings.ToLower(strings.TrimSpace(str1)))
	s2 := strings.Fields(strings.ToLower(strings.TrimSpace(str2)))

	// Calculate intersection of words
	intersection := 0
	for _, w1 := range s1 {
		for _, w2 := range s2 {
			if w1 == w2 {
				intersection++
				break
			}
		}
	}

	// Calculate union of words
	union := len(s1) + len(s2) - intersection

	if union == 0 {
		return 1.0 // Both strings are empty
	}

	// Jaccard similarity
	similarity := float64(intersection) / float64(union)
	
	// Also consider the length difference
	maxLen := math.Max(float64(len(s1)), float64(len(s2)))
	if maxLen == 0 {
		return 1.0
	}
	
	lengthSimilarity := 1.0 - math.Abs(float64(len(s1))-float64(len(s2)))/maxLen
	
	// Weight the similarity scores (50% Jaccard, 50% length similarity)
	weightedScore := 0.5*similarity + 0.5*lengthSimilarity
	
	return weightedScore
}