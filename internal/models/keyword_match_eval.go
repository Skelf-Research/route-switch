package models

import (
	"strings"
)

// KeywordMatchEvaluationStrategy checks if keywords from expected output appear in actual output
type KeywordMatchEvaluationStrategy struct {
	Keywords []string
}

// NewKeywordMatchEvaluationStrategy creates a new KeywordMatchEvaluationStrategy
func NewKeywordMatchEvaluationStrategy(keywords []string) *KeywordMatchEvaluationStrategy {
	return &KeywordMatchEvaluationStrategy{
		Keywords: keywords,
	}
}

// Evaluate checks if keywords from expected output appear in actual output
func (k *KeywordMatchEvaluationStrategy) Evaluate(prompt string, expectedOutput string, actualOutput string, model Model) (*EvaluationResult, error) {
	// If no keywords provided, extract them from the expected output
	keywords := k.Keywords
	if len(keywords) == 0 {
		// Extract keywords by taking significant words from expected output
		keywords = extractKeywords(expectedOutput)
	}
	
	actualOutputLower := strings.ToLower(actualOutput)
	
	matchedKeywords := 0
	for _, keyword := range keywords {
		if strings.Contains(actualOutputLower, strings.ToLower(keyword)) {
			matchedKeywords++
		}
	}
	
	score := 0.0
	if len(keywords) > 0 {
		score = float64(matchedKeywords) / float64(len(keywords))
	}
	
	return &EvaluationResult{
		Score:   score,
		Correct: score >= 0.7, // Consider correct if 70% or more keywords match
		Details: map[string]interface{}{
			"strategy": "keyword_match",
			"matched_keywords": matchedKeywords,
			"total_keywords": len(keywords),
		},
	}, nil
}

// Name returns the name of this strategy
func (k *KeywordMatchEvaluationStrategy) Name() string {
	return "KeywordMatch"
}

// extractKeywords extracts significant keywords from text
func extractKeywords(text string) []string {
	// Convert to lowercase and split into words
	words := strings.Fields(strings.ToLower(text))
	
	// Filter out common stop words and return significant words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
		"should": true, "may": true, "might": true, "must": true, "can": true,
	}
	
	var keywords []string
	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?:;\"'()[]{}")
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}
	
	return keywords
}