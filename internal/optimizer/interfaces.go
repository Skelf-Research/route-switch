package optimizer

import (
	"github.com/skelf-research/route-switch/internal/models"
)

// BayesianOptimizer defines the interface for Bayesian optimization
type BayesianOptimizer interface {
	Optimize(searchSpace map[string]interface{}, objectiveFn func(params map[string]interface{}) (float64, error)) (map[string]interface{}, float64, error)
	Name() string
}

// PromptOptimizationResult holds the result of prompt optimization
type PromptOptimizationResult struct {
	Prompt string
	Score  float64
	Metadata map[string]interface{}
}

// ExtendedPromptOptimizer extends the basic optimizer interface with more specific functionality
type ExtendedPromptOptimizer interface {
	OptimizePrompt(basePrompt string, model models.Model, examples []models.Example) (*PromptOptimizationResult, error)
	EvaluatePrompt(prompt string, model models.Model, examples []models.Example) (float64, error)
	GetBestCandidate() *PromptOptimizationResult
}