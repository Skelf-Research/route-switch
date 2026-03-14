# Route-Switch

Route-Switch is a Go-powered prompt function manager that combines MIPROv2-style optimization, gollm-based multi-provider routing, and production-aware analytics. Register a prompt template once, capture real invocations in lightweight per-prompt datasets, and let the router continuously optimize and load-balance traffic across every model you trust.

## Capabilities
- **Prompt registration & templates** – define prompts with variables plus a baseline model and automatically capture every invocation.
- **Faithful MIPROv2 optimization** – reuse production traces as the calibration set when adapting a prompt to additional models or providers.
- **Gateway & inference endpoint** – expose a single OpenAI-compatible endpoint that handles prompt rendering, model selection, and failover.
- **Analytics & packaging** – store request/response metadata in DuckDB (default) and export portable prompt packages with manifests, logs, and datasets for other environments.
- **Extensible providers** – connect OpenAI, Anthropic, Google, Ollama, Cohere, Mistral, Hugging Face, or a mock provider through the gollm abstraction.

## Quick Start
1. **Build the CLI**
   ```bash
   go build -o route-switch
   ```
2. **Create a configuration file** (`config.yaml`):
   ```yaml
   model_providers:
     openai:
       api_key: "your-openai-api-key"
       base_url: "https://api.openai.com/v1"
       models: ["gpt-4o", "gpt-4", "gpt-3.5-turbo"]
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
     strategy: "round_robin"
     optimization:
       enabled: true
       interval_seconds: 3600
   ```
3. **(Optional) Register a prompt template**
   ```bash
   ./route-switch template register \
     --template-id support-flow \
     --name "Support Onboarding Flow" \
     --prompt-file prompts/support.txt \
     --variables customer_name,ticket_id,issue \
     --config config.yaml
   ```
   Reuse `--template-id support-flow` later with `route-switch --template-id support-flow ...` to pull in the stored prompt/model/provider defaults.

4. **Optimize or evaluate via CLI**
   ```bash
   # Optimize the prompt for a specific model
   ./route-switch --config config.yaml --prompt "Write a poem about programming" --model "gpt-4" --provider gollm --optimize-prompt

   # Search for a cheaper/better model
   ./route-switch --config config.yaml --prompt "Summarize the latest AI news" --model "gpt-4" --provider gollm --find-best-model
   ```
5. **Start the gateway** (exposes `/v1/chat/completions`, `/health`, `/status`, and analytics APIs)
   ```bash
   ./route-switch --config config.yaml --gateway --provider gollm --prompt "Default onboarding prompt" --model gpt-4 --addr :8080
   ```
Once running, send OpenAI-style chat requests to `POST /v1/chat/completions` and Route-Switch will inject the optimized template, call the selected model, and log the interaction for future optimization runs. Operational telemetry is exposed via:
- `GET /status` – snapshot of every prompt/model combination plus live performance metrics.
- `GET /v1/prompts/{id}/stats` – aggregated success rate, latency, and cost for a specific template or combination ID.
- `GET /v1/system/analytics` – global request totals backed by the DuckDB analytics store.

Route-Switch talks to upstream providers exclusively through the [gollm](https://github.com/teilomillet/gollm) adapter. Declare each upstream provider under `model_providers`, then use `--provider gollm` (default) to fan out across all configured backends, or pass a specific provider name (e.g., `--provider openai`) to restrict routing to that subset.

## Prompt Dataset Storage
Each registered prompt gets its own SQLite database under the configured `dataset.base_path`. Every invocation records:
- rendered prompt input/output
- variable bindings supplied at invocation time
- provider/model metadata
- success flags, cost estimates, and timestamps

Records are automatically trimmed to `dataset.max_records` to keep storage bounded. These datasets double as the calibration/evaluation set for MIPROv2 when onboarding new models.

The analytics subsystem mirrors each invocation into DuckDB (`analytics.path`) and powers the status/statistics endpoints, packaging metadata, and future dashboards.

### Supplying Variables
When calling the gateway, include a `variables` object alongside the OpenAI-style payload:

```json
{
  "model": "gpt-4",
  "messages": [{"role": "user", "content": "Please help."}],
  "variables": {
    "customer_name": "Jordan",
    "issue": "account locked"
  }
}
```

Route-Switch replaces `{customer_name}` and `{issue}` placeholders inside the optimized prompt before invoking the provider and stores those bindings with the dataset record.

## Registering Prompt Templates
Use the CLI to persist prompt templates (with variables and default routing hints) alongside their dataset and manifest:

```bash
./route-switch template register \
  --template-id support-flow \
  --name "Support Onboarding Flow" \
  --prompt-file prompts/support.txt \
  --variables customer_name,ticket_id,issue \
  --default-model gpt-4 \
  --default-provider gollm \
  --config config.yaml
```

This command writes `data/prompts/support-flow/manifest.yaml`, ensuring the template metadata travels with the dataset snapshot and portable package artifacts.

## Portable Prompt Packages
Share an optimized template, dataset, analytics snapshot, and recent logs with a single command:
```bash
./route-switch package export \
  --template-id support-flow \
  --output-dir packages \
  --include-analytics \
  --config config.yaml
```
The exporter creates `packages/support-flow-<timestamp>/` containing:
- `manifest.yaml` – the registered template (prompt text, variables, default routing).
- `dataset/<template>.db` – the per-prompt SQLite dataset.
- `logs/recent.jsonl` – the latest N invocations for quick inspection.
- `analytics/<file>.duckdb` – optional DuckDB snapshot for dashboards and load-balancing heuristics.
- `package.yaml` – a manifest describing the bundle and whether git history was initialized.

Every package is `git init`-ed automatically so changes remain tracked as you inspect, edit, or commit inside the exported directory. Copy the folder to a new server, configure credentials, and the gateway can hydrate itself from the manifest.

### Importing Packages
Hydrate a package onto another machine (manifest, dataset, logs, and optional analytics snapshot):
```bash
./route-switch package import \
  --path packages/support-flow-20241208-104500 \
  --config config.yaml \
  --restore-analytics \
  --overwrite
```
The importer copies `manifest.yaml` and `support-flow.db` into your configured `dataset.base_path`, restores `logs/recent.jsonl` under the template directory, and replaces the DuckDB file at `analytics.path` when `--restore-analytics` is provided. Use `--overwrite` when you intentionally want to replace an existing manifest/dataset/analytics file.

## Documentation
- [Architecture](docs/architecture.md)
- [Configuration](docs/configuration.md)
- [CLI Usage](docs/cli.md)
- [API Reference](docs/api-reference.md)
- [Evaluation & Optimization](docs/evaluation.md)

Run `env GOCACHE=$(pwd)/.cache GOMODCACHE=$(pwd)/.modcache go test ./...` to execute the unit test suite. Use the same cache variables when building inside constrained environments.
