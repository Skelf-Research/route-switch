package models

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/skelf-research/route-switch/internal/utils"
	gollm "github.com/teilomillet/gollm"
	gollmconfig "github.com/teilomillet/gollm/config"
)

// GollmProvider implements the ModelProvider interface using the gollm library
type GollmProvider struct {
	providerType    string
	config          map[string]interface{}
	providerConfigs map[string]map[string]interface{}
	modelAliases    map[string]string
	rateLimiters    map[string]*rateLimiter
	llms            map[string]gollm.LLM
	costCalculator  *utils.CostCalculator
}

// NewGollmProvider creates a new provider using gollm
func NewGollmProvider() *GollmProvider {
	return &GollmProvider{
		llms:            make(map[string]gollm.LLM),
		costCalculator:  utils.NewCostCalculator(),
		providerConfigs: make(map[string]map[string]interface{}),
		modelAliases:    make(map[string]string),
		rateLimiters:    make(map[string]*rateLimiter),
	}
}

// Name returns the name of the provider
func (g *GollmProvider) Name() string {
	return "GollmProvider"
}

// ListModels returns all configured models across providers.
func (g *GollmProvider) ListModels() ([]Model, error) {
	configs := g.providerConfigs
	if len(configs) == 0 && g.config != nil {
		if generic, ok := g.config["providers"].(map[string]interface{}); ok {
			configs = make(map[string]map[string]interface{})
			for name, raw := range generic {
				if typed, ok := raw.(map[string]interface{}); ok {
					configs[name] = typed
				}
			}
		}
	}

	if len(configs) == 0 {
		return []Model{}, nil
	}

	seen := make(map[string]struct{})
	var models []Model

	for _, cfg := range configs {
		for _, modelName := range toStringSlice(cfg["models"]) {
			if _, ok := seen[modelName]; ok {
				continue
			}
			modelInfo, err := g.GetModel(modelName)
			if err != nil {
				continue
			}
			models = append(models, modelInfo)
			seen[modelName] = struct{}{}
		}
	}

	return models, nil
}

// GetModel returns a specific model by name
func (g *GollmProvider) GetModel(name string) (Model, error) {
	if name == "" {
		return Model{}, fmt.Errorf("model name cannot be empty")
	}

	// Check if LLM instance exists, if not create it
	if _, exists := g.llms[name]; !exists {
		// Create a default configuration for this model
		// We'll need to determine the provider type from the model name
		providerType := g.detectProviderType(name)

		llm, err := g.createLLMInstance(providerType, name, g.config)
		if err != nil {
			return Model{}, fmt.Errorf("failed to create LLM instance for model %s: %w", name, err)
		}
		g.llms[name] = llm
	}

	// Get pricing from LiteLLM-based cost calculator
	pricing, err := g.costCalculator.GetModelPricing(name)
	if err != nil {
		// Fall back to basic estimation if pricing not available
		return Model{
			Name:         name,
			Provider:     g.detectProviderType(name),
			CostPerToken: 0.000002, // Default fallback
			MaxTokens:    4096,
			Description:  fmt.Sprintf("%s model via Gollm unified API", name),
		}, nil
	}

	maxTokens := pricing.MaxInputTokens
	if maxTokens == 0 {
		maxTokens = pricing.MaxTokens
	}
	if maxTokens == 0 {
		maxTokens = 4096 // Default
	}

	// Use average of input and output cost for CostPerToken (backward compat)
	avgCostPerToken := (pricing.InputCostPerToken + pricing.OutputCostPerToken) / 2

	return Model{
		Name:         name,
		Provider:     pricing.LiteLLMProvider,
		CostPerToken: avgCostPerToken,
		MaxTokens:    maxTokens,
		Description:  fmt.Sprintf("%s model via Gollm unified API", name),
	}, nil
}

// CallModel calls the model with the given prompt
func (g *GollmProvider) CallModel(modelName, prompt string) (string, error) {
	if modelName == "" {
		return "", fmt.Errorf("model name cannot be empty")
	}
	if prompt == "" {
		return "", fmt.Errorf("prompt cannot be empty")
	}

	// Get or create the LLM instance
	llm, exists := g.llms[modelName]
	if !exists {
		model, err := g.GetModel(modelName)
		if err != nil {
			return "", fmt.Errorf("failed to get model config: %w", err)
		}
		llm, err = g.createLLMInstance(model.Provider, modelName, g.config)
		if err != nil {
			return "", fmt.Errorf("failed to create LLM instance: %w", err)
		}
		g.llms[modelName] = llm
	}

	// Create a prompt using gollm's prompt system
	gollmPrompt := gollm.NewPrompt(prompt)

	if limiter := g.getRateLimiter(modelName); limiter != nil {
		if err := limiter.Allow(); err != nil {
			return "", err
		}
	}

	// Generate response
	response, err := llm.Generate(context.Background(), gollmPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate response from model %s: %w", modelName, err)
	}

	return response, nil
}

