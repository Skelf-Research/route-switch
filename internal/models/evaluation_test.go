package models

import (
	"testing"
)

func TestNewEvaluationStrategy(t *testing.T) {
	tests := []struct {
		name         string
		strategyName string
		wantName     string
		wantErr      bool
	}{
		{"empty defaults to similarity", "", "Similarity", false},
		{"similarity", "similarity", "Similarity", false},
		{"Similarity uppercase", "Similarity", "Similarity", false},
		{"similarity with spaces", "  similarity  ", "Similarity", false},
		{"keyword", "keyword", "KeywordMatch", false},
		{"exact", "exact", "ExactMatch", false},
		{"exact_match", "exact_match", "ExactMatch", false},
		{"unknown strategy", "unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy, err := NewEvaluationStrategy(tt.strategyName)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if strategy.Name() != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, strategy.Name())
			}
		})
	}
}

func TestExactMatchEvaluationStrategy(t *testing.T) {
	strategy := NewExactMatchEvaluationStrategy()
	model := Model{Name: "test-model"}

	tests := []struct {
		name           string
		expected       string
		actual         string
		wantScore      float64
		wantCorrect    bool
	}{
		{
			name:        "exact match",
			expected:    "hello world",
			actual:      "hello world",
			wantScore:   1.0,
			wantCorrect: true,
		},
		{
			name:        "match with whitespace",
			expected:    "  hello world  ",
			actual:      "hello world",
			wantScore:   1.0,
			wantCorrect: true,
		},
		{
			name:        "no match",
			expected:    "hello",
			actual:      "world",
			wantScore:   0.0,
			wantCorrect: false,
		},
		{
			name:        "case sensitive",
			expected:    "Hello World",
			actual:      "hello world",
			wantScore:   0.0,
			wantCorrect: false,
		},
		{
			name:        "empty strings match",
			expected:    "",
			actual:      "",
			wantScore:   1.0,
			wantCorrect: true,
		},
		{
			name:        "partial match is no match",
			expected:    "hello world",
			actual:      "hello",
			wantScore:   0.0,
			wantCorrect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := strategy.Evaluate("prompt", tt.expected, tt.actual, model)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Score != tt.wantScore {
				t.Errorf("expected score %f, got %f", tt.wantScore, result.Score)
			}
			if result.Correct != tt.wantCorrect {
				t.Errorf("expected correct=%v, got %v", tt.wantCorrect, result.Correct)
			}
			if result.Details["strategy"] != "exact_match" {
				t.Errorf("expected strategy=exact_match in details")
			}
		})
	}

	t.Run("Name", func(t *testing.T) {
		if strategy.Name() != "ExactMatch" {
			t.Errorf("expected name ExactMatch, got %s", strategy.Name())
		}
	})
}

func TestSimilarityEvaluationStrategy(t *testing.T) {
	strategy := NewSimilarityEvaluationStrategy()
	model := Model{Name: "test-model"}

	tests := []struct {
		name           string
		expected       string
		actual         string
		minScore       float64
		maxScore       float64
		wantCorrect    bool
	}{
		{
			name:        "identical strings",
			expected:    "the quick brown fox",
			actual:      "the quick brown fox",
			minScore:    0.99,
			maxScore:    1.01,
			wantCorrect: true,
		},
		{
			name:        "similar strings",
			expected:    "the quick brown fox jumps",
			actual:      "the quick brown dog jumps",
			minScore:    0.7,
			maxScore:    1.0,
			wantCorrect: true,
		},
		{
			name:        "completely different",
			expected:    "hello world",
			actual:      "goodbye universe",
			minScore:    0.0,
			maxScore:    0.5,
			wantCorrect: false,
		},
		{
			name:        "empty strings",
			expected:    "",
			actual:      "",
			minScore:    0.99,
			maxScore:    1.01,
			wantCorrect: true,
		},
		{
			name:        "case insensitive",
			expected:    "HELLO WORLD",
			actual:      "hello world",
			minScore:    0.99,
			maxScore:    1.01,
			wantCorrect: true,
		},
		{
			name:        "partial overlap",
			expected:    "the quick brown fox",
			actual:      "quick fox",
			minScore:    0.3,
			maxScore:    0.8,
			wantCorrect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := strategy.Evaluate("prompt", tt.expected, tt.actual, model)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Score < tt.minScore || result.Score > tt.maxScore {
				t.Errorf("expected score between %f and %f, got %f", tt.minScore, tt.maxScore, result.Score)
			}
			if result.Correct != tt.wantCorrect {
				t.Errorf("expected correct=%v, got %v (score=%f)", tt.wantCorrect, result.Correct, result.Score)
			}
			if result.Details["strategy"] != "similarity" {
				t.Errorf("expected strategy=similarity in details")
			}
		})
	}

	t.Run("Name", func(t *testing.T) {
		if strategy.Name() != "Similarity" {
			t.Errorf("expected name Similarity, got %s", strategy.Name())
		}
	})
}

