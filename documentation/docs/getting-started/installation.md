# Installation

This guide covers how to install and set up Route-Switch on your system.

## Prerequisites

- **Go 1.21+** - Route-Switch is built with Go
- **API Keys** - At least one provider API key (OpenAI, Anthropic, etc.)

## Building from Source

Clone the repository and build the binary:

```bash
git clone https://github.com/your-org/route-switch.git
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

You can provide API keys via environment variables instead of the config file:

| Variable | Description |
|----------|-------------|
| `OPENAI_API_KEY` | OpenAI API key |
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `GOOGLE_API_KEY` | Google AI API key |

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
