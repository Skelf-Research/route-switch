package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/skelf-research/route-switch/internal/config"
	"github.com/skelf-research/route-switch/internal/models"
	"github.com/skelf-research/route-switch/internal/optimizer"
	"github.com/skelf-research/route-switch/internal/storage/dataset"
)

// Service errors
var (
	ErrNilServiceConfig   = errors.New("service configuration cannot be nil")
	ErrNilModelProvider   = errors.New("model provider cannot be nil")
	ErrNilOptimizer       = errors.New("optimizer cannot be nil")
	ErrNilEvaluator       = errors.New("evaluator cannot be nil")
	ErrNoModelsAvailable  = errors.New("no models available for optimization")
	ErrEmptyPrompt        = errors.New("prompt cannot be empty")
	ErrEmptyModel         = errors.New("model name cannot be empty")
	ErrOptimizationFailed = errors.New("optimization failed")
)

// Service encapsulates the main functionality of route-switch
type Service struct {
	modelProvider models.ModelProvider
	optimizer     optimizer.ExtendedPromptOptimizer
	evaluator     models.EvaluationStrategy
	config        *config.Config
	datasetStore  dataset.DatasetStore
	cacheMu       sync.RWMutex
	exampleCache  map[string]examplesCacheEntry
	cacheTTL      time.Duration
}

type examplesCacheEntry struct {
	examples []models.Example
	expires  time.Time
}

// Result represents the output of operations
type Result struct {
	OriginalPrompt   string
	OptimizedPrompt  string
	Model            string
	Cost             float64
	ImprovementScore float64
	Details          map[string]interface{}
}

// ServiceConfig holds configuration for the service
type ServiceConfig struct {
	ModelProvider models.ModelProvider
	Evaluator     models.EvaluationStrategy
	Optimizer     optimizer.ExtendedPromptOptimizer
	Config        *config.Config
	DatasetStore  dataset.DatasetStore
}

// Validate validates the service configuration
func (sc *ServiceConfig) Validate() error {
	if sc == nil {
		return ErrNilServiceConfig
	}
	if sc.ModelProvider == nil {
		return ErrNilModelProvider
	}
	if sc.Optimizer == nil {
		return ErrNilOptimizer
	}
	if sc.Evaluator == nil {
		return ErrNilEvaluator
	}
	return nil
}

// NewService creates a new instance of Service with dependency injection
func NewService(serviceConfig *ServiceConfig) *Service {
	if serviceConfig == nil {
		return nil
	}

	return &Service{
		modelProvider: serviceConfig.ModelProvider,
		optimizer:     serviceConfig.Optimizer,
		evaluator:     serviceConfig.Evaluator,
		config:        serviceConfig.Config,
		datasetStore:  serviceConfig.DatasetStore,
		exampleCache:  make(map[string]examplesCacheEntry),
		cacheTTL:      defaultExampleCacheTTL(serviceConfig.Config),
	}
}

// NewServiceWithValidation creates a new service instance with validation
func NewServiceWithValidation(serviceConfig *ServiceConfig) (*Service, error) {
	if err := serviceConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid service configuration: %w", err)
	}

	return &Service{
		modelProvider: serviceConfig.ModelProvider,
		optimizer:     serviceConfig.Optimizer,
		evaluator:     serviceConfig.Evaluator,
		config:        serviceConfig.Config,
		datasetStore:  serviceConfig.DatasetStore,
		exampleCache:  make(map[string]examplesCacheEntry),
		cacheTTL:      defaultExampleCacheTTL(serviceConfig.Config),
	}, nil
}

// OptimizePrompt optimizes a prompt for a specific model
func (s *Service) OptimizePrompt(prompt, model string) (*Result, error) {
	return s.OptimizePromptWithTemplate(prompt, model, "")
}

