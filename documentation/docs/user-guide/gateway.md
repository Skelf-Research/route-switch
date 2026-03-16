# Gateway

The Route-Switch gateway provides an OpenAI-compatible inference endpoint with automatic prompt optimization, load balancing, and analytics.

## Starting the Gateway

```bash
./route-switch --config config.yaml \
  --gateway \
  --provider gollm \
  --prompt "Default onboarding prompt" \
  --model gpt-4 \
  --addr :8080
```

## Endpoints

### POST /v1/chat/completions

Primary inference endpoint accepting OpenAI-style chat payloads.

**Request:**

```json
{
  "model": "gpt-4",
  "messages": [
    {"role": "user", "content": "Tell me about renewable energy"}
  ],
  "stream": false,
  "variables": {
    "topic": "solar power",
    "tone": "friendly"
  }
}
```

**Response:**

```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1702000000,
  "model": "gpt-4",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Solar power is a fascinating topic..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 50,
    "completion_tokens": 150,
    "total_tokens": 200
  }
}
```

### Streaming Responses

Enable streaming with `"stream": true`:

```json
{
  "model": "gpt-4",
  "messages": [{"role": "user", "content": "Hello"}],
  "stream": true
}
```

Responses follow the OpenAI chunk format with usage totals in the final chunk.

### GET /health

Lightweight readiness probe.

```json
{"status": "healthy"}
```

### GET /status

Operational snapshot of all prompt/model combinations.

```json
{
  "status": "ok",
  "count": 2,
  "combinations": [
    {
      "id": "support-flow-gpt4",
      "template_id": "support-flow",
      "model": "gpt-4",
      "provider": "gollm",
      "weight": 10,
      "performance": {
        "success_rate": 0.98,
        "response_time_avg": 1.2,
        "total_requests": 1042
      }
    }
  ]
}
```

### GET /v1/prompts/{id}/stats

Per-prompt statistics.

```json
{
  "prompt_id": "support-flow",
  "template_id": "support-flow",
  "total_requests": 1042,
  "success_rate": 0.98,
  "avg_latency_ms": 1200,
  "avg_cost": 0.0023,
  "error_count": 21
}
```

### GET /v1/system/analytics

Global analytics across all prompts.

```json
{
  "total_prompts": 5,
  "total_requests": 5412,
  "success_rate": 0.94,
  "avg_latency_ms": 1320,
  "avg_cost": 0.0017
}
```

### GET /health/storage

Verifies dataset and analytics backends.

## Load Balancing Strategies

Configure the strategy in `config.yaml`:

```yaml
gateway:
  strategy: "performance_based"
```

| Strategy | Description |
|----------|-------------|
| `round_robin` | Rotate through combinations evenly |
| `weighted_round_robin` | Respect combination weights |
| `performance_based` | Route based on success rate and latency |

## Prompt Combinations

Define static combinations in config:

```yaml
gateway:
  combinations:
    - id: "default"
      name: "default"
      prompt: "Default onboarding prompt: {topic}"
      model: "gpt-4"
      provider: "openai"
      is_primary: true
      weight: 10
    - id: "fallback"
      name: "fallback"
      prompt: "Default onboarding prompt: {topic}"
      model: "gpt-3.5-turbo"
      provider: "openai"
      weight: 5
```

## Background Optimization

Enable automatic optimization:

```yaml
gateway:
  optimization:
    enabled: true
    interval_seconds: 1800
```

The background optimizer periodically:

1. Selects prompts needing refresh
2. Runs MIPROv2 with fresh production traces
3. Updates the registry with optimized prompts

## Fallback Configuration

Configure automatic fallback when combinations underperform:

```yaml
gateway:
  fallback_threshold: 0.8
```

When a combination's success rate drops below 0.8, its weight is reduced and fallbacks are promoted.

## Request Metadata

Target specific combinations or templates via metadata:

```json
{
  "model": "gpt-4",
  "messages": [...],
  "metadata": {
    "combination_id": "combo-1"
  }
}
```

Or by template:

```json
{
  "metadata": {
    "template_id": "support-flow"
  }
}
```

## Rate Limiting

Configure per-provider rate limits:

```yaml
model_providers:
  openai:
    rate_limit: 1200  # requests per minute
```

The gateway enforces these limits to prevent upstream quota violations.
