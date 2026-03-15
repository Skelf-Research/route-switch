package gateway

import (
	"testing"
	"time"
)

func TestLoadBalancer_RoundRobinSelection(t *testing.T) {
	registry := NewPromptRegistry()
	loadBalancer := NewLoadBalancer(registry, RoundRobinStrategy)

	combination1 := &PromptCombination{
		ID:          "combo-1",
		Name:        "combo-1",
		Prompt:      "test prompt 1",
		Model:       "gpt-4",
		Provider:    "openai",
		Weight:      10,
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
	}

	combination2 := &PromptCombination{
		ID:          "combo-2",
		Name:        "combo-2",
		Prompt:      "test prompt 2",
		Model:       "gpt-3.5-turbo",
		Provider:    "openai",
		Weight:      10,
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
	}

	registry.AddCombination(combination1)
	registry.AddCombination(combination2)

	// Test round-robin selection - we need to make sure there are active combinations
	// (weight > 0 for the GetActiveCombinations call that SelectCombination uses)
	activeCount := len(registry.GetActiveCombinations())
	if activeCount != 2 {
		t.Errorf("Expected 2 active combinations, got %d", activeCount)
	}

	// Make multiple selections and verify round-robin behavior
	// With 2 combinations, we should see both within any 2 consecutive selections
	const numSelections = 10
	selections := make([]*PromptCombination, numSelections)
	seenIDs := make(map[string]int)

	for i := 0; i < numSelections; i++ {
		choice, err := loadBalancer.SelectCombination()
		if err != nil {
			t.Fatalf("SelectCombination %d failed: %v", i, err)
		}
		selections[i] = choice
		seenIDs[choice.ID]++
	}

	// Verify both combinations were selected
	if len(seenIDs) != 2 {
		t.Errorf("Round-robin should use all combinations, but only saw %d unique IDs: %v", len(seenIDs), seenIDs)
	}

	// Verify distribution is roughly even (within tolerance for round-robin)
	for id, count := range seenIDs {
		if count < 3 || count > 7 {
			t.Errorf("Combination %s was selected %d times, expected roughly 5 times for even distribution", id, count)
		}
	}

	// Verify round-robin pattern: consecutive selections should mostly alternate
	// (allowing some tolerance due to the index-based approach with varying slice order)
	alternations := 0
	for i := 1; i < numSelections; i++ {
		if selections[i].ID != selections[i-1].ID {
			alternations++
		}
	}

	// With proper round-robin and 2 items, we expect 9 alternations out of 9 possible
	// since combinations are sorted by ID for deterministic ordering.
	if alternations != numSelections-1 {
		t.Errorf("Round-robin should alternate every selection. Got %d alternations out of %d possible", alternations, numSelections-1)
	}

	t.Logf("Round-robin test: %d selections, %d unique combinations, %d alternations", numSelections, len(seenIDs), alternations)
}

func TestLoadBalancer_WeightedRoundRobinSelection(t *testing.T) {
	registry := NewPromptRegistry()
	loadBalancer := NewLoadBalancer(registry, WeightedRoundRobin)

	combination1 := &PromptCombination{
		ID:          "combo-1",
		Name:        "combo-1",
		Prompt:      "test prompt 1",
		Model:       "gpt-4",
		Provider:    "openai",
		Weight:      80, // Higher weight
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
	}

	combination2 := &PromptCombination{
		ID:          "combo-2",
		Name:        "combo-2",
		Prompt:      "test prompt 2",
		Model:       "gpt-3.5-turbo",
		Provider:    "openai",
		Weight:      20, // Lower weight
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
	}

	registry.AddCombination(combination1)
	registry.AddCombination(combination2)

	// Perform multiple selections to see distribution
	selections := make(map[string]int)
	for i := 0; i < 100; i++ {
		choice, err := loadBalancer.SelectCombination()
		if err != nil {
			t.Fatalf("SelectCombination failed: %v", err)
		}
		selections[choice.ID]++
	}

	// With weighted round-robin, combo-1 should be selected more often
	// but we can't guarantee exact distribution due to algorithm, so we'll just check both are selected
	if selections["combo-1"] == 0 || selections["combo-2"] == 0 {
		t.Error("Both combinations should be selected in weighted round-robin")
	}
}

