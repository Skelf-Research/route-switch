# CLI Usage

The `route-switch` binary exposes all management operations: prompt optimization, multi-model search, and running the OpenAI-compatible gateway.

## Global Flags
- `-c, --config string` – path to the YAML/JSON config file.
- `-p, --prompt string` – prompt template or default text (required for CLI modes).
- `-m, --model string` – base model name (required for CLI modes).
- `-r, --provider string` – provider alias. Defaults to `gollm`, which fans out across every entry in `model_providers`. Pass a specific provider name (e.g., `openai`) to limit routing to that backend, or use `mock` for local testing.
- `--template-id string` – use a previously registered template; fills in the prompt, default model, and default provider unless explicitly overridden.
- `-a, --addr string` – gateway address when `--gateway` is set (default `:8080`).
- `-g, --gateway` – run the HTTP gateway instead of one-off optimization.
- `-o, --optimize-prompt` – optimize the prompt for the specified model.
- `-f, --find-best-model` – search across all provider models.

## Prompt Optimization
```bash
./route-switch \
  --config config.yaml \
  --prompt "Explain {topic} to a 10 year old" \
  --model gpt-4 \
  --provider gollm \
  --optimize-prompt
```
The CLI prints progress (bootstrapping, instruction search, Bayesian trials) and outputs the optimized template, chosen model, and estimated cost. All invocations are logged to the prompt’s dataset store automatically.

## Model Search
```bash
./route-switch \
  --config config.yaml \
  --prompt "Summarize the latest AI trends" \
  --model gpt-4 \
  --provider gollm \
  --find-best-model
```
The optimizer iterates over every provider/model listed in the configuration and reports the best-performing prompt/model pair.

## Gateway Mode
Run the OpenAI-compatible proxy:
```bash
./route-switch \
  --config config.yaml \
  --provider gollm \
  --gateway \
  --prompt "Customer support helper" \
  --model gpt-4 \
  --addr :8080
```
- `POST /v1/chat/completions` – accepts OpenAI-chat payloads. Route-Switch injects the optimized template, selects a model via the load balancer, and streams the result.
- `GET /health` – lightweight readiness probe.
- `GET /status` – JSON summary of every prompt/model combination plus live performance metrics.
- `GET /v1/prompts/{id}/stats` – aggregated success rate/latency/cost for an individual template ID (or combination ID).
- `GET /v1/system/analytics` – global request and cost totals across all prompts.

Every gateway request is captured in the per-prompt SQLite dataset so background optimization has up-to-date examples.

## Template Registration
Persist prompt templates (including variable names and defaults) with the new subcommand:

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

If `--prompt-file` is omitted, use `--prompt-text` instead. The manifest is written under `dataset.base_path/<template-id>/manifest.yaml`.

## Package Export
Bundle a prompt template, dataset snapshot, logs, and analytics excerpt into a git-initialized folder:

```bash
./route-switch package export \
  --template-id onboarding \
  --output-dir packages \
  --include-analytics \
  --logs-limit 200 \
  --config config.yaml
```

The package is ready to copy to other servers and includes `manifest.yaml`, `package.yaml`, `dataset/<template>.db`, `logs/recent.jsonl`, and (optionally) `analytics/<file>.duckdb`. Because the folder is already a git repository, every edit you make is tracked automatically.

## Package Import
Restore a package on a new server:

```bash
./route-switch package import \
  --path packages/support-flow-20241208-104500 \
  --config config.yaml \
  --restore-analytics \
  --overwrite
```

Flags:
- `--path` (required): folder containing `package.yaml`.
- `--restore-analytics`: copy the packaged DuckDB snapshot to `analytics.path` (default true).
- `--overwrite`: replace any existing manifest/dataset/log files.

After import, the manifest and dataset live under `dataset.base_path/<template-id>/`, so the template is immediately available for gateway modes or future exports.

## Environment Variables
Common provider keys can be read from the environment when omitted from config:
- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`
- `GOOGLE_API_KEY`

## Troubleshooting
- **Provider errors** – ensure the provider block exists in `model_providers` and contains the right API key/base URL.
- **Gateway exits immediately** – check that a prompt/model is supplied or prompt combinations are defined under `gateway.combinations`.
- **No traffic recorded** – confirm `dataset.base_path` is writable; Route-Switch creates a SQLite DB per prompt ID.