func TestKeywordMatchEvaluationStrategy(t *testing.T) {
	model := Model{Name: "test-model"}

	t.Run("with predefined keywords", func(t *testing.T) {
		strategy := NewKeywordMatchEvaluationStrategy([]string{"quick", "brown", "fox"})

		tests := []struct {
			name        string
			actual      string
			wantScore   float64
			wantCorrect bool
		}{
			{
				name:        "all keywords match",
				actual:      "The quick brown fox jumps",
				wantScore:   1.0,
				wantCorrect: true,
			},
			{
				name:        "some keywords match",
				actual:      "The quick dog jumps",
				wantScore:   1.0 / 3.0,
				wantCorrect: false,
			},
			{
				name:        "no keywords match",
				actual:      "The slow gray dog walks",
				wantScore:   0.0,
				wantCorrect: false,
			},
			{
				name:        "case insensitive",
				actual:      "THE QUICK BROWN FOX",
				wantScore:   1.0,
				wantCorrect: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := strategy.Evaluate("prompt", "expected", tt.actual, model)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result.Score != tt.wantScore {
					t.Errorf("expected score %f, got %f", tt.wantScore, result.Score)
				}
				if result.Correct != tt.wantCorrect {
					t.Errorf("expected correct=%v, got %v", tt.wantCorrect, result.Correct)
				}
			})
		}
	})

	t.Run("extract keywords from expected output", func(t *testing.T) {
		strategy := NewKeywordMatchEvaluationStrategy(nil)

		// When no keywords provided, should extract from expected output
		result, err := strategy.Evaluate("prompt", "The important concept is machine learning algorithms", "machine learning algorithms are important", model)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have high score since main keywords match
		if result.Score < 0.5 {
			t.Errorf("expected score >= 0.5 for matching keywords, got %f", result.Score)
		}

		if result.Details["total_keywords"].(int) == 0 {
			t.Error("expected some keywords to be extracted")
		}
	})

	t.Run("empty keywords with empty expected", func(t *testing.T) {
		strategy := NewKeywordMatchEvaluationStrategy(nil)
		result, err := strategy.Evaluate("prompt", "the a an", "anything", model) // Only stop words
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// With only stop words, no keywords are extracted, score should be 0
		if result.Score != 0.0 {
			t.Errorf("expected score 0.0 for no keywords, got %f", result.Score)
		}
	})

	t.Run("Name", func(t *testing.T) {
		strategy := NewKeywordMatchEvaluationStrategy(nil)
		if strategy.Name() != "KeywordMatch" {
			t.Errorf("expected name KeywordMatch, got %s", strategy.Name())
		}
	})
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "extracts content words",
			text:     "The quick brown fox jumps over the lazy dog",
			expected: []string{"quick", "brown", "fox", "jumps", "over", "lazy", "dog"},
		},
		{
			name:     "removes punctuation",
			text:     "Hello, world! How are you?",
			expected: []string{"hello", "world", "how", "you"},
		},
		{
			name:     "filters short words",
			text:     "I am a go programmer",
			expected: []string{"programmer"},
		},
		{
			name:     "empty string",
			text:     "",
			expected: nil,
		},
		{
			name:     "only stop words",
			text:     "the a an and or but in on",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractKeywords(tt.text)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d keywords, got %d: %v", len(tt.expected), len(result), result)
				return
			}
			for i, kw := range result {
				if kw != tt.expected[i] {
					t.Errorf("expected keyword %q at index %d, got %q", tt.expected[i], i, kw)
				}
			}
		})
	}
}

func TestCalculateSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		str1     string
		str2     string
		minScore float64
		maxScore float64
	}{
		{
			name:     "identical strings",
			str1:     "hello world",
			str2:     "hello world",
			minScore: 0.99,
			maxScore: 1.01,
		},
		{
			name:     "empty strings",
			str1:     "",
			str2:     "",
			minScore: 0.99,
			maxScore: 1.01,
		},
		{
			name:     "one empty string",
			str1:     "hello",
			str2:     "",
			minScore: 0.0,
			maxScore: 0.5,
		},
		{
			name:     "completely different",
			str1:     "abc def",
			str2:     "xyz uvw",
			minScore: 0.0,
			maxScore: 0.6,
		},
		{
			name:     "partial overlap",
			str1:     "hello world",
			str2:     "hello there",
			minScore: 0.4,
			maxScore: 0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateSimilarity(tt.str1, tt.str2)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("expected score between %f and %f, got %f", tt.minScore, tt.maxScore, score)
			}
		})
	}
}
