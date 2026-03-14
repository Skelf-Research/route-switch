package gateway

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/skelf-research/route-switch/internal/analytics"
	"github.com/skelf-research/route-switch/internal/config"
	"github.com/skelf-research/route-switch/internal/core"
	"github.com/skelf-research/route-switch/internal/models"
	"github.com/skelf-research/route-switch/internal/storage/dataset"
)

// Gateway manages the complete route-switch functionality including
// prompt optimization, model switching, and load balancing
type Gateway struct {
	registry     *PromptRegistry
	loadBalancer *LoadBalancer
	proxy        *ProxyServer
	service      *core.Service

	// Background optimization
	bgOptimizer *BackgroundOptimizer
	ctx         context.Context
	cancel      context.CancelFunc

	mu sync.RWMutex
}

// GatewayConfig holds configuration for the gateway
type GatewayConfig struct {
	Addr                 string
	LoadBalancerStrategy LoadBalancerStrategy
	OptimizationEnabled  bool
	OptimizationInterval time.Duration
}

// NewGateway creates a new gateway instance
func NewGateway(serviceConfig *core.ServiceConfig, gatewayConfig *GatewayConfig, appConfig *config.Config, datasetStore dataset.Store, analyticsStore analytics.Store) (*Gateway, error) {
	registry := NewPromptRegistry()
	loadBalancer := NewLoadBalancer(registry, gatewayConfig.LoadBalancerStrategy)

	proxy := NewProxyServer(registry, loadBalancer, gatewayConfig.Addr, datasetStore, analyticsStore)

	// Create context for background operations
	ctx, cancel := context.WithCancel(context.Background())

	gateway := &Gateway{
		registry:     registry,
		loadBalancer: loadBalancer,
		proxy:        proxy,
		service:      core.NewService(serviceConfig),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Initialize prompt combinations from configuration
	if err := gateway.loadCombinationsFromConfig(appConfig); err != nil {
		return nil, fmt.Errorf("failed to load combinations from config: %w", err)
	}

	// Initialize background optimizer if enabled
	if gatewayConfig.OptimizationEnabled {
		gateway.bgOptimizer = NewBackgroundOptimizer(gateway, gatewayConfig.OptimizationInterval)
	}

	return gateway, nil
}

// loadCombinationsFromConfig loads prompt combinations from the application configuration
func (g *Gateway) loadCombinationsFromConfig(appConfig *config.Config) error {
	for _, comboConfig := range appConfig.Gateway.Combinations {
		if !comboConfig.Enabled {
			continue
		}

		// If the prompt needs optimization (not already optimized), run optimization
		optimizedPrompt := comboConfig.Prompt
		if comboConfig.Metadata == nil {
			comboConfig.Metadata = make(map[string]interface{})
		}

		if comboConfig.Metadata["optimized"] == nil || comboConfig.Metadata["optimized"] == false {
			result, err := g.service.OptimizePrompt(comboConfig.Prompt, comboConfig.Model)
			if err != nil {
				fmt.Printf("Warning: Failed to optimize prompt for %s, using original: %v\n", comboConfig.Name, err)
				optimizedPrompt = comboConfig.Prompt
			} else {
				optimizedPrompt = result.OptimizedPrompt
				comboConfig.Metadata["optimized"] = true
			}
		}

		templateID := comboConfig.TemplateID
		if templateID == "" {
			if metaVal, ok := comboConfig.Metadata["template_id"].(string); ok && metaVal != "" {
				templateID = metaVal
			} else {
				templateID = comboConfig.ID
			}
		}
		if comboConfig.Metadata == nil {
			comboConfig.Metadata = map[string]interface{}{}
		}
		comboConfig.Metadata["template_id"] = templateID

		// Create the combination
		combination := &PromptCombination{
			ID:         comboConfig.ID,
			Name:       comboConfig.Name,
			TemplateID: templateID,
			Prompt:     optimizedPrompt,
			Model:      comboConfig.Model,
			Provider:   comboConfig.Provider,
			Weight:     comboConfig.Weight,
			CreatedAt:  time.Now(),
			Performance: &PerformanceMetrics{
				SuccessRate: 1.0, // Start with 100% success rate
			},
			Metadata: comboConfig.Metadata,
		}

		// Add to registry
		if err := g.registry.AddCombination(combination); err != nil {
			return fmt.Errorf("failed to add combination %s: %w", comboConfig.ID, err)
		}

		// If this is a primary model, we might want to track it specially
		if comboConfig.IsPrimary {
			// For now we just log it, but we could add special handling for primary models
			fmt.Printf("Primary model combination loaded: %s (Model: %s)\n", comboConfig.Name, comboConfig.Model)
		}
	}

	return nil
}

// RegisterProvider registers a model provider with the gateway
func (g *Gateway) RegisterProvider(name string, provider models.ModelProvider) {
	g.proxy.RegisterProvider(name, provider)
}

// AddPromptCombination adds a new prompt+model combination to the gateway
func (g *Gateway) AddPromptCombination(prompt, modelName, providerName, name string) error {
	// First, optimize the prompt for the specific model
	optimizedResult, err := g.service.OptimizePrompt(prompt, modelName)
	if err != nil {
		return fmt.Errorf("failed to optimize prompt: %w", err)
	}

	// Create the combination
	combination := &PromptCombination{
		ID:         fmt.Sprintf("%s-%s-%d", name, modelName, time.Now().Unix()),
		Name:       name,
		TemplateID: fmt.Sprintf("%s-template", name),
		Prompt:     optimizedResult.OptimizedPrompt,
		Model:      modelName,
		Provider:   providerName,
		Weight:     10, // Default weight
		CreatedAt:  time.Now(),
		Performance: &PerformanceMetrics{
			SuccessRate: 1.0, // Start with 100% success rate
		},
		Metadata: map[string]interface{}{
			"original_prompt": prompt,
			"template_id":     fmt.Sprintf("%s-template", name),
		},
	}

	return g.registry.AddCombination(combination)
}

// Start starts the gateway services
func (g *Gateway) Start() error {
	// Start background optimizer if enabled
	if g.bgOptimizer != nil {
		go g.bgOptimizer.Start()
	}

	// Start the proxy server
	return g.proxy.Start()
}

// Stop stops the gateway services
func (g *Gateway) Stop() {
	g.cancel()

	// Stop background optimizer if running
	if g.bgOptimizer != nil {
		g.bgOptimizer.Stop()
	}
}

// GetActiveCombinations returns all active prompt+model combinations
func (g *Gateway) GetActiveCombinations() []*PromptCombination {
	return g.registry.GetActiveCombinations()
}

// UpdateCombinationWeight updates the weight of a combination for load balancing
func (g *Gateway) UpdateCombinationWeight(id string, weight int) error {
	combination, exists := g.registry.GetCombination(id)
	if !exists {
		return models.ErrNotFound
	}

	combination.Weight = weight
	return nil // The registry stores the pointer, so this updates it in place
}

// BackgroundOptimizer handles automatic optimization of prompt combinations
type BackgroundOptimizer struct {
	gateway  *Gateway
	interval time.Duration
	ticker   *time.Ticker
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewBackgroundOptimizer creates a new background optimizer
func NewBackgroundOptimizer(gateway *Gateway, interval time.Duration) *BackgroundOptimizer {
	return &BackgroundOptimizer{
		gateway:  gateway,
		interval: interval,
	}
}

// Start starts the background optimization process
func (bo *BackgroundOptimizer) Start() {
	bo.ticker = time.NewTicker(bo.interval)
	bo.ctx, bo.cancel = context.WithCancel(context.Background())

	bo.wg.Add(1)
	go func() {
		defer bo.wg.Done()
		for {
			select {
			case <-bo.ticker.C:
				bo.optimize()
			case <-bo.ctx.Done():
				bo.ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the background optimization process
func (bo *BackgroundOptimizer) Stop() {
	if bo.ticker != nil {
		bo.ticker.Stop()
		bo.cancel()
	}
	bo.wg.Wait()
}

// optimize runs the optimization process for all combinations
func (bo *BackgroundOptimizer) optimize() {
	combinations := bo.gateway.GetActiveCombinations()

	for _, combination := range combinations {
		// Check if the combination should be optimized (e.g., based on last optimization time)
		if time.Since(combination.LastOptimized) > 24*time.Hour { // Optimize if not optimized in last 24 hours
			bo.optimizeCombination(combination)
		}
	}
}

// optimizeCombination optimizes a specific prompt+model combination
func (bo *BackgroundOptimizer) optimizeCombination(combination *PromptCombination) {
	originalPrompt, ok := combination.Metadata["original_prompt"].(string)
	if !ok {
		// If we don't have the original prompt, skip optimization
		return
	}

	// Re-optimize the prompt for the same model
	optimizedResult, err := bo.gateway.service.OptimizePrompt(originalPrompt, combination.Model)
	if err != nil {
		// Log error but continue with other combinations
		return
	}

	// Update the combination with the newly optimized prompt
	combination.Prompt = optimizedResult.OptimizedPrompt
	combination.LastOptimized = time.Now()
}

// UpdateStrategy updates the load balancing strategy
func (g *Gateway) UpdateStrategy(strategy LoadBalancerStrategy) {
	g.loadBalancer.UpdateStrategy(strategy)
}

// GetStrategy returns the current load balancing strategy
func (g *Gateway) GetStrategy() LoadBalancerStrategy {
	return g.loadBalancer.GetStrategy()
}

// PerformModelSwitching finds alternative models for a given prompt
func (g *Gateway) PerformModelSwitching(prompt, baseModel string, providers []string) error {
	// Find the best model for the given prompt
	result, err := g.service.FindBestModel(prompt, baseModel)
	if err != nil {
		return fmt.Errorf("failed to find best model: %w", err)
	}

	// Create a new combination for this best model
	combination := &PromptCombination{
		ID:         fmt.Sprintf("switched-%s-%s-%d", result.Model, baseModel, time.Now().Unix()),
		Name:       fmt.Sprintf("switched-from-%s", baseModel),
		TemplateID: fmt.Sprintf("switched-%s", baseModel),
		Prompt:     result.OptimizedPrompt,
		Model:      result.Model,
		Provider:   g.identifyProvider(result.Model), // This would need to be implemented
		Weight:     5,                                // Lower weight initially
		CreatedAt:  time.Now(),
		Performance: &PerformanceMetrics{
			SuccessRate: 1.0,
		},
		Metadata: map[string]interface{}{
			"original_prompt":  prompt,
			"switching_reason": "model switching for cost/performance",
			"template_id":      fmt.Sprintf("switched-%s", baseModel),
		},
	}

	return g.registry.AddCombination(combination)
}

// identifyProvider would identify which provider a model belongs to
// This is a placeholder that would need proper implementation
func (g *Gateway) identifyProvider(modelName string) string {
	// This would look up the provider based on the model name
	// For now, returning a generic provider
	return "auto-detected"
}
