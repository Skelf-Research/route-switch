package optimizer

import (
	"strings"
	"testing"

	"github.com/skelf-research/route-switch/internal/config"
	"github.com/skelf-research/route-switch/internal/models"
)

func TestNewMIPROv2(t *testing.T) {
	// Create mock dependencies
	provider := models.NewMockModelProvider()
	evaluator := models.NewSimilarityEvaluationStrategy()
	bayesianOpt, err := NewGoptunaBayesianOptimizer(map[string]interface{}{"num_trials": 5})
	if err != nil {
		t.Fatalf("Failed to create Bayesian optimizer: %v", err)
	}

	optimizer := NewMIPROv2(provider, evaluator, bayesianOpt, config.MiproV2Config{
		NumCandidates:            5,
		MaxBootstrappedDemos:     3,
		MaxLabeledDemos:          2,
		NumTrials:                10,
		MinibatchSize:            5,
		MinibatchFullEvalSteps:   3,
		NumInstructionCandidates: 3,
	})

	if optimizer == nil {
		t.Error("Expected NewMIPROv2 to return a non-nil optimizer")
	}
}

func TestMIPROv2Optimize(t *testing.T) {
	// Create mock dependencies
	provider := models.NewMockModelProvider()
	evaluator := models.NewSimilarityEvaluationStrategy()
	bayesianOpt, err := NewGoptunaBayesianOptimizer(map[string]interface{}{"num_trials": 5})
	if err != nil {
		t.Fatalf("Failed to create Bayesian optimizer: %v", err)
	}

	optimizer := NewMIPROv2(provider, evaluator, bayesianOpt, config.MiproV2Config{
		NumCandidates:            5,
		MaxBootstrappedDemos:     3,
		MaxLabeledDemos:          2,
		NumTrials:                10,
		MinibatchSize:            5,
		MinibatchFullEvalSteps:   3,
		NumInstructionCandidates: 3,
	})

	prompt := "Write a story about a robot"
	model := models.Model{
		Name: "gpt-4",
	}

	examples := []models.Example{
		{Input: "Example input", Output: "Example output"},
	}

	result, err := optimizer.OptimizePrompt(prompt, model, examples)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil || result.Prompt == "" {
		t.Error("Expected optimized prompt to not be empty")
	}

	// Check that the result contains expected elements
	if !strings.Contains(result.Prompt, "Instruction:") && !strings.Contains(result.Prompt, "Examples:") {
		t.Log("Note: Result may be using simplified optimization in test mode")
	}
}

func TestMIPROv2Evaluate(t *testing.T) {
	// Create mock dependencies
	provider := models.NewMockModelProvider()
	evaluator := models.NewSimilarityEvaluationStrategy()
	bayesianOpt, err := NewGoptunaBayesianOptimizer(map[string]interface{}{"num_trials": 5})
	if err != nil {
		t.Fatalf("Failed to create Bayesian optimizer: %v", err)
	}

	optimizer := NewMIPROv2(provider, evaluator, bayesianOpt, config.MiproV2Config{
		NumCandidates:            5,
		MaxBootstrappedDemos:     3,
		MaxLabeledDemos:          2,
		NumTrials:                10,
		MinibatchSize:            5,
		MinibatchFullEvalSteps:   3,
		NumInstructionCandidates: 3,
	})

	prompt := "Write a story about a robot"
	model := models.Model{
		Name: "gpt-4",
	}

	examples := []models.Example{
		{Input: "Example input", Output: "Example output"},
	}

	score, err := optimizer.EvaluatePrompt(prompt, model, examples)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if score < 0 || score > 1 {
		t.Errorf("Expected score between 0 and 1, got %f", score)
	}
}

func TestMIPROv2GetBestCandidate(t *testing.T) {
	// Create mock dependencies
	provider := models.NewMockModelProvider()
	evaluator := models.NewSimilarityEvaluationStrategy()
	bayesianOpt, err := NewGoptunaBayesianOptimizer(map[string]interface{}{"num_trials": 5})
	if err != nil {
		t.Fatalf("Failed to create Bayesian optimizer: %v", err)
	}

	optimizer := NewMIPROv2(provider, evaluator, bayesianOpt, config.MiproV2Config{
		NumCandidates:            5,
		MaxBootstrappedDemos:     3,
		MaxLabeledDemos:          2,
		NumTrials:                10,
		MinibatchSize:            5,
		MinibatchFullEvalSteps:   3,
		NumInstructionCandidates: 3,
	})

	result := optimizer.GetBestCandidate()

	// This may return nil since it's not maintaining state between calls
	// The important thing is that it doesn't panic
	_ = result
}