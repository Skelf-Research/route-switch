# Configuration

Route-Switch reads YAML or JSON configuration files. All runtime services (CLI, gateway, optimizer) use the same schema, so you can keep provider credentials, dataset paths, and gateway rules in one place.

## Example
```yaml
model_providers:
  openai:
    api_key: "your-key"
    base_url: "https://api.openai.com/v1"
    models: ["gpt-4o", "gpt-4", "gpt-3.5-turbo"]
    options:
      temperature: 0.2
    rate_limit: 1200
  mock:
    api_key: mock
mipro_v2:
  num_candidates: 5
  max_bootstrapped_demos: 3
  max_labeled_demos: 2
  num_trials: 20
  minibatch_size: 5
  minibatch_full_eval_steps: 3
  num_instruction_candidates: 4
  evaluation_strategy: "Similarity"
evaluation:
  default_strategy: "Similarity"
  threshold: 0.7
  max_retries: 3
api:
  timeout_seconds: 30
  max_retries: 3
dataset:
  base_path: "data/prompts"
  max_records: 1000
gateway:
  addr: ":8080"
  strategy: "performance_based"
  optimization:
    enabled: true
    interval_seconds: 1800
  combinations:
    - id: "default"
      name: "default"
      prompt: "Default onboarding prompt: {topic}"
      model: "gpt-4"
      provider: "openai"
      is_primary: true
      weight: 10
      metadata:
        optimized: false
analytics:
  driver: "duckdb"
  path: "data/analytics/metrics.duckdb"
```

## Sections

### `model_providers`
Define as many providers as needed. Each entry maps a provider alias to credentials and optional defaults. Route-Switch uses the gollm adapter to reach upstream vendors, so these blocks are forwarded directly to gollm. Run-time provider selection works as follows:
- `--provider gollm` (default) loads every entry listed in `model_providers` and allows the router to choose across them.
- `--provider <name>` (e.g., `openai`, `anthropic`) restricts gollm to that specific entry from the configuration.
- `--provider mock` keeps everything local for testing.
- `api_key`, `base_url`, `models`, `rate_limit` behave as expected.
- `options` can contain provider-specific overrides (temperature, stop sequences, etc.).

### `mipro_v2`
Controls the optimization loop. Increase `num_trials` for more thorough Bayesian searches or adjust `num_instruction_candidates` to generate additional instruction proposals.

### `evaluation`
Sets the default evaluation strategy (`Similarity`, `Keyword`, or `ExactMatch`) used by both CLI runs and the gateway unless overridden. `threshold` determines success/failure when computing improvement scores.

### `api`
Tunables for provider HTTP calls: request timeout and retry count.

### `dataset`
- `base_path`: directory where per-prompt SQLite databases are stored.
- `max_records`: rolling cap per prompt; older records are pruned automatically.

### `gateway`
- `addr`: listen address for the OpenAI-compatible proxy.
- `strategy`: `round_robin`, `weighted_round_robin`, or `performance_based`.
- `combinations`: optional static prompt/model combos; if omitted, CLI flags or prompt registration APIs can populate them at runtime.
- `optimization`: background optimizer toggle plus interval (seconds).
- `fallback_threshold`: optional success-rate floor (0-1). When a combination dips below it, the registry automatically reduces its weight and promotes configured fallbacks.

### `analytics`
Configures the analytics sink (DuckDB by default). Specify the on-disk path for the database. Additional drivers can be added by implementing the analytics interface.

## Loading
```go
cfgManager := config.NewSimpleConfigManager()
if err := cfgManager.Load("config.yaml"); err != nil {
    log.Fatalf("load config: %v", err)
}
config := cfgManager.GetConfig()
```
From the CLI you can pass `--config` to point at any YAML/JSON file.

## Validation
The configuration manager enforces:
- counts (`num_candidates`, `num_trials`, `max_records`) > 0
- evaluation thresholds between 0 and 1
- API timeout > 0
- dataset base path is non-empty
Additional semantic validation (e.g., provider/model presence) occurs when the gateway/optimizer starts.
