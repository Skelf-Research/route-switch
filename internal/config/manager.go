package config

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SimpleConfigManager implements the ConfigManager interface
type SimpleConfigManager struct {
	config *Config
}

// NewSimpleConfigManager creates a new SimpleConfigManager
func NewSimpleConfigManager() *SimpleConfigManager {
	return &SimpleConfigManager{
		config: &Config{
			ModelProviders: make(map[string]ProviderConfig),
			MiproV2: MiproV2Config{
				NumCandidates:            5,
				MaxBootstrappedDemos:     3,
				MaxLabeledDemos:          2,
				NumTrials:                10,
				MinibatchSize:            5,
				MinibatchFullEvalSteps:   3,
				NumInstructionCandidates: 3,
				EvaluationStrategy:       "Similarity",
			},
			Evaluation: EvaluationConfig{
				DefaultStrategy: "Similarity",
				Threshold:       0.7,
				MaxRetries:      3,
			},
			API: APIConfig{
				TimeoutSeconds: 30,
				MaxRetries:     3,
			},
			Dataset: DatasetConfig{
				BasePath:   "data/prompts",
				MaxRecords: 1000,
			},
			Analytics: AnalyticsConfig{
				Driver: "duckdb",
				Path:   "data/analytics/metrics.duckdb",
			},
		},
	}
}

// Load loads configuration from a file
func (scm *SimpleConfigManager) Load(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Try to parse as YAML first, then JSON if that fails
	if err := yaml.Unmarshal(data, scm.config); err != nil {
		// If YAML parsing fails, try JSON
		if err2 := json.Unmarshal(data, scm.config); err2 != nil {
			return fmt.Errorf("failed to parse config file as YAML (%v) or JSON (%v)", err, err2)
		}
	}

	return scm.Validate()
}

// Save saves configuration to a file
func (scm *SimpleConfigManager) Save(configPath string) error {
	data, err := yaml.Marshal(scm.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}

// GetConfig returns the current configuration
func (scm *SimpleConfigManager) GetConfig() *Config {
	return scm.config
}

// UpdateConfig updates configuration values
func (scm *SimpleConfigManager) UpdateConfig(updates map[string]interface{}) error {
	// Convert current config to map for easy updates
	configBytes, err := json.Marshal(scm.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config for update: %w", err)
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(configBytes, &configMap); err != nil {
		return fmt.Errorf("failed to unmarshal config for update: %w", err)
	}

	// Update the map with new values
	for key, value := range updates {
		configMap[key] = value
	}

	// Convert back to struct
	updatedBytes, err := json.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	var updatedConfig Config
	if err := json.Unmarshal(updatedBytes, &updatedConfig); err != nil {
		return fmt.Errorf("failed to unmarshal updated config: %w", err)
	}

	// Validate the updated config
	if err := updatedConfig.Validate(); err != nil {
		return fmt.Errorf("updated config validation failed: %w", err)
	}

	scm.config = &updatedConfig
	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.MiproV2.NumCandidates <= 0 {
		return fmt.Errorf("NumCandidates must be greater than 0")
	}
	if c.MiproV2.NumTrials <= 0 {
		return fmt.Errorf("NumTrials must be greater than 0")
	}
	if c.Evaluation.Threshold <= 0 || c.Evaluation.Threshold > 1 {
		return fmt.Errorf("Threshold must be between 0 and 1")
	}
	if c.API.TimeoutSeconds <= 0 {
		return fmt.Errorf("TimeoutSeconds must be greater than 0")
	}
	if c.Dataset.BasePath == "" {
		return fmt.Errorf("dataset base path must be provided")
	}
	if c.Dataset.MaxRecords <= 0 {
		return fmt.Errorf("dataset max records must be greater than 0")
	}
	if c.Analytics.Driver == "" {
		c.Analytics.Driver = "duckdb"
	}
	if c.Analytics.Driver == "duckdb" && c.Analytics.Path == "" {
		return fmt.Errorf("analytics path must be provided for duckdb driver")
	}

	return nil
}

// Validate validates the configuration
func (scm *SimpleConfigManager) Validate() error {
	return scm.config.Validate()
}
