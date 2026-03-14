package gateway

import (
	"math/rand"
	"sync"
	"time"

	"github.com/skelf-research/route-switch/internal/models"
)

// LoadBalancerStrategy defines the strategy for load balancing
type LoadBalancerStrategy string

const (
	RoundRobinStrategy LoadBalancerStrategy = "round_robin"
	LeastConnections   LoadBalancerStrategy = "least_connections"
	RandomStrategy     LoadBalancerStrategy = "random"
	WeightedRoundRobin LoadBalancerStrategy = "weighted_round_robin"
	PerformanceBased   LoadBalancerStrategy = "performance_based"
)

// LeastConnectionsStrategy constant
const LeastConnectionsStrategy = LeastConnections

// LoadBalancer distributes requests across different prompt+model combinations
type LoadBalancer struct {
	registry  *PromptRegistry
	strategy  LoadBalancerStrategy
	mutex     sync.RWMutex
	rrCounter int
	rand      *rand.Rand
}

// NewLoadBalancer creates a new load balancer with the specified strategy
func NewLoadBalancer(registry *PromptRegistry, strategy LoadBalancerStrategy) *LoadBalancer {
	return &LoadBalancer{
		registry: registry,
		strategy: strategy,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SelectCombination selects the appropriate prompt+model combination based on the strategy
func (lb *LoadBalancer) SelectCombination() (*PromptCombination, error) {
	activeCombinations := lb.registry.GetActiveCombinations()
	if len(activeCombinations) == 0 {
		return nil, models.ErrNotFound
	}

	switch lb.strategy {
	case RoundRobinStrategy:
		return lb.roundRobinSelection(activeCombinations)
	case LeastConnectionsStrategy:
		// For now, treating all as equal until we implement connection tracking
		return lb.roundRobinSelection(activeCombinations)
	case RandomStrategy:
		return lb.randomSelection(activeCombinations)
	case WeightedRoundRobin:
		return lb.weightedRoundRobinSelection(activeCombinations)
	case PerformanceBased:
		return lb.performanceBasedSelection(activeCombinations)
	default:
		// Default to round robin
		return lb.roundRobinSelection(activeCombinations)
	}
}

// roundRobinSelection selects combinations in a round-robin fashion
func (lb *LoadBalancer) roundRobinSelection(combinations []*PromptCombination) (*PromptCombination, error) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	if len(combinations) == 0 {
		return nil, models.ErrNotFound
	}

	selected := combinations[lb.rrCounter%len(combinations)]
	lb.rrCounter++

	return selected, nil
}

// randomSelection selects a random combination
func (lb *LoadBalancer) randomSelection(combinations []*PromptCombination) (*PromptCombination, error) {
	if len(combinations) == 0 {
		return nil, models.ErrNotFound
	}

	idx := lb.rand.Intn(len(combinations))
	return combinations[idx], nil
}

// weightedRoundRobinSelection selects based on weights assigned to combinations
func (lb *LoadBalancer) weightedRoundRobinSelection(combinations []*PromptCombination) (*PromptCombination, error) {
	if len(combinations) == 0 {
		return nil, models.ErrNotFound
	}

	// Calculate total weight
	totalWeight := 0
	for _, combo := range combinations {
		totalWeight += combo.Weight
	}

	if totalWeight <= 0 {
		// Fallback to round robin if no valid weights
		return lb.roundRobinSelection(combinations)
	}

	// Select based on weight
	randomWeight := lb.rand.Intn(totalWeight)
	currentWeight := 0

	for _, combo := range combinations {
		currentWeight += combo.Weight
		if randomWeight < currentWeight {
			return combo, nil
		}
	}

	// Fallback to first combination
	return combinations[0], nil
}

// performanceBasedSelection selects based on performance metrics (response time, success rate)
func (lb *LoadBalancer) performanceBasedSelection(combinations []*PromptCombination) (*PromptCombination, error) {
	if len(combinations) == 0 {
		return nil, models.ErrNotFound
	}

	// Find the combination with the best score
	// Score: lower response time + higher success rate = better performance
	bestCombo := combinations[0]
	bestScore := calculatePerformanceScore(bestCombo.Performance)

	for _, combo := range combinations[1:] {
		score := calculatePerformanceScore(combo.Performance)
		if score > bestScore {
			bestCombo = combo
			bestScore = score
		}
	}

	return bestCombo, nil
}

// calculatePerformanceScore calculates a score based on performance metrics
// Higher score is better
func calculatePerformanceScore(metrics *PerformanceMetrics) float64 {
	if metrics == nil {
		// Default to neutral score if no metrics
		return 0.5
	}

	// Normalize metrics to 0-1 scale for scoring
	// Invert response time (lower is better) and prioritize success rate
	score := 0.0

	// Success rate contributes 60% to the score
	score += metrics.SuccessRate * 0.6

	// Inverse of response time (capped to avoid extreme values)
	// Higher response time = lower score
	if metrics.ResponseTimeAvg > 0 {
		// Cap the response time contribution to 0.4 to avoid it dominating
		// Normalize by assuming 10s is the worst response time we'd see
		responseTimeScore := 1.0 - (metrics.ResponseTimeAvg / 10.0)
		if responseTimeScore < 0 {
			responseTimeScore = 0
		}

		score += responseTimeScore * 0.4
	} else {
		// If no response time data, give the maximum possible for response time score
		score += 0.4
	}

	return score
}

// UpdateStrategy updates the load balancing strategy
func (lb *LoadBalancer) UpdateStrategy(strategy LoadBalancerStrategy) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	lb.strategy = strategy
}

// GetStrategy returns the current load balancing strategy
func (lb *LoadBalancer) GetStrategy() LoadBalancerStrategy {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()
	return lb.strategy
}