// OptimizePromptWithTemplate optimizes a prompt with an optional template ID used for dataset lookups
func (s *Service) OptimizePromptWithTemplate(prompt, model, templateID string) (*Result, error) {
	// Validate inputs
	if prompt == "" {
		return nil, ErrEmptyPrompt
	}
	if model == "" {
		return nil, ErrEmptyModel
	}

	// Validate service is properly initialized
	if s.modelProvider == nil {
		return nil, ErrNilModelProvider
	}
	if s.optimizer == nil {
		return nil, ErrNilOptimizer
	}

	// Get the model to ensure it's valid
	modelInfo, err := s.modelProvider.GetModel(model)
	if err != nil {
		return nil, fmt.Errorf("model not available: %w", err)
	}

	// Prepare examples sourced from the dataset when available
	examples := s.getExamples(templateID)

	// Use the optimizer to improve the prompt for the given model
	optimizationResult, err := s.optimizer.OptimizePrompt(prompt, modelInfo, examples)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOptimizationFailed, err)
	}

	if optimizationResult == nil {
		return nil, fmt.Errorf("%w: optimizer returned nil result", ErrOptimizationFailed)
	}

	// Evaluate the improvement
	score, err := s.optimizer.EvaluatePrompt(optimizationResult.Prompt, modelInfo, examples)
	if err != nil {
		// Log but don't fail - evaluation is not critical
		score = 0.0
	}

	// Calculate cost with proper error handling
	cost := s.calculateCost(model, prompt, optimizationResult.Prompt)

	return &Result{
		OriginalPrompt:   prompt,
		OptimizedPrompt:  optimizationResult.Prompt,
		Model:            model,
		Cost:             cost,
		ImprovementScore: score,
		Details:          optimizationResult.Metadata,
	}, nil
}

// FindBestModel finds the best model and optimizes the prompt for it
func (s *Service) FindBestModel(prompt, baseModel string) (*Result, error) {
	return s.FindBestModelWithTemplate(prompt, baseModel, "")
}

// FindBestModelWithTemplate finds the best model while leveraging dataset context identified by templateID.
func (s *Service) FindBestModelWithTemplate(prompt, baseModel, templateID string) (*Result, error) {
	// Validate inputs
	if prompt == "" {
		return nil, ErrEmptyPrompt
	}

	// Validate service is properly initialized
	if s.modelProvider == nil {
		return nil, ErrNilModelProvider
	}
	if s.optimizer == nil {
		return nil, ErrNilOptimizer
	}

	// Get all available models
	modelsList, err := s.modelProvider.ListModels()
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	if len(modelsList) == 0 {
		return nil, ErrNoModelsAvailable
	}

	var bestResult *Result
	bestScore := -1.0
	var lastError error

	// Get examples
	examples := s.getExamples(templateID)

	// Try each model and find the best combination of quality and cost
	for _, modelInfo := range modelsList {
		result, err := s.tryModel(prompt, modelInfo, examples)
		if err != nil {
			lastError = err
			continue // Try the next model
		}

		// Calculate a weighted score that considers both quality and cost
		weightedScore := s.calculateWeightedScore(result.ImprovementScore, result.Cost)

		if weightedScore > bestScore {
			bestScore = weightedScore
			bestResult = result
		}
	}

	if bestResult == nil {
		if lastError != nil {
			return nil, fmt.Errorf("failed to find a suitable model for the prompt: %w", lastError)
		}
		return nil, fmt.Errorf("failed to find a suitable model for the prompt")
	}

	return bestResult, nil
}

// tryModel attempts to optimize a prompt for a specific model
func (s *Service) tryModel(prompt string, modelInfo models.Model, examples []models.Example) (*Result, error) {
	optimizationResult, err := s.optimizer.OptimizePrompt(prompt, modelInfo, examples)
	if err != nil {
		return nil, fmt.Errorf("optimization failed for model %s: %w", modelInfo.Name, err)
	}

	if optimizationResult == nil {
		return nil, fmt.Errorf("optimizer returned nil result for model %s", modelInfo.Name)
	}

	// Evaluate the quality of the optimized prompt
	score, err := s.optimizer.EvaluatePrompt(optimizationResult.Prompt, modelInfo, examples)
	if err != nil {
		// Log but don't fail - use default score
		score = 0.0
	}

	// Calculate cost
	cost := s.calculateCost(modelInfo.Name, prompt, optimizationResult.Prompt)

	return &Result{
		OriginalPrompt:   prompt,
		OptimizedPrompt:  optimizationResult.Prompt,
		Model:            modelInfo.Name,
		Cost:             cost,
		ImprovementScore: score,
		Details:          optimizationResult.Metadata,
	}, nil
}