// EstimateCost estimates the cost for a given number of tokens using LiteLLM pricing
func (g *GollmProvider) EstimateCost(modelName string, inputTokens, outputTokens int) (float64, error) {
	if modelName == "" {
		return 0, fmt.Errorf("model name cannot be empty")
	}
	if inputTokens < 0 || outputTokens < 0 {
		return 0, fmt.Errorf("token counts cannot be negative: input=%d, output=%d", inputTokens, outputTokens)
	}

	cost, err := g.costCalculator.CalculateCost(modelName, inputTokens, outputTokens)
	if err != nil {
		// If exact pricing not available, use fallback
		model, modelErr := g.GetModel(modelName)
		if modelErr != nil {
			return 0, fmt.Errorf("failed to estimate cost for model %s: %w", modelName, err)
		}
		totalTokens := inputTokens + outputTokens
		return float64(totalTokens) * model.CostPerToken, nil
	}

	return cost, nil
}

// GetTokenCount returns an estimated token count
func (g *GollmProvider) GetTokenCount(text string) (int, error) {
	if len(text) == 0 {
		return 0, nil
	}
	// Using a simple estimation (4 characters per token)
	// In a real implementation, we'd use proper tokenizers per provider
	return len([]rune(text)) / 4, nil
}

// Initialize sets up the provider with configuration
func (g *GollmProvider) Initialize(config map[string]interface{}) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	g.config = config
	if providers, ok := config["providers"].(map[string]map[string]interface{}); ok {
		g.providerConfigs = providers
	} else if generic, ok := config["providers"].(map[string]interface{}); ok {
		normalized := make(map[string]map[string]interface{})
		for name, raw := range generic {
			if typed, ok := raw.(map[string]interface{}); ok {
				normalized[name] = typed
			}
		}
		if len(normalized) > 0 {
			g.providerConfigs = normalized
		}
	}
	g.setupRateLimiters()
	return nil
}

// Close performs cleanup operations
func (g *GollmProvider) Close() error {
	// Clean up LLM instances if needed
	for _, llm := range g.llms {
		// gollm doesn't have explicit close methods, so we just clear the map
		_ = llm // Use the variable to avoid unused error in a real implementation
	}
	g.llms = make(map[string]gollm.LLM)
	return nil
}

// GetCostCalculator returns the underlying cost calculator for advanced operations
func (g *GollmProvider) GetCostCalculator() *utils.CostCalculator {
	return g.costCalculator
}

// GetCostBreakdown provides a detailed cost breakdown for a model call
func (g *GollmProvider) GetCostBreakdown(modelName string, inputTokens, outputTokens int) (*utils.CostBreakdown, error) {
	return g.costCalculator.CalculateCostBreakdown(modelName, inputTokens, outputTokens)
}

// GetMaxTokens returns the maximum token limits for a model
func (g *GollmProvider) GetMaxTokens(modelName string) (inputMax, outputMax int, err error) {
	return g.costCalculator.GetMaxTokens(modelName)
}

// detectProviderType determines the provider type from the model name
func (g *GollmProvider) detectProviderType(modelName string) string {
	// First, try to get provider from cost calculator (uses LiteLLM data)
	if g.costCalculator != nil {
		provider, err := g.costCalculator.GetProviderForModel(modelName)
		if err == nil && provider != "" {
			return provider
		}
	}

	// Fall back to heuristic based on model names
	return g.detectProviderTypeHeuristic(modelName)
}

// detectProviderTypeHeuristic uses pattern matching to detect provider type
func (g *GollmProvider) detectProviderTypeHeuristic(modelName string) string {
	switch {
	case containsAny(modelName, []string{"gpt-", "o1-", "text-", "dall-e", "tts-", "whisper-"}):
		return "openai"
	case containsAny(modelName, []string{"claude-"}):
		return "anthropic"
	case containsAny(modelName, []string{"ollama", "llama", "mistral", "mixtral", "codellama", "phi", "gemma", "qwen", "deepseek"}):
		return "ollama"
	case containsAny(modelName, []string{"command-", "c4-"}):
		return "cohere"
	case containsAny(modelName, []string{"gemini-"}):
		return "google"
	case containsAny(modelName, []string{"huggingface-", "hf-"}):
		return "huggingface"
	default:
		// Default to openai for unknown models, which is common
		return "openai"
	}
}

