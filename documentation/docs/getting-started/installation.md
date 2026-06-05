# Installation

This guide covers how to install and set up Route-Switch on your system.

## Prerequisites

- **Go 1.24+** - the module declares `go 1.24` (see `go.mod`)
- **API Keys** - At least one provider API key (OpenAI, Anthropic, etc.)
- **C toolchain** - DuckDB analytics requires cgo; ensure a working `gcc`/`clang` on the host

## Building from Source

Use the installer for a turnkey build:

```bash
git clone https://github.com/Skelf-Research/route-switch.git
cd route-switch
./install.sh
```

Or build manually:

```bash
git clone https://github.com/Skelf-Research/route-switch.git
cd route-switch
go build -o route-switch
```

Verify the installation:

```bash
./route-switch --help
```

## Configuration

Create a configuration file named `config.yaml`:

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

## Environment Variables

The config manager (`internal/config/manager.go`) overlays the following environment variables on top of `config.yaml`. Provider keys are checked in order — the `ROUTE_SWITCH_*` form wins over the standard form, which wins over whatever is in the file.

| Variable | Description |
|----------|-------------|
| `ROUTE_SWITCH_<PROVIDER>_API_KEY` | Provider-scoped override, e.g. `ROUTE_SWITCH_OPENAI_API_KEY` |
| `OPENAI_API_KEY` | Standard OpenAI key fallback |
| `ANTHROPIC_API_KEY` | Standard Anthropic key fallback |
| `GOOGLE_API_KEY` / `GEMINI_API_KEY` | Google / Gemini key fallback |
| `COHERE_API_KEY` | Cohere key fallback |
| `MISTRAL_API_KEY` | Mistral key fallback |
| `GROQ_API_KEY` | Groq key fallback |
| `ROUTE_SWITCH_ANALYTICS_PATH` | Override DuckDB analytics file path |
| `ROUTE_SWITCH_DATASET_PATH` | Override dataset base path |

This matches the [`env_only`](https://github.com/Skelf-Research/route-switch/blob/main/internal/config/manager.go) secrets posture: nothing is read from a keyring or encrypted file.

## Makefile Targets

The repository ships a `Makefile` for common workflows:

| Target | Description |
|--------|-------------|
| `make build` | `go build -o route-switch` |
| `make test` | `go test ./...` |
| `make test-coverage` | `go test -cover ./...` |
| `make example` | Run `./example.sh` |
| `make clean` | Remove the binary |
| `make deps` | `go mod tidy` |
| `make run` | Build then run the binary |

## Directory Structure

After installation, Route-Switch expects the following structure:

```
project/
├── route-switch          # Binary
├── config.yaml           # Configuration
└── data/
    └── prompts/          # Per-prompt datasets
```

The `data/prompts/` directory is created automatically when you first use Route-Switch.

## Next Steps

Once installed, proceed to the [Quick Start](quickstart.md) guide to run your first optimization.
