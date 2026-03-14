package gateway

import (
	"sync"
	"time"

	"github.com/skelf-research/route-switch/internal/models"
)

// PromptCombination represents an optimized prompt for a specific model
type PromptCombination struct {
	ID            string
	Name          string
	TemplateID    string
	Prompt        string
	Model         string
	Provider      string
	Weight        int // Weight for load balancing (0-100)
	CreatedAt     time.Time
	LastOptimized time.Time
	Performance   *PerformanceMetrics
	Metadata      map[string]interface{}
}

// PerformanceMetrics tracks performance of a prompt+model combination
type PerformanceMetrics struct {
	ResponseTimeAvg float64
	SuccessRate     float64
	CostPerRequest  float64
	TotalRequests   int64
	LastUsed        time.Time
}

// PromptRegistry manages multiple prompt+model combinations
type PromptRegistry struct {
	mu           sync.RWMutex
	combinations map[string]*PromptCombination
}

// NewPromptRegistry creates a new prompt registry
func NewPromptRegistry() *PromptRegistry {
	return &PromptRegistry{
		combinations: make(map[string]*PromptCombination),
	}
}

// AddCombination adds a new prompt+model combination to the registry
func (pr *PromptRegistry) AddCombination(combination *PromptCombination) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if combination.ID == "" {
		return models.ErrInvalidInput
	}

	if combination.CreatedAt.IsZero() {
		combination.CreatedAt = time.Now()
	}

	if combination.Performance == nil {
		combination.Performance = &PerformanceMetrics{}
	}

	pr.combinations[combination.ID] = combination
	return nil
}

// GetCombination retrieves a prompt+model combination by ID
func (pr *PromptRegistry) GetCombination(id string) (*PromptCombination, bool) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	combination, exists := pr.combinations[id]
	return combination, exists
}

// GetCombinationByName retrieves a prompt+model combination by name
func (pr *PromptRegistry) GetCombinationByName(name string) (*PromptCombination, bool) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	for _, combination := range pr.combinations {
		if combination.Name == name {
			return combination, true
		}
	}

	return nil, false
}

// GetAllCombinations returns all registered prompt+model combinations
func (pr *PromptRegistry) GetAllCombinations() []*PromptCombination {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	combinations := make([]*PromptCombination, 0, len(pr.combinations))
	for _, combination := range pr.combinations {
		combinations = append(combinations, combination)
	}

	return combinations
}

// RemoveCombination removes a prompt+model combination by ID
func (pr *PromptRegistry) RemoveCombination(id string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if _, exists := pr.combinations[id]; !exists {
		return models.ErrNotFound
	}

	delete(pr.combinations, id)
	return nil
}

// UpdatePerformance updates the performance metrics for a combination
func (pr *PromptRegistry) UpdatePerformance(id string, responseTime time.Duration, success bool, cost float64) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	combination, exists := pr.combinations[id]
	if !exists {
		return models.ErrNotFound
	}

	metrics := combination.Performance
	if metrics == nil {
		metrics = &PerformanceMetrics{}
		combination.Performance = metrics
	}

	// Update response time average (simple moving average)
	totalRequests := metrics.TotalRequests + 1
	metrics.ResponseTimeAvg = (metrics.ResponseTimeAvg*float64(metrics.TotalRequests) + float64(responseTime.Seconds())) / float64(totalRequests)

	// Update success rate
	if success {
		metrics.SuccessRate = (metrics.SuccessRate*float64(metrics.TotalRequests) + 1.0) / float64(totalRequests)
	} else {
		metrics.SuccessRate = (metrics.SuccessRate * float64(metrics.TotalRequests)) / float64(totalRequests)
	}

	// Update cost
	metrics.CostPerRequest = (metrics.CostPerRequest*float64(metrics.TotalRequests) + cost) / float64(totalRequests)

	metrics.TotalRequests = totalRequests
	metrics.LastUsed = time.Now()

	return nil
}

// GetCombinationsByProvider returns combinations for a specific provider
func (pr *PromptRegistry) GetCombinationsByProvider(provider string) []*PromptCombination {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	var combinations []*PromptCombination
	for _, combination := range pr.combinations {
		if combination.Provider == provider {
			combinations = append(combinations, combination)
		}
	}

	return combinations
}

// GetActiveCombinations returns combinations with weight > 0 (active for load balancing)
func (pr *PromptRegistry) GetActiveCombinations() []*PromptCombination {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	var combinations []*PromptCombination
	for _, combination := range pr.combinations {
		if combination.Weight > 0 {
			combinations = append(combinations, combination)
		}
	}

	return combinations
}