// createLLMInstance creates an LLM instance for the given provider and model
func (g *GollmProvider) createLLMInstance(providerType, modelName string, config map[string]interface{}) (gollm.LLM, error) {
	if providerType == "" {
		return nil, fmt.Errorf("provider type cannot be empty")
	}
	if modelName == "" {
		return nil, fmt.Errorf("model name cannot be empty")
	}

	var opts []gollm.ConfigOption

	// Set provider type
	opts = append(opts, gollmconfig.SetProvider(providerType))

	// Set model name
	opts = append(opts, gollmconfig.SetModel(modelName))

	// Add provider-specific API key
	var apiKey string
	var hasAPIKey bool

	switch providerType {
	case "anthropic":
		apiKey, hasAPIKey = g.getAPIKey(config, "anthropic_api_key", "api_key")
	case "ollama":
		// Ollama can work without an API key in some setups
		if host, ok := config["ollama_host"].(string); ok && host != "" {
			opts = append(opts, gollmconfig.SetOllamaEndpoint(host))
		}
		apiKey, hasAPIKey = g.getAPIKey(config, "ollama_api_key", "api_key")
	case "cohere":
		apiKey, hasAPIKey = g.getAPIKey(config, "cohere_api_key", "api_key")
	case "google":
		apiKey, hasAPIKey = g.getAPIKey(config, "google_api_key", "api_key")
	case "huggingface":
		apiKey, hasAPIKey = g.getAPIKey(config, "huggingface_api_key", "api_key")
	case "mistral":
		apiKey, hasAPIKey = g.getAPIKey(config, "mistral_api_key", "api_key")
	case "openai":
		fallthrough
	default:
		apiKey, hasAPIKey = g.getAPIKey(config, "openai_api_key", "api_key")
	}

	if hasAPIKey && apiKey != "" {
		opts = append(opts, gollmconfig.SetAPIKey(apiKey))
	}

	// Create the LLM instance
	llm, err := gollm.NewLLM(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM instance for provider %s, model %s: %w", providerType, modelName, err)
	}

	return llm, nil
}

// getAPIKey extracts API key from config, trying primary key first then fallback
func (g *GollmProvider) getAPIKey(config map[string]interface{}, primaryKey, fallbackKey string) (string, bool) {
	if config == nil {
		return "", false
	}

	// Try primary key first
	if apiKey, ok := config[primaryKey].(string); ok && apiKey != "" {
		return apiKey, true
	}

	// Try fallback key
	if apiKey, ok := config[fallbackKey].(string); ok && apiKey != "" {
		return apiKey, true
	}

	return "", false
}

// containsAny checks if the string contains any of the substrings
func containsAny(s string, substrings []string) bool {
	sLower := strings.ToLower(s)
	for _, substr := range substrings {
		if strings.Contains(sLower, strings.ToLower(substr)) {
			return true
		}
	}
	return false
}

func toStringSlice(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		var out []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}

func (g *GollmProvider) getRateLimiter(modelName string) *rateLimiter {
	alias := g.getModelAlias(modelName)
	if alias == "" {
		alias = g.detectProviderType(modelName)
	}
	return g.rateLimiters[strings.ToLower(alias)]
}

func (g *GollmProvider) getModelAlias(modelName string) string {
	if alias, ok := g.modelAliases[modelName]; ok {
		return alias
	}
	for alias, cfg := range g.providerConfigs {
		for _, m := range toStringSlice(cfg["models"]) {
			g.modelAliases[m] = alias
		}
	}
	return g.modelAliases[modelName]
}

func (g *GollmProvider) setupRateLimiters() {
	for alias, cfg := range g.providerConfigs {
		rate := toInt(cfg["rate_limit"])
		if rate <= 0 {
			continue
		}
		g.rateLimiters[strings.ToLower(alias)] = newRateLimiter(rate)
	}
}

func toInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

type rateLimiter struct {
	mu        sync.Mutex
	capacity  float64
	tokens    float64
	fillRate  float64
	lastCheck time.Time
}

func newRateLimiter(requestsPerMinute int) *rateLimiter {
	return &rateLimiter{
		capacity:  float64(requestsPerMinute),
		tokens:    float64(requestsPerMinute),
		fillRate:  float64(requestsPerMinute) / 60.0,
		lastCheck: time.Now(),
	}
}

func (r *rateLimiter) Allow() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastCheck).Seconds()
	r.lastCheck = now
	r.tokens += elapsed * r.fillRate
	if r.tokens > r.capacity {
		r.tokens = r.capacity
	}

	if r.tokens < 1 {
		return ErrRateLimited
	}

	r.tokens -= 1
	return nil
}
