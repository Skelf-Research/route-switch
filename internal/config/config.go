package config

// Config holds application configuration
type Config struct {
	// Model provider configurations
	ModelProviders map[string]ProviderConfig `json:"model_providers" yaml:"model_providers"`

	// MIPROV2 optimization settings
	MiproV2 MiproV2Config `json:"mipro_v2" yaml:"mipro_v2"`

	// Evaluation settings
	Evaluation EvaluationConfig `json:"evaluation" yaml:"evaluation"`

	// API configuration
	API APIConfig `json:"api" yaml:"api"`

	// Gateway configurations
	Gateway GatewayConfig `json:"gateway" yaml:"gateway"`

	// Dataset storage configuration
	Dataset DatasetConfig `json:"dataset" yaml:"dataset"`

	// Analytics configuration
	Analytics AnalyticsConfig `json:"analytics" yaml:"analytics"`
}

// GatewayConfig holds gateway-specific configuration
type GatewayConfig struct {
	Addr         string                    `json:"addr" yaml:"addr"`
	Strategy     string                    `json:"strategy" yaml:"strategy"`
	Combinations []PromptCombinationConfig `json:"combinations" yaml:"combinations"`
	Optimization OptimizationConfig        `json:"optimization" yaml:"optimization"`
	FallbackThreshold float64              `json:"fallback_threshold" yaml:"fallback_threshold"`
}

// PromptCombinationConfig holds configuration for a prompt+model combination
type PromptCombinationConfig struct {
	ID         string                 `json:"id" yaml:"id"`
	Name       string                 `json:"name" yaml:"name"`
	TemplateID string                 `json:"template_id" yaml:"template_id"`
	Prompt     string                 `json:"prompt" yaml:"prompt"`
	Model      string                 `json:"model" yaml:"model"`
	Provider   string                 `json:"provider" yaml:"provider"`
	IsPrimary  bool                   `json:"is_primary" yaml:"is_primary"`
	Weight     int                    `json:"weight" yaml:"weight"`
	Fallbacks  []string               `json:"fallbacks" yaml:"fallbacks"`
	Enabled    bool                   `json:"enabled" yaml:"enabled"`
	Metadata   map[string]interface{} `json:"metadata" yaml:"metadata"`
}

// OptimizationConfig holds background optimization settings
type OptimizationConfig struct {
	Enabled   bool `json:"enabled" yaml:"enabled"`
	Interval  int  `json:"interval_seconds" yaml:"interval_seconds"` // in seconds
	TargetRPS int  `json:"target_rps" yaml:"target_rps"`             // Target requests per second for optimization
}

// ProviderConfig holds configuration for a model provider
type ProviderConfig struct {
	APIKey    string                 `json:"api_key" yaml:"api_key"`
	BaseURL   string                 `json:"base_url" yaml:"base_url"`
	Models    []string               `json:"models" yaml:"models"`
	Options   map[string]interface{} `json:"options" yaml:"options"`
	RateLimit int                    `json:"rate_limit" yaml:"rate_limit"` // requests per minute
}

// DatasetConfig configures per-prompt dataset storage
type DatasetConfig struct {
	BasePath   string `json:"base_path" yaml:"base_path"`
	MaxRecords int    `json:"max_records" yaml:"max_records"`
}

// AnalyticsConfig configures analytics persistence.
type AnalyticsConfig struct {
	Driver string `json:"driver" yaml:"driver"`
	Path   string `json:"path" yaml:"path"`
}

// MiproV2Config holds configuration for MIPROv2 optimization
type MiproV2Config struct {
	NumCandidates            int    `json:"num_candidates" yaml:"num_candidates"`
	MaxBootstrappedDemos     int    `json:"max_bootstrapped_demos" yaml:"max_bootstrapped_demos"`
	MaxLabeledDemos          int    `json:"max_labeled_demos" yaml:"max_labeled_demos"`
	NumTrials                int    `json:"num_trials" yaml:"num_trials"`
	MinibatchSize            int    `json:"minibatch_size" yaml:"minibatch_size"`
	MinibatchFullEvalSteps   int    `json:"minibatch_full_eval_steps" yaml:"minibatch_full_eval_steps"`
	NumInstructionCandidates int    `json:"num_instruction_candidates" yaml:"num_instruction_candidates"`
	EvaluationStrategy       string `json:"evaluation_strategy" yaml:"evaluation_strategy"`
}

// EvaluationConfig holds configuration for evaluation strategies
type EvaluationConfig struct {
	DefaultStrategy string  `json:"default_strategy" yaml:"default_strategy"`
	Threshold       float64 `json:"threshold" yaml:"threshold"`
	MaxRetries      int     `json:"max_retries" yaml:"max_retries"`
}

// APIConfig holds configuration for API calls
type APIConfig struct {
	TimeoutSeconds int `json:"timeout_seconds" yaml:"timeout_seconds"`
	MaxRetries     int `json:"max_retries" yaml:"max_retries"`
}

// ConfigManager defines how configuration should be loaded and managed
type ConfigManager interface {
	Load(configPath string) error
	Save(configPath string) error
	GetConfig() *Config
	UpdateConfig(updates map[string]interface{}) error
	Validate() error
}
