# Route-Switch

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/Skelf-Research/route-switch?style=social)](https://github.com/Skelf-Research/route-switch)

**Intelligent LLM routing with automatic prompt optimization**

Route-Switch is a production-ready gateway that routes requests across multiple LLM providers, automatically optimizes prompts using MIPROv2, and tracks performance analytics — all through a single OpenAI-compatible API.

## Features

- **Multi-Provider Routing** — OpenAI, Anthropic, Google, Ollama, Cohere, Mistral, and more via [gollm](https://github.com/teilomillet/gollm)
- **Automatic Prompt Optimization** — MIPROv2 algorithm improves prompts using production traces
- **OpenAI-Compatible API** — Drop-in replacement with `/v1/chat/completions` endpoint
- **Smart Load Balancing** — Round-robin, weighted, performance-based, and least-connections strategies
- **Production Analytics** — DuckDB-backed metrics with per-prompt success rates, latency, and cost tracking
- **Portable Packages** — Export and import prompt templates with datasets across environments

## Quick Start

### Install

```bash
# Clone and build
git clone https://github.com/Skelf-Research/route-switch.git
cd route-switch
./install.sh

# Or build manually
go build -o route-switch
```

### Configure

Create `config.yaml`:

```yaml
model_providers:
  openai:
    api_key: "sk-..."
    models: ["gpt-4o", "gpt-4", "gpt-3.5-turbo"]

gateway:
  addr: ":8080"
  strategy: "round_robin"

dataset:
  base_path: "data/prompts"
```

### Run

```bash
# Start the gateway
./route-switch --config config.yaml --gateway

# Or optimize a prompt
./route-switch --config config.yaml \
  --prompt "Write a poem about programming" \
  --model gpt-4 \
  --optimize-prompt
```

## Gateway API

Send OpenAI-compatible requests:

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

With template variables:

```json
{
  "model": "gpt-4",
  "messages": [{"role": "user", "content": "Help me"}],
  "variables": {
    "customer_name": "Jordan",
    "issue": "account locked"
  }
}
```

### Endpoints

| Endpoint | Description |
|----------|-------------|
| `POST /v1/chat/completions` | OpenAI-compatible inference |
| `GET /status` | All prompt/model combinations with metrics |
| `GET /v1/prompts/{id}/stats` | Stats for a specific template |
| `GET /v1/system/analytics` | Global analytics |
| `GET /health` | Health check |

## CLI Reference

```bash
# Register a prompt template
./route-switch template register \
  --template-id support-flow \
  --name "Support Flow" \
  --prompt-file prompts/support.txt \
  --variables customer_name,ticket_id \
  --config config.yaml

# List templates
./route-switch template list --config config.yaml

# Find the best model for a prompt
./route-switch --config config.yaml \
  --prompt "Summarize this article" \
  --find-best-model

# Export a portable package
./route-switch package export \
  --template-id support-flow \
  --output-dir packages \
  --include-analytics

# Import a package
./route-switch package import \
  --path packages/support-flow-20241208-104500 \
  --config config.yaml
```

## Documentation

- [Architecture](docs/architecture.md)
- [Configuration](docs/configuration.md)
- [CLI Usage](docs/cli.md)
- [API Reference](docs/api-reference.md)
- [Evaluation & Optimization](docs/evaluation.md)

## License

[MIT](LICENSE)
