package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewSimpleConfigManager(t *testing.T) {
	mgr := NewSimpleConfigManager()
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}

	cfg := mgr.GetConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	// Verify defaults
	if cfg.MiproV2.NumCandidates != 5 {
		t.Errorf("expected NumCandidates=5, got %d", cfg.MiproV2.NumCandidates)
	}
	if cfg.MiproV2.NumTrials != 10 {
		t.Errorf("expected NumTrials=10, got %d", cfg.MiproV2.NumTrials)
	}
	if cfg.Evaluation.DefaultStrategy != "Similarity" {
		t.Errorf("expected DefaultStrategy=Similarity, got %s", cfg.Evaluation.DefaultStrategy)
	}
	if cfg.Evaluation.Threshold != 0.7 {
		t.Errorf("expected Threshold=0.7, got %f", cfg.Evaluation.Threshold)
	}
	if cfg.API.TimeoutSeconds != 30 {
		t.Errorf("expected TimeoutSeconds=30, got %d", cfg.API.TimeoutSeconds)
	}
	if cfg.Dataset.BasePath != "data/prompts" {
		t.Errorf("expected BasePath=data/prompts, got %s", cfg.Dataset.BasePath)
	}
	if cfg.Analytics.Driver != "duckdb" {
		t.Errorf("expected Driver=duckdb, got %s", cfg.Analytics.Driver)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid default config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "invalid NumCandidates",
			modify: func(c *Config) {
				c.MiproV2.NumCandidates = 0
			},
			wantErr: true,
			errMsg:  "NumCandidates must be greater than 0",
		},
		{
			name: "invalid NumTrials",
			modify: func(c *Config) {
				c.MiproV2.NumTrials = 0
			},
			wantErr: true,
			errMsg:  "NumTrials must be greater than 0",
		},
		{
			name: "threshold too low",
			modify: func(c *Config) {
				c.Evaluation.Threshold = 0
			},
			wantErr: true,
			errMsg:  "Threshold must be between 0 and 1",
		},
		{
			name: "threshold too high",
			modify: func(c *Config) {
				c.Evaluation.Threshold = 1.5
			},
			wantErr: true,
			errMsg:  "Threshold must be between 0 and 1",
		},
		{
			name: "invalid TimeoutSeconds",
			modify: func(c *Config) {
				c.API.TimeoutSeconds = 0
			},
			wantErr: true,
			errMsg:  "TimeoutSeconds must be greater than 0",
		},
		{
			name: "empty dataset base path",
			modify: func(c *Config) {
				c.Dataset.BasePath = ""
			},
			wantErr: true,
			errMsg:  "dataset base path must be provided",
		},
		{
			name: "invalid MaxRecords",
			modify: func(c *Config) {
				c.Dataset.MaxRecords = 0
			},
			wantErr: true,
			errMsg:  "dataset max records must be greater than 0",
		},
		{
			name: "empty analytics path for duckdb",
			modify: func(c *Config) {
				c.Analytics.Driver = "duckdb"
				c.Analytics.Path = ""
			},
			wantErr: true,
			errMsg:  "analytics path must be provided for duckdb driver",
		},
		{
			name: "fallback threshold too high",
			modify: func(c *Config) {
				c.Gateway.FallbackThreshold = 1.5
			},
			wantErr: true,
			errMsg:  "gateway fallback threshold must be between 0 and 1",
		},
		{
			name: "fallback threshold negative",
			modify: func(c *Config) {
				c.Gateway.FallbackThreshold = -0.1
			},
			wantErr: true,
			errMsg:  "gateway fallback threshold must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewSimpleConfigManager()
			tt.modify(mgr.config)
			err := mgr.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoadYAML(t *testing.T) {
	yamlContent := `
model_providers:
  openai:
    api_key: "test-key"
    models:
      - gpt-4
      - gpt-3.5-turbo
mipro_v2:
  num_candidates: 10
  num_trials: 20
  max_bootstrapped_demos: 5
  max_labeled_demos: 3
  minibatch_size: 10
  minibatch_full_eval_steps: 5
  num_instruction_candidates: 5
  evaluation_strategy: "Similarity"
evaluation:
  default_strategy: "Similarity"
  threshold: 0.8
  max_retries: 5
api:
  timeout_seconds: 60
  max_retries: 5
dataset:
  base_path: "custom/path"
  max_records: 500
analytics:
  driver: "duckdb"
  path: "data/analytics.duckdb"
gateway:
  fallback_threshold: 0.5
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	mgr := NewSimpleConfigManager()
	if err := mgr.Load(configPath); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	cfg := mgr.GetConfig()
	if cfg.MiproV2.NumCandidates != 10 {
		t.Errorf("expected NumCandidates=10, got %d", cfg.MiproV2.NumCandidates)
	}
	if cfg.MiproV2.NumTrials != 20 {
		t.Errorf("expected NumTrials=20, got %d", cfg.MiproV2.NumTrials)
	}
	if cfg.Evaluation.Threshold != 0.8 {
		t.Errorf("expected Threshold=0.8, got %f", cfg.Evaluation.Threshold)
	}
	if cfg.API.TimeoutSeconds != 60 {
		t.Errorf("expected TimeoutSeconds=60, got %d", cfg.API.TimeoutSeconds)
	}
	if cfg.Dataset.BasePath != "custom/path" {
		t.Errorf("expected BasePath=custom/path, got %s", cfg.Dataset.BasePath)
	}
	if cfg.Gateway.FallbackThreshold != 0.5 {
		t.Errorf("expected FallbackThreshold=0.5, got %f", cfg.Gateway.FallbackThreshold)
	}

	// Check provider config
	openai, ok := cfg.ModelProviders["openai"]
	if !ok {
		t.Fatal("expected openai provider")
	}
	if openai.APIKey != "test-key" {
		t.Errorf("expected APIKey=test-key, got %s", openai.APIKey)
	}
	if len(openai.Models) != 2 {
		t.Errorf("expected 2 models, got %d", len(openai.Models))
	}
}

func TestLoadJSON(t *testing.T) {
	jsonContent := `{
		"model_providers": {
			"anthropic": {
				"api_key": "anthropic-key",
				"models": ["claude-3-opus"]
			}
		},
		"mipro_v2": {
			"num_candidates": 8,
			"num_trials": 15,
			"max_bootstrapped_demos": 4,
			"max_labeled_demos": 2,
			"minibatch_size": 8,
			"minibatch_full_eval_steps": 4,
			"num_instruction_candidates": 4,
			"evaluation_strategy": "Similarity"
		},
		"evaluation": {
			"default_strategy": "Similarity",
			"threshold": 0.75,
			"max_retries": 4
		},
		"api": {
			"timeout_seconds": 45,
			"max_retries": 4
		},
		"dataset": {
			"base_path": "json/path",
			"max_records": 750
		},
		"analytics": {
			"driver": "duckdb",
			"path": "analytics.duckdb"
		}
	}`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	mgr := NewSimpleConfigManager()
	if err := mgr.Load(configPath); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	cfg := mgr.GetConfig()
	if cfg.MiproV2.NumCandidates != 8 {
		t.Errorf("expected NumCandidates=8, got %d", cfg.MiproV2.NumCandidates)
	}
	if cfg.Dataset.BasePath != "json/path" {
		t.Errorf("expected BasePath=json/path, got %s", cfg.Dataset.BasePath)
	}

	anthropic, ok := cfg.ModelProviders["anthropic"]
	if !ok {
		t.Fatal("expected anthropic provider")
	}
	if anthropic.APIKey != "anthropic-key" {
		t.Errorf("expected APIKey=anthropic-key, got %s", anthropic.APIKey)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	mgr := NewSimpleConfigManager()
	err := mgr.Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadInvalidConfig(t *testing.T) {
	invalidContent := `
mipro_v2:
  num_candidates: 0
evaluation:
  threshold: 0.7
api:
  timeout_seconds: 30
dataset:
  base_path: "data"
  max_records: 100
analytics:
  driver: "duckdb"
  path: "data/analytics.duckdb"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	mgr := NewSimpleConfigManager()
	err := mgr.Load(configPath)
	if err == nil {
		t.Error("expected validation error")
	}
}

func TestSave(t *testing.T) {
	mgr := NewSimpleConfigManager()
	cfg := mgr.GetConfig()
	cfg.MiproV2.NumCandidates = 15
	cfg.ModelProviders["openai"] = ProviderConfig{
		APIKey: "saved-key",
		Models: []string{"gpt-4"},
	}

	tmpDir := t.TempDir()
	savePath := filepath.Join(tmpDir, "saved-config.yaml")

	if err := mgr.Save(savePath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Load the saved config to verify
	mgr2 := NewSimpleConfigManager()
	if err := mgr2.Load(savePath); err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	cfg2 := mgr2.GetConfig()
	if cfg2.MiproV2.NumCandidates != 15 {
		t.Errorf("expected NumCandidates=15, got %d", cfg2.MiproV2.NumCandidates)
	}
	openai, ok := cfg2.ModelProviders["openai"]
	if !ok {
		t.Fatal("expected openai provider in saved config")
	}
	if openai.APIKey != "saved-key" {
		t.Errorf("expected APIKey=saved-key, got %s", openai.APIKey)
	}
}

func TestUpdateConfig(t *testing.T) {
	mgr := NewSimpleConfigManager()

	updates := map[string]interface{}{
		"api": map[string]interface{}{
			"timeout_seconds": 90,
			"max_retries":     10,
		},
	}

	if err := mgr.UpdateConfig(updates); err != nil {
		t.Fatalf("failed to update config: %v", err)
	}

	cfg := mgr.GetConfig()
	if cfg.API.TimeoutSeconds != 90 {
		t.Errorf("expected TimeoutSeconds=90, got %d", cfg.API.TimeoutSeconds)
	}
	if cfg.API.MaxRetries != 10 {
		t.Errorf("expected MaxRetries=10, got %d", cfg.API.MaxRetries)
	}
}

func TestUpdateConfigValidationFails(t *testing.T) {
	mgr := NewSimpleConfigManager()

	updates := map[string]interface{}{
		"api": map[string]interface{}{
			"timeout_seconds": 0, // Invalid
		},
	}

	err := mgr.UpdateConfig(updates)
	if err == nil {
		t.Error("expected validation error for invalid update")
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	yamlContent := `
model_providers:
  openai:
    api_key: ""
    models:
      - gpt-4
  anthropic:
    api_key: "config-key"
    models:
      - claude-3
mipro_v2:
  num_candidates: 5
  num_trials: 10
  max_bootstrapped_demos: 3
  max_labeled_demos: 2
  minibatch_size: 5
  minibatch_full_eval_steps: 3
  num_instruction_candidates: 3
evaluation:
  threshold: 0.7
api:
  timeout_seconds: 30
dataset:
  base_path: "data"
  max_records: 100
analytics:
  driver: "duckdb"
  path: "data/analytics.duckdb"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Set environment variables
	os.Setenv("OPENAI_API_KEY", "env-openai-key")
	os.Setenv("ROUTE_SWITCH_ANTHROPIC_API_KEY", "env-anthropic-key")
	os.Setenv("ROUTE_SWITCH_ANALYTICS_PATH", "custom/analytics.duckdb")
	os.Setenv("ROUTE_SWITCH_DATASET_PATH", "custom/dataset")
	defer func() {
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("ROUTE_SWITCH_ANTHROPIC_API_KEY")
		os.Unsetenv("ROUTE_SWITCH_ANALYTICS_PATH")
		os.Unsetenv("ROUTE_SWITCH_DATASET_PATH")
	}()

	mgr := NewSimpleConfigManager()
	if err := mgr.Load(configPath); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	cfg := mgr.GetConfig()

	// OpenAI should be overridden by OPENAI_API_KEY (fallback)
	openai, ok := cfg.ModelProviders["openai"]
	if !ok {
		t.Fatal("expected openai provider")
	}
	if openai.APIKey != "env-openai-key" {
		t.Errorf("expected OpenAI APIKey=env-openai-key, got %s", openai.APIKey)
	}

	// Anthropic should be overridden by ROUTE_SWITCH_ANTHROPIC_API_KEY (explicit)
	anthropic, ok := cfg.ModelProviders["anthropic"]
	if !ok {
		t.Fatal("expected anthropic provider")
	}
	if anthropic.APIKey != "env-anthropic-key" {
		t.Errorf("expected Anthropic APIKey=env-anthropic-key, got %s", anthropic.APIKey)
	}

	// Analytics path should be overridden
	if cfg.Analytics.Path != "custom/analytics.duckdb" {
		t.Errorf("expected Analytics.Path=custom/analytics.duckdb, got %s", cfg.Analytics.Path)
	}

	// Dataset path should be overridden
	if cfg.Dataset.BasePath != "custom/dataset" {
		t.Errorf("expected Dataset.BasePath=custom/dataset, got %s", cfg.Dataset.BasePath)
	}
}

func TestEnvironmentOverridesPriority(t *testing.T) {
	// Test that ROUTE_SWITCH_<PROVIDER>_API_KEY takes priority over standard env vars
	yamlContent := `
model_providers:
  openai:
    api_key: ""
    models:
      - gpt-4
mipro_v2:
  num_candidates: 5
  num_trials: 10
  max_bootstrapped_demos: 3
  max_labeled_demos: 2
  minibatch_size: 5
  minibatch_full_eval_steps: 3
  num_instruction_candidates: 3
evaluation:
  threshold: 0.7
api:
  timeout_seconds: 30
dataset:
  base_path: "data"
  max_records: 100
analytics:
  driver: "duckdb"
  path: "data/analytics.duckdb"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Set both env vars - ROUTE_SWITCH_ should take priority
	os.Setenv("OPENAI_API_KEY", "standard-key")
	os.Setenv("ROUTE_SWITCH_OPENAI_API_KEY", "route-switch-key")
	defer func() {
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("ROUTE_SWITCH_OPENAI_API_KEY")
	}()

	mgr := NewSimpleConfigManager()
	if err := mgr.Load(configPath); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	cfg := mgr.GetConfig()
	openai, ok := cfg.ModelProviders["openai"]
	if !ok {
		t.Fatal("expected openai provider")
	}
	if openai.APIKey != "route-switch-key" {
		t.Errorf("expected APIKey=route-switch-key, got %s", openai.APIKey)
	}
}

func TestGatewayConfig(t *testing.T) {
	yamlContent := `
gateway:
  addr: ":8080"
  strategy: "weighted_round_robin"
  fallback_threshold: 0.7
  combinations:
    - id: "combo-1"
      name: "Primary GPT-4"
      prompt: "You are a helpful assistant"
      model: "gpt-4"
      provider: "openai"
      is_primary: true
      weight: 80
      enabled: true
      fallbacks:
        - "combo-2"
    - id: "combo-2"
      name: "Fallback Claude"
      prompt: "You are a helpful assistant"
      model: "claude-3-opus"
      provider: "anthropic"
      weight: 20
      enabled: true
  optimization:
    enabled: true
    interval_seconds: 3600
    target_rps: 100
mipro_v2:
  num_candidates: 5
  num_trials: 10
  max_bootstrapped_demos: 3
  max_labeled_demos: 2
  minibatch_size: 5
  minibatch_full_eval_steps: 3
  num_instruction_candidates: 3
evaluation:
  threshold: 0.7
api:
  timeout_seconds: 30
dataset:
  base_path: "data"
  max_records: 100
analytics:
  driver: "duckdb"
  path: "data/analytics.duckdb"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gateway.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	mgr := NewSimpleConfigManager()
	if err := mgr.Load(configPath); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	cfg := mgr.GetConfig()
	if cfg.Gateway.Addr != ":8080" {
		t.Errorf("expected Addr=:8080, got %s", cfg.Gateway.Addr)
	}
	if cfg.Gateway.Strategy != "weighted_round_robin" {
		t.Errorf("expected Strategy=weighted_round_robin, got %s", cfg.Gateway.Strategy)
	}
	if cfg.Gateway.FallbackThreshold != 0.7 {
		t.Errorf("expected FallbackThreshold=0.7, got %f", cfg.Gateway.FallbackThreshold)
	}
	if len(cfg.Gateway.Combinations) != 2 {
		t.Fatalf("expected 2 combinations, got %d", len(cfg.Gateway.Combinations))
	}

	combo1 := cfg.Gateway.Combinations[0]
	if combo1.ID != "combo-1" {
		t.Errorf("expected ID=combo-1, got %s", combo1.ID)
	}
	if !combo1.IsPrimary {
		t.Error("expected combo-1 to be primary")
	}
	if combo1.Weight != 80 {
		t.Errorf("expected Weight=80, got %d", combo1.Weight)
	}
	if len(combo1.Fallbacks) != 1 || combo1.Fallbacks[0] != "combo-2" {
		t.Errorf("expected Fallbacks=[combo-2], got %v", combo1.Fallbacks)
	}

	if !cfg.Gateway.Optimization.Enabled {
		t.Error("expected optimization to be enabled")
	}
	if cfg.Gateway.Optimization.Interval != 3600 {
		t.Errorf("expected Interval=3600, got %d", cfg.Gateway.Optimization.Interval)
	}
}