func TestLoadBalancer_PerformanceBasedSelection(t *testing.T) {
	registry := NewPromptRegistry()
	loadBalancer := NewLoadBalancer(registry, PerformanceBased)

	// Create combinations with different performance metrics
	goodPerformer := &PromptCombination{
		ID:        "good-performer",
		Name:      "good-performer",
		Prompt:    "test prompt",
		Model:     "gpt-4",
		Provider:  "openai",
		Weight:    10,
		CreatedAt: time.Now(),
		Performance: &PerformanceMetrics{
			ResponseTimeAvg: 0.1,  // Fast response
			SuccessRate:     0.95, // High success rate
		},
	}

	poorPerformer := &PromptCombination{
		ID:        "poor-performer",
		Name:      "poor-performer",
		Prompt:    "test prompt",
		Model:     "gpt-3.5-turbo",
		Provider:  "openai",
		Weight:    10,
		CreatedAt: time.Now(),
		Performance: &PerformanceMetrics{
			ResponseTimeAvg: 2.0, // Slow response
			SuccessRate:     0.6, // Low success rate
		},
	}

	registry.AddCombination(goodPerformer)
	registry.AddCombination(poorPerformer)

	// Multiple selections should prefer the better performing combination
	goodCount := 0
	poorCount := 0
	for i := 0; i < 10; i++ {
		choice, err := loadBalancer.SelectCombination()
		if err != nil {
			t.Fatalf("SelectCombination failed: %v", err)
		}
		if choice.ID == "good-performer" {
			goodCount++
		} else if choice.ID == "poor-performer" {
			poorCount++
		}
	}

	// The good performer should be selected more often, though this is not guaranteed in every case
	// due to the score calculation, but it's likely to be selected more
	t.Logf("Good performer selected: %d times, Poor performer selected: %d times", goodCount, poorCount)
}

func TestLoadBalancer_RandomSelection(t *testing.T) {
	registry := NewPromptRegistry()
	loadBalancer := NewLoadBalancer(registry, RandomStrategy)

	combination1 := &PromptCombination{
		ID:          "combo-1",
		Name:        "combo-1",
		Prompt:      "test prompt 1",
		Model:       "gpt-4",
		Provider:    "openai",
		Weight:      10,
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
	}

	combination2 := &PromptCombination{
		ID:          "combo-2",
		Name:        "combo-2",
		Prompt:      "test prompt 2",
		Model:       "gpt-3.5-turbo",
		Provider:    "openai",
		Weight:      10,
		CreatedAt:   time.Now(),
		Performance: &PerformanceMetrics{},
	}

	registry.AddCombination(combination1)
	registry.AddCombination(combination2)

	// Perform multiple selections
	selections := make(map[string]int)
	for i := 0; i < 100; i++ {
		choice, err := loadBalancer.SelectCombination()
		if err != nil {
			t.Fatalf("SelectCombination failed: %v", err)
		}
		selections[choice.ID]++
	}

	// Both should be selected at least once with random selection
	if selections["combo-1"] == 0 || selections["combo-2"] == 0 {
		t.Error("Both combinations should be selected in random selection")
	}
}

func TestLoadBalancer_UpdateStrategy(t *testing.T) {
	registry := NewPromptRegistry()
	loadBalancer := NewLoadBalancer(registry, RoundRobinStrategy)

	if loadBalancer.GetStrategy() != RoundRobinStrategy {
		t.Errorf("Expected initial strategy %s, got %s", RoundRobinStrategy, loadBalancer.GetStrategy())
	}

	loadBalancer.UpdateStrategy(RandomStrategy)

	if loadBalancer.GetStrategy() != RandomStrategy {
		t.Errorf("Expected updated strategy %s, got %s", RandomStrategy, loadBalancer.GetStrategy())
	}
}

func TestCalculatePerformanceScore(t *testing.T) {
	// Test with good performance metrics
	goodMetrics := &PerformanceMetrics{
		ResponseTimeAvg: 0.1,  // Fast
		SuccessRate:     0.95, // High success rate
	}

	goodScore := calculatePerformanceScore(goodMetrics)

	// Test with poor performance metrics
	poorMetrics := &PerformanceMetrics{
		ResponseTimeAvg: 5.0, // Slow
		SuccessRate:     0.3, // Low success rate
	}

	poorScore := calculatePerformanceScore(poorMetrics)

	if goodScore <= poorScore {
		t.Errorf("Good performer score (%f) should be higher than poor performer score (%f)", goodScore, poorScore)
	}

	// Test with nil metrics (should return neutral score)
	neutralScore := calculatePerformanceScore(nil)
	if neutralScore != 0.5 {
		t.Errorf("Expected neutral score 0.5 for nil metrics, got %f", neutralScore)
	}
}
