# CLI Reference

The `route-switch` binary provides all management operations: optimization, model search, gateway, templates, and packages.

## Global Flags

| Flag | Short | Type | Description |
|------|-------|------|-------------|
| `--config` | `-c` | string | Path to YAML/JSON config file |
| `--prompt` | `-p` | string | Prompt template or default text |
| `--model` | `-m` | string | Base model name |
| `--provider` | `-r` | string | Provider alias (default: `gollm`) |
| `--template-id` | | string | Use registered template |
| `--addr` | `-a` | string | Gateway address (default: `:8080`) |
| `--gateway` | `-g` | bool | Run HTTP gateway mode |
| `--optimize-prompt` | `-o` | bool | Optimize the prompt |
| `--find-best-model` | `-f` | bool | Search across all models |
| `--evaluation-strategy` | | string | Override evaluation strategy |

## Commands

### Prompt Optimization

Optimize a prompt for a specific model:

```bash
./route-switch \
  --config config.yaml \
  --prompt "Explain {topic} to a 10 year old" \
  --model gpt-4 \
  --provider gollm \
  --optimize-prompt
```

**Output:**

- Bootstrapping progress
- Instruction search status
- Bayesian trial results
- Optimized prompt
- Improvement score
- Cost estimate

### Model Search

Find the best model for a prompt:

```bash
./route-switch \
  --config config.yaml \
  --prompt "Summarize the latest AI trends" \
  --model gpt-4 \
  --provider gollm \
  --find-best-model
```

**Output:**

- Per-model evaluation results
- Best prompt/model pair
- Performance comparison

### Gateway Mode

Run the OpenAI-compatible gateway:

```bash
./route-switch \
  --config config.yaml \
  --gateway \
  --provider gollm \
  --prompt "Customer support helper" \
  --model gpt-4 \
  --addr :8080
```

**Endpoints exposed:**

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/chat/completions` | POST | Chat inference |
| `/health` | GET | Readiness probe |
| `/status` | GET | Combinations status |
| `/v1/prompts/{id}/stats` | GET | Per-prompt stats |
| `/v1/system/analytics` | GET | Global analytics |

---

## template Commands

### template register

Register a new prompt template:

```bash
./route-switch template register \
  --template-id onboarding \
  --name "Customer Onboarding" \
  --prompt-file prompts/onboarding.txt \
  --variables customer_name,plan,tone \
  --default-model gpt-4 \
  --default-provider gollm \
  --config config.yaml
```

| Flag | Type | Description |
|------|------|-------------|
| `--template-id` | string | Unique identifier (required) |
| `--name` | string | Display name (required) |
| `--prompt-file` | string | Path to prompt text file |
| `--prompt-text` | string | Inline prompt text |
| `--variables` | string | Comma-separated variable names |
| `--default-model` | string | Default model |
| `--default-provider` | string | Default provider |

### template list

List all registered templates:

```bash
./route-switch template list --config config.yaml
```

**Output:**

```
ID              Name
onboarding      Customer Onboarding
support-flow    Support Flow
```

---

## package Commands

### package export

Export a template as a portable package:

```bash
./route-switch package export \
  --template-id onboarding \
  --output-dir packages \
  --include-analytics \
  --logs-limit 200 \
  --config config.yaml
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--template-id` | string | | Template to export (required) |
| `--output-dir` | string | | Destination directory |
| `--include-analytics` | bool | false | Include DuckDB snapshot |
| `--logs-limit` | int | 100 | Number of recent logs |

**Output structure:**

```
packages/onboarding-20241208-104500/
├── manifest.yaml
├── package.yaml
├── dataset/onboarding.db
├── logs/recent.jsonl
└── analytics/metrics.duckdb
```

### package import

Import a package:

```bash
./route-switch package import \
  --path packages/support-flow-20241208-104500 \
  --config config.yaml \
  --restore-analytics \
  --overwrite
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--path` | string | | Package directory (required) |
| `--restore-analytics` | bool | true | Restore DuckDB snapshot |
| `--overwrite` | bool | false | Replace existing files |

---

## gateway Commands

### gateway combinations

View configured prompt/model combinations:

```bash
./route-switch gateway combinations --config config.yaml
```

**Output:**

```
ID          Model       Provider    Weight    Primary
default     gpt-4       openai      10        true
fallback    gpt-3.5     openai      5         false
```

---

## Using Templates

Reference a registered template with `--template-id`:

```bash
./route-switch --config config.yaml \
  --template-id support-flow \
  --optimize-prompt
```

This loads:

- Prompt text from manifest
- Default model
- Default provider

Override any setting with explicit flags.

---

## Provider Selection

| Value | Behavior |
|-------|----------|
| `gollm` (default) | Fan out across all `model_providers` |
| `<provider>` | Restrict to specific provider (e.g., `openai`) |
| `mock` | Local testing without API calls |

---

## Environment Variables

Provider keys are loaded after the YAML file. `ROUTE_SWITCH_<PROVIDER>_API_KEY` takes precedence over the standard form, which in turn overrides values in `model_providers.<provider>.api_key`.

| Variable | Description |
|----------|-------------|
| `ROUTE_SWITCH_<PROVIDER>_API_KEY` | Provider-scoped override (e.g. `ROUTE_SWITCH_OPENAI_API_KEY`) |
| `OPENAI_API_KEY` | OpenAI key fallback |
| `ANTHROPIC_API_KEY` | Anthropic key fallback |
| `GOOGLE_API_KEY` / `GEMINI_API_KEY` | Google / Gemini key fallback |
| `COHERE_API_KEY` | Cohere key fallback |
| `MISTRAL_API_KEY` | Mistral key fallback |
| `GROQ_API_KEY` | Groq key fallback |
| `ROUTE_SWITCH_ANALYTICS_PATH` | Override DuckDB analytics file path |
| `ROUTE_SWITCH_DATASET_PATH` | Override dataset base path |

---

## Troubleshooting

### Provider errors

Ensure the provider block exists in `model_providers` with valid API key and base URL.

### Gateway exits immediately

Check that:

- A prompt/model is supplied via flags
- Or combinations are defined under `gateway.combinations`

### No traffic recorded

Verify `dataset.base_path` is writable. Route-Switch creates a SQLite DB per prompt ID.

### Template not found

Ensure the template was registered and the manifest exists at:

```
data/prompts/<template-id>/manifest.yaml
```
