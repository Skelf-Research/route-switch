package core

import (
	"github.com/skelf-research/route-switch/internal/models"
	"github.com/skelf-research/route-switch/internal/optimizer"
	"github.com/skelf-research/route-switch/internal/utils"
)

// Service encapsulates the main functionality of route-switch
type Service struct {
	modelProvider models.ModelProvider
	optimizer     optimizer.PromptOptimizer
	costCalc      *utils.CostCalculator
}

// Result represents the output of operations
type Result struct {
	OriginalPrompt   string
	OptimizedPrompt  string
	Model            string
	Cost             float64
	ImprovementScore float64
}

// NewService creates a new instance of Service
func NewService() *Service {
	// Initialize with mock implementations for now
	provider := models.NewMockModelProvider()
	// Using the new MIPROv2 optimizer
	opt := optimizer.NewMIPROv2()
	costCalc := utils.NewCostCalculator()
	
	return &Service{
		modelProvider: provider,
		optimizer:     opt,
		costCalc:      costCalc,
	}
}

// OptimizePrompt optimizes a prompt for a specific model
func (s *Service) OptimizePrompt(prompt, model string) (*Result, error) {
	// Use the optimizer to improve the prompt for the given model
	optimizedPrompt, err := s.optimizer.Optimize(prompt, model)
	if err != nil {
		return nil, err
	}
	
	// Evaluate the improvement
	score, err := s.optimizer.Evaluate(optimizedPrompt, model)
	if err != nil {
		return nil, err
	}
	
	return &Result{
		OriginalPrompt:   prompt,
		OptimizedPrompt:  optimizedPrompt,
		Model:            model,
		ImprovementScore: score,
	}, nil
}

// FindBestModel finds the best model and optimizes the prompt for it
func (s *Service) FindBestModel(prompt, model string) (*Result, error) {
	// First, optimize the prompt for the given model
	optimizedPrompt, err := s.optimizer.Optimize(prompt, model)
	if err != nil {
		return nil, err
	}
	
	// Get all available models
	modelsList, err := s.modelProvider.ListModels()
	if err != nil {
		return nil, err
	}
	
	// For simplicity, we'll just use the first model as the "best" for now
	// In a real implementation, this would evaluate cost/quality tradeoffs
	bestModel := modelsList[0]
	if len(modelsList) > 1 {
		// Find the cheapest model
		for _, m := range modelsList {
			if m.CostPerToken < bestModel.CostPerToken {
				bestModel = m
			}
		}
	}
	
	// Calculate cost (simplified)
	// In a real implementation, this would be based on actual token usage
	cost, err := s.costCalc.CalculateCost(bestModel.Name, 100, 200) // 100 input tokens, 200 output tokens
	if err != nil {
		return nil, err
	}
	
	// Evaluate the improvement
	score, err := s.optimizer.Evaluate(optimizedPrompt, bestModel.Name)
	if err != nil {
		return nil, err
	}
	
	return &Result{
		OriginalPrompt:   prompt,
		OptimizedPrompt:  optimizedPrompt,
		Model:            bestModel.Name,
		Cost:             cost,
		ImprovementScore: score,
	}, nil
}