package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// LiteLLMPricingURL is the URL to fetch model pricing data from LiteLLM
const LiteLLMPricingURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"

// ModelPricing holds pricing information for a model from LiteLLM
type ModelPricing struct {
	InputCostPerToken             float64 `json:"input_cost_per_token"`
	OutputCostPerToken            float64 `json:"output_cost_per_token"`
	MaxInputTokens                int     `json:"max_input_tokens"`
	MaxOutputTokens               int     `json:"max_output_tokens"`
	MaxTokens                     int     `json:"max_tokens"`
	LiteLLMProvider               string  `json:"litellm_provider"`
	Mode                          string  `json:"mode"`
	CacheCreationInputTokenCost   float64 `json:"cache_creation_input_token_cost,omitempty"`
	CacheReadInputTokenCost       float64 `json:"cache_read_input_token_cost,omitempty"`
	SupportsVision                bool    `json:"supports_vision,omitempty"`
	SupportsFunctionCalling       bool    `json:"supports_function_calling,omitempty"`
	SupportsParallelFunctionCalling bool  `json:"supports_parallel_function_calling,omitempty"`
}

// CostCalculator provides methods for calculating model costs using LiteLLM pricing
type CostCalculator struct {
	mu             sync.RWMutex
	pricingData    map[string]ModelPricing
	lastFetch      time.Time
	cacheDuration  time.Duration
	httpClient     *http.Client
	fallbackPrices map[string]ModelPricing
}

// CostCalculatorOption is a functional option for configuring CostCalculator
type CostCalculatorOption func(*CostCalculator)

// WithCacheDuration sets the cache duration for pricing data
func WithCacheDuration(d time.Duration) CostCalculatorOption {
	return func(c *CostCalculator) {
		c.cacheDuration = d
	}
}

// WithHTTPClient sets a custom HTTP client for fetching pricing data
func WithHTTPClient(client *http.Client) CostCalculatorOption {
	return func(c *CostCalculator) {
		c.httpClient = client
	}
}

// NewCostCalculator creates a new instance of CostCalculator
func NewCostCalculator(opts ...CostCalculatorOption) *CostCalculator {
	c := &CostCalculator{
		pricingData:   make(map[string]ModelPricing),
		cacheDuration: 24 * time.Hour, // Default cache for 24 hours
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		fallbackPrices: initFallbackPrices(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// initFallbackPrices returns hardcoded fallback prices for common models
// These are used when LiteLLM pricing feed is unavailable
func initFallbackPrices() map[string]ModelPricing {
	return map[string]ModelPricing{
		// OpenAI models
		"gpt-4o": {
			InputCostPerToken:  0.0000025,  // $2.50 per 1M tokens
			OutputCostPerToken: 0.00001,    // $10 per 1M tokens
			MaxInputTokens:     128000,
			MaxOutputTokens:    16384,
			LiteLLMProvider:    "openai",
			Mode:               "chat",
		},
		"gpt-4o-mini": {
			InputCostPerToken:  0.00000015, // $0.15 per 1M tokens
			OutputCostPerToken: 0.0000006,  // $0.60 per 1M tokens
			MaxInputTokens:     128000,
			MaxOutputTokens:    16384,
			LiteLLMProvider:    "openai",
			Mode:               "chat",
		},
		"gpt-4-turbo": {
			InputCostPerToken:  0.00001,    // $10 per 1M tokens
			OutputCostPerToken: 0.00003,    // $30 per 1M tokens
			MaxInputTokens:     128000,
			MaxOutputTokens:    4096,
			LiteLLMProvider:    "openai",
			Mode:               "chat",
		},
		"gpt-4": {
			InputCostPerToken:  0.00003,    // $30 per 1M tokens
			OutputCostPerToken: 0.00006,    // $60 per 1M tokens
			MaxInputTokens:     8192,
			MaxOutputTokens:    8192,
			LiteLLMProvider:    "openai",
			Mode:               "chat",
		},
		"gpt-3.5-turbo": {
			InputCostPerToken:  0.0000005,  // $0.50 per 1M tokens
			OutputCostPerToken: 0.0000015,  // $1.50 per 1M tokens
			MaxInputTokens:     16385,
			MaxOutputTokens:    4096,
			LiteLLMProvider:    "openai",
			Mode:               "chat",
		},
		"o1": {
			InputCostPerToken:  0.000015,   // $15 per 1M tokens
			OutputCostPerToken: 0.00006,    // $60 per 1M tokens
			MaxInputTokens:     200000,
			MaxOutputTokens:    100000,
			LiteLLMProvider:    "openai",
			Mode:               "chat",
		},
		"o1-mini": {
			InputCostPerToken:  0.000003,   // $3 per 1M tokens
			OutputCostPerToken: 0.000012,   // $12 per 1M tokens
			MaxInputTokens:     128000,
			MaxOutputTokens:    65536,
			LiteLLMProvider:    "openai",
			Mode:               "chat",
		},
		// Anthropic models
		"claude-3-5-sonnet-20241022": {
			InputCostPerToken:  0.000003,   // $3 per 1M tokens
			OutputCostPerToken: 0.000015,   // $15 per 1M tokens
			MaxInputTokens:     200000,
			MaxOutputTokens:    8192,
			LiteLLMProvider:    "anthropic",
			Mode:               "chat",
		},
		"claude-3-opus-20240229": {
			InputCostPerToken:  0.000015,   // $15 per 1M tokens
			OutputCostPerToken: 0.000075,   // $75 per 1M tokens
			MaxInputTokens:     200000,
			MaxOutputTokens:    4096,
			LiteLLMProvider:    "anthropic",
			Mode:               "chat",
		},
		"claude-3-sonnet-20240229": {
			InputCostPerToken:  0.000003,   // $3 per 1M tokens
			OutputCostPerToken: 0.000015,   // $15 per 1M tokens
			MaxInputTokens:     200000,
			MaxOutputTokens:    4096,
			LiteLLMProvider:    "anthropic",
			Mode:               "chat",
		},
		"claude-3-haiku-20240307": {
			InputCostPerToken:  0.00000025, // $0.25 per 1M tokens
			OutputCostPerToken: 0.00000125, // $1.25 per 1M tokens
			MaxInputTokens:     200000,
			MaxOutputTokens:    4096,
			LiteLLMProvider:    "anthropic",
			Mode:               "chat",
		},
		// Google models
		"gemini-1.5-pro": {
			InputCostPerToken:  0.00000125, // $1.25 per 1M tokens
			OutputCostPerToken: 0.000005,   // $5 per 1M tokens
			MaxInputTokens:     2097152,
			MaxOutputTokens:    8192,
			LiteLLMProvider:    "google",
			Mode:               "chat",
		},
		"gemini-1.5-flash": {
			InputCostPerToken:  0.000000075, // $0.075 per 1M tokens
			OutputCostPerToken: 0.0000003,   // $0.30 per 1M tokens
			MaxInputTokens:     1048576,
			MaxOutputTokens:    8192,
			LiteLLMProvider:    "google",
			Mode:               "chat",
		},
		// Mistral models
		"mistral-large-latest": {
			InputCostPerToken:  0.000002, // $2 per 1M tokens
			OutputCostPerToken: 0.000006, // $6 per 1M tokens
			MaxInputTokens:     128000,
			MaxOutputTokens:    8192,
			LiteLLMProvider:    "mistral",
			Mode:               "chat",
		},
		"mistral-small-latest": {
			InputCostPerToken:  0.0000002, // $0.20 per 1M tokens
			OutputCostPerToken: 0.0000006, // $0.60 per 1M tokens
			MaxInputTokens:     32000,
			MaxOutputTokens:    8192,
			LiteLLMProvider:    "mistral",
			Mode:               "chat",
		},
		// Local/Ollama models (essentially free)
		"ollama/llama3": {
			InputCostPerToken:  0,
			OutputCostPerToken: 0,
			MaxInputTokens:     8192,
			MaxOutputTokens:    4096,
			LiteLLMProvider:    "ollama",
			Mode:               "chat",
		},
	}
}

// FetchPricing fetches and caches pricing data from LiteLLM
func (c *CostCalculator) FetchPricing() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if cache is still valid
	if time.Since(c.lastFetch) < c.cacheDuration && len(c.pricingData) > 0 {
		return nil
	}

	resp, err := c.httpClient.Get(LiteLLMPricingURL)
	if err != nil {
		return fmt.Errorf("failed to fetch LiteLLM pricing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LiteLLM pricing API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read LiteLLM pricing response: %w", err)
	}

	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(body, &rawData); err != nil {
		return fmt.Errorf("failed to parse LiteLLM pricing JSON: %w", err)
	}

	newPricingData := make(map[string]ModelPricing)
	for modelName, rawPricing := range rawData {
		// Skip the sample_spec entry
		if modelName == "sample_spec" {
			continue
		}

		var pricing ModelPricing
		if err := json.Unmarshal(rawPricing, &pricing); err != nil {
			// Skip malformed entries
			continue
		}
		newPricingData[modelName] = pricing
	}

	if len(newPricingData) == 0 {
		return fmt.Errorf("no valid pricing data found in LiteLLM response")
	}

	c.pricingData = newPricingData
	c.lastFetch = time.Now()

	return nil
}

// GetModelPricing returns pricing for a specific model
func (c *CostCalculator) GetModelPricing(modelName string) (ModelPricing, error) {
	// Try to fetch latest pricing (uses cache if available)
	if err := c.FetchPricing(); err != nil {
		// Fall through to fallback prices if fetch fails
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Normalize the model name for lookup
	normalizedName := normalizeModelName(modelName)

	// Try exact match first
	if pricing, ok := c.pricingData[modelName]; ok {
		return pricing, nil
	}

	// Try normalized name
	if pricing, ok := c.pricingData[normalizedName]; ok {
		return pricing, nil
	}

	// Try with common prefixes
	prefixes := []string{"openai/", "anthropic/", "google/", "mistral/", "ollama/", "cohere/"}
	for _, prefix := range prefixes {
		if pricing, ok := c.pricingData[prefix+normalizedName]; ok {
			return pricing, nil
		}
	}

	// Try partial match (for versioned model names)
	for name, pricing := range c.pricingData {
		if strings.Contains(strings.ToLower(name), strings.ToLower(normalizedName)) {
			return pricing, nil
		}
	}

	// Fall back to hardcoded prices
	if pricing, ok := c.fallbackPrices[normalizedName]; ok {
		return pricing, nil
	}

	// Try partial match in fallback
	for name, pricing := range c.fallbackPrices {
		if strings.Contains(strings.ToLower(normalizedName), strings.ToLower(name)) ||
			strings.Contains(strings.ToLower(name), strings.ToLower(normalizedName)) {
			return pricing, nil
		}
	}

	return ModelPricing{}, fmt.Errorf("pricing not found for model: %s", modelName)
}

// CalculateCost calculates the cost of using a model for a given number of tokens
func (c *CostCalculator) CalculateCost(modelName string, inputTokens, outputTokens int) (float64, error) {
	if inputTokens < 0 || outputTokens < 0 {
		return 0, fmt.Errorf("token counts cannot be negative: input=%d, output=%d", inputTokens, outputTokens)
	}

	pricing, err := c.GetModelPricing(modelName)
	if err != nil {
		return 0, fmt.Errorf("failed to get pricing for model %s: %w", modelName, err)
	}

	inputCost := float64(inputTokens) * pricing.InputCostPerToken
	outputCost := float64(outputTokens) * pricing.OutputCostPerToken
	totalCost := inputCost + outputCost

	return totalCost, nil
}

// CalculateCostWithCaching calculates cost including prompt caching costs
func (c *CostCalculator) CalculateCostWithCaching(modelName string, inputTokens, outputTokens, cacheWriteTokens, cacheReadTokens int) (float64, error) {
	if inputTokens < 0 || outputTokens < 0 || cacheWriteTokens < 0 || cacheReadTokens < 0 {
		return 0, fmt.Errorf("token counts cannot be negative")
	}

	pricing, err := c.GetModelPricing(modelName)
	if err != nil {
		return 0, fmt.Errorf("failed to get pricing for model %s: %w", modelName, err)
	}

	inputCost := float64(inputTokens) * pricing.InputCostPerToken
	outputCost := float64(outputTokens) * pricing.OutputCostPerToken
	cacheWriteCost := float64(cacheWriteTokens) * pricing.CacheCreationInputTokenCost
	cacheReadCost := float64(cacheReadTokens) * pricing.CacheReadInputTokenCost

	totalCost := inputCost + outputCost + cacheWriteCost + cacheReadCost

	return totalCost, nil
}

// FindCheapestModel determines the cheapest model from a list for a given token estimate
func (c *CostCalculator) FindCheapestModel(models []string, prompt string) (string, float64, error) {
	if len(models) == 0 {
		return "", 0, fmt.Errorf("no models provided")
	}

	// Estimate tokens from prompt (rough approximation: 4 chars per token)
	estimatedInputTokens := len(prompt) / 4
	if estimatedInputTokens < 1 {
		estimatedInputTokens = 1
	}
	// Assume similar output length
	estimatedOutputTokens := estimatedInputTokens

	var cheapestModel string
	cheapestCost := -1.0

	for _, model := range models {
		cost, err := c.CalculateCost(model, estimatedInputTokens, estimatedOutputTokens)
		if err != nil {
			// Skip models without pricing
			continue
		}

		if cheapestCost < 0 || cost < cheapestCost {
			cheapestCost = cost
			cheapestModel = model
		}
	}

	if cheapestModel == "" {
		return "", 0, fmt.Errorf("could not determine pricing for any of the provided models")
	}

	return cheapestModel, cheapestCost, nil
}

// GetMaxTokens returns the maximum token limits for a model
func (c *CostCalculator) GetMaxTokens(modelName string) (inputMax, outputMax int, err error) {
	pricing, err := c.GetModelPricing(modelName)
	if err != nil {
		return 0, 0, err
	}

	inputMax = pricing.MaxInputTokens
	outputMax = pricing.MaxOutputTokens

	// Use legacy MaxTokens field as fallback
	if inputMax == 0 && pricing.MaxTokens > 0 {
		inputMax = pricing.MaxTokens
	}
	if outputMax == 0 && pricing.MaxTokens > 0 {
		outputMax = pricing.MaxTokens / 2 // Assume half for output
	}

	return inputMax, outputMax, nil
}

// GetProviderForModel returns the provider name for a model
func (c *CostCalculator) GetProviderForModel(modelName string) (string, error) {
	pricing, err := c.GetModelPricing(modelName)
	if err != nil {
		return "", err
	}
	return pricing.LiteLLMProvider, nil
}

// normalizeModelName normalizes model names for consistent lookup
func normalizeModelName(name string) string {
	// Remove common prefixes
	name = strings.TrimPrefix(name, "openai/")
	name = strings.TrimPrefix(name, "anthropic/")
	name = strings.TrimPrefix(name, "google/")
	name = strings.TrimPrefix(name, "mistral/")
	name = strings.TrimPrefix(name, "ollama/")
	name = strings.TrimPrefix(name, "cohere/")
	name = strings.TrimPrefix(name, "azure/")

	return strings.ToLower(strings.TrimSpace(name))
}

// CostBreakdown provides a detailed cost breakdown
type CostBreakdown struct {
	InputCost       float64 `json:"input_cost"`
	OutputCost      float64 `json:"output_cost"`
	CacheWriteCost  float64 `json:"cache_write_cost,omitempty"`
	CacheReadCost   float64 `json:"cache_read_cost,omitempty"`
	TotalCost       float64 `json:"total_cost"`
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	CacheWriteTokens int    `json:"cache_write_tokens,omitempty"`
	CacheReadTokens  int    `json:"cache_read_tokens,omitempty"`
	Model           string  `json:"model"`
	Provider        string  `json:"provider"`
}

// CalculateCostBreakdown provides a detailed cost breakdown for a model call
func (c *CostCalculator) CalculateCostBreakdown(modelName string, inputTokens, outputTokens int) (*CostBreakdown, error) {
	pricing, err := c.GetModelPricing(modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing for model %s: %w", modelName, err)
	}

	inputCost := float64(inputTokens) * pricing.InputCostPerToken
	outputCost := float64(outputTokens) * pricing.OutputCostPerToken

	return &CostBreakdown{
		InputCost:    inputCost,
		OutputCost:   outputCost,
		TotalCost:    inputCost + outputCost,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Model:        modelName,
		Provider:     pricing.LiteLLMProvider,
	}, nil
}
