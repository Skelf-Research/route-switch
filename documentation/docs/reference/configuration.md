# Configuration Reference

Route-Switch uses YAML or JSON configuration files. All services (CLI, gateway, optimizer) share the same schema.

## Complete Example

```yaml
model_providers:
  openai:
    api_key: "your-key"
    base_url: "https://api.openai.com/v1"
    models: ["gpt-4o", "gpt-4", "gpt-3.5-turbo"]
    options:
      temperature: 0.2
    rate_limit: 1200
  anthropic:
    api_key: "your-anthropic-key"
    models: ["claude-3-opus", "claude-3-sonnet"]
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
  fallback_threshold: 0.8
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

### model_providers

Define provider credentials and settings.

```yaml
model_providers:
  openai:
    api_key: "your-key"
    base_url: "https://api.openai.com/v1"
    models: ["gpt-4o", "gpt-4", "gpt-3.5-turbo"]
    options:
      temperature: 0.2
    rate_limit: 1200
```

| Field | Type | Description |
|-------|------|-------------|
| `api_key` | string | Provider API key |
| `base_url` | string | API base URL |
| `models` | string[] | Available models |
| `options` | object | Provider-specific options (temperature, etc.) |
| `rate_limit` | int | Requests per minute limit |

**Provider Selection:**

- `--provider gollm` (default) - Uses all configured providers
- `--provider <name>` - Restricts to specific provider
- `--provider mock` - Local testing

### mipro_v2

Controls the MIPROv2 optimization loop.

```yaml
mipro_v2:
  num_candidates: 5
  max_bootstrapped_demos: 3
  max_labeled_demos: 2
  num_trials: 20
  minibatch_size: 5
  minibatch_full_eval_steps: 3
  num_instruction_candidates: 4
  evaluation_strategy: "Similarity"
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `num_candidates` | int | 5 | Final prompt candidates to evaluate |
| `max_bootstrapped_demos` | int | 3 | Max examples for bootstrapping |
| `max_labeled_demos` | int | 2 | Max labeled training examples |
| `num_trials` | int | 20 | Bayesian optimization trials |
| `minibatch_size` | int | 5 | Evaluation minibatch size |
| `minibatch_full_eval_steps` | int | 3 | Full evaluation frequency |
| `num_instruction_candidates` | int | 4 | Instruction proposals to generate |
| `evaluation_strategy` | string | "Similarity" | Evaluation strategy |

### evaluation

Default evaluation settings.

```yaml
evaluation:
  default_strategy: "Similarity"
  threshold: 0.7
  max_retries: 3
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `default_strategy` | string | "Similarity" | `Similarity`, `Keyword`, or `ExactMatch` |
| `threshold` | float | 0.7 | Success threshold (0-1) |
| `max_retries` | int | 3 | Retry count for failed evaluations |

### api

HTTP client settings for provider calls.

```yaml
api:
  timeout_seconds: 30
  max_retries: 3
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `timeout_seconds` | int | 30 | Request timeout |
| `max_retries` | int | 3 | Retry count |

### dataset

Per-prompt dataset storage settings.

```yaml
dataset:
  base_path: "data/prompts"
  max_records: 1000
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `base_path` | string | "data/prompts" | Storage directory |
| `max_records` | int | 1000 | Max records per prompt |

### gateway

Gateway server configuration.

```yaml
gateway:
  addr: ":8080"
  strategy: "performance_based"
  fallback_threshold: 0.8
  optimization:
    enabled: true
    interval_seconds: 1800
  combinations:
    - id: "default"
      name: "default"
      prompt: "Default prompt: {topic}"
      model: "gpt-4"
      provider: "openai"
      is_primary: true
      weight: 10
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `addr` | string | ":8080" | Listen address |
| `strategy` | string | "round_robin" | Load balancing strategy |
| `fallback_threshold` | float | - | Success rate floor (0-1) |
| `optimization.enabled` | bool | false | Enable background optimization |
| `optimization.interval_seconds` | int | 3600 | Optimization interval |
| `combinations` | array | - | Static prompt/model combinations |

**Strategies (defined in `internal/gateway/load_balancer.go`):**

- `round_robin` - Even rotation across enabled combinations
- `random` - Uniform random selection
- `weighted_round_robin` - Weighted random by `weight`
- `performance_based` - Route by recent success rate and latency
- `least_connections` - Route to the combination with the fewest in-flight requests

**Combination Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier |
| `name` | string | Display name |
| `prompt` | string | Prompt template |
| `model` | string | Model name |
| `provider` | string | Provider name |
| `is_primary` | bool | Primary combination flag |
| `weight` | int | Load balancing weight |
| `metadata` | object | Additional metadata |

### analytics

Analytics storage configuration.

```yaml
analytics:
  driver: "duckdb"
  path: "data/analytics/metrics.duckdb"
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `driver` | string | "duckdb" | Storage driver |
| `path` | string | - | Database file path |

## Loading Configuration

### CLI

```bash
./route-switch --config config.yaml
```

### Go

```go
cfgManager := config.NewSimpleConfigManager()
if err := cfgManager.Load("config.yaml"); err != nil {
    log.Fatalf("load config: %v", err)
}
config := cfgManager.GetConfig()
```

## Validation

The configuration manager validates:

- Counts (`num_candidates`, `num_trials`, `max_records`) > 0
- Thresholds between 0 and 1
- API timeout > 0
- Dataset base path is non-empty

Additional validation occurs at service startup.

## Environment Variables

Environment overrides are applied in `internal/config/manager.go` after the file is parsed. Provider keys are checked in this priority order: `ROUTE_SWITCH_<PROVIDER>_API_KEY` → standard form → value in `model_providers.<provider>.api_key`.

| Variable | Description |
|----------|-------------|
| `ROUTE_SWITCH_<PROVIDER>_API_KEY` | Provider-scoped override |
| `OPENAI_API_KEY` | OpenAI key fallback |
| `ANTHROPIC_API_KEY` | Anthropic key fallback |
| `GOOGLE_API_KEY` / `GEMINI_API_KEY` | Google / Gemini key fallback |
| `COHERE_API_KEY` | Cohere key fallback |
| `MISTRAL_API_KEY` | Mistral key fallback |
| `GROQ_API_KEY` | Groq key fallback |
| `ROUTE_SWITCH_ANALYTICS_PATH` | Override DuckDB analytics file path |
| `ROUTE_SWITCH_DATASET_PATH` | Override dataset base path |

This is intentional: Route-Switch never reads secrets from a keyring or encrypted file. The expectation is env-only secret injection in deployment.