// calculateCost calculates the cost for a model call
func (s *Service) calculateCost(model, inputText, outputText string) float64 {
	if s.modelProvider == nil {
		return 0.0
	}

	inputTokens, err := s.modelProvider.GetTokenCount(inputText)
	if err != nil {
		inputTokens = len(inputText) / 4 // Fallback estimation
	}

	outputTokens, err := s.modelProvider.GetTokenCount(outputText)
	if err != nil {
		outputTokens = len(outputText) / 4 // Fallback estimation
	}

	cost, err := s.modelProvider.EstimateCost(model, inputTokens, outputTokens)
	if err != nil {
		return 0.0 // Fallback to zero cost if estimation fails
	}

	return cost
}

// calculateWeightedScore calculates a score that balances quality and cost
func (s *Service) calculateWeightedScore(qualityScore, cost float64) float64 {
	if cost <= 0 {
		return qualityScore
	}

	// Adjust the score based on cost (lower cost = higher effective score)
	// This is a simple model - can be customized via config
	return qualityScore / (1.0 + cost)
}

// getDefaultExamples returns default examples for optimization
// In a real implementation, these would come from a dataset
func (s *Service) getDefaultExamples() []models.Example {
	return []models.Example{
		{Input: "Write a poem about nature", Output: "Nature's beauty unfolds in morning light..."},
		{Input: "Explain quantum computing", Output: "Quantum computing uses quantum bits..."},
		{Input: "Describe the water cycle", Output: "The water cycle involves evaporation..."},
	}
}

// getExamples attempts to fetch examples for a template from the dataset store, falling back to defaults.
func (s *Service) getExamples(templateID string) []models.Example {
	if templateID == "" || s.datasetStore == nil {
		return s.getDefaultExamples()
	}

	if cached := s.getCachedExamples(templateID); cached != nil {
		return cached
	}

	ctx := context.Background()
	limit := 10
	if s.config != nil && s.config.MiproV2.MinibatchSize > 0 {
		limit = s.config.MiproV2.MinibatchSize * 2
	}

	records, err := s.datasetStore.ListRecent(ctx, templateID, limit)
	if err != nil || len(records) == 0 {
		return s.getDefaultExamples()
	}

	examples := make([]models.Example, 0, len(records))
	for _, record := range records {
		if record == nil || record.Input == "" || record.Output == "" {
			continue
		}
		examples = append(examples, models.Example{
			Input:  record.Input,
			Output: record.Output,
		})
	}

	if len(examples) == 0 {
		return s.getDefaultExamples()
	}

	s.storeExamples(templateID, examples)
	return examples
}

func (s *Service) getCachedExamples(templateID string) []models.Example {
	s.cacheMu.RLock()
	entry, ok := s.exampleCache[templateID]
	s.cacheMu.RUnlock()
	if !ok || time.Now().After(entry.expires) {
		return nil
	}
	return entry.examples
}

func (s *Service) storeExamples(templateID string, examples []models.Example) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.exampleCache[templateID] = examplesCacheEntry{
		examples: examples,
		expires:  time.Now().Add(s.cacheTTL),
	}
}

func defaultExampleCacheTTL(cfg *config.Config) time.Duration {
	if cfg != nil && cfg.Gateway.Optimization.Interval > 0 {
		return time.Duration(cfg.Gateway.Optimization.Interval/2) * time.Second
	}
	return 5 * time.Minute
}

// GetModelProvider returns the service's model provider
func (s *Service) GetModelProvider() models.ModelProvider {
	return s.modelProvider
}

// GetOptimizer returns the service's optimizer
func (s *Service) GetOptimizer() optimizer.ExtendedPromptOptimizer {
	return s.optimizer
}

// GetEvaluator returns the service's evaluator
func (s *Service) GetEvaluator() models.EvaluationStrategy {
	return s.evaluator
}

// GetConfig returns the service's configuration
func (s *Service) GetConfig() *config.Config {
	return s.config
}
