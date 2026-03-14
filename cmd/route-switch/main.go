package main

import (
	"fmt"
	"log"

	"github.com/skelf-research/route-switch/internal/config"
	"github.com/skelf-research/route-switch/internal/core"
	"github.com/skelf-research/route-switch/internal/models"
	"github.com/skelf-research/route-switch/internal/optimizer"
)

func main() {
	// For demonstration, create a simple configuration
	configManager := config.NewSimpleConfigManager()
	appConfig := configManager.GetConfig()

	// Initialize model provider (using mock for this example)
	provider := models.NewMockModelProvider()

	// Initialize evaluation strategy
	evaluator := models.NewSimilarityEvaluationStrategy()

	// Initialize Bayesian optimizer
	bayesianOpt, err := optimizer.NewGoptunaBayesianOptimizer(map[string]interface{}{
		"num_trials": appConfig.MiproV2.NumTrials,
	})
	if err != nil {
		log.Fatalf("Failed to initialize Bayesian optimizer: %v", err)
	}

	// Initialize the MIPROv2 optimizer with all dependencies
	miproConfig := appConfig.MiproV2
	opt := optimizer.NewMIPROv2(provider, evaluator, bayesianOpt, miproConfig)

	// Set up service configuration
	serviceConfig := &core.ServiceConfig{
		ModelProvider: provider,
		Evaluator:     evaluator,
		Optimizer:     opt,
		Config:        appConfig,
	}

	// Initialize the service
	service := core.NewService(serviceConfig)

	// For now, let's run a simple example
	prompt := "Write a poem about technology"
	model := "gpt-4"

	// Example of optimizing a prompt
	fmt.Printf("Optimizing prompt: %s\n", prompt)
	result, err := service.OptimizePrompt(prompt, model)
	if err != nil {
		log.Printf("Error optimizing prompt: %v", err)
	} else {
		fmt.Printf("Optimized prompt: %s\n", result.OptimizedPrompt)
		fmt.Printf("Model: %s\n", result.Model)
		fmt.Printf("Cost: $%.6f\n", result.Cost)
	}

	// Example of finding the best model
	fmt.Printf("\nFinding best model for prompt: %s\n", prompt)
	bestResult, err := service.FindBestModel(prompt, model)
	if err != nil {
		log.Printf("Error finding best model: %v", err)
	} else {
		fmt.Printf("Best model: %s\n", bestResult.Model)
		fmt.Printf("Optimized prompt: %s\n", bestResult.OptimizedPrompt)
		fmt.Printf("Cost: $%.6f\n", bestResult.Cost)
	}

	fmt.Println("Thank you for using route-switch!")
}