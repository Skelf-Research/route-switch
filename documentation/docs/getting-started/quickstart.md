# Quick Start

This guide walks you through your first prompt optimization and gateway deployment with Route-Switch.

## Step 1: Optimize a Prompt

Run your first prompt optimization:

```bash
./route-switch --config config.yaml \
  --prompt "Write a poem about programming" \
  --model "gpt-4" \
  --provider gollm \
  --optimize-prompt
```

Route-Switch will:

1. Bootstrap samples from any existing dataset
2. Generate instruction candidates
3. Run Bayesian optimization trials
4. Output the optimized prompt and improvement score

## Step 2: Find the Best Model

Search across all configured models to find the best performer:

```bash
./route-switch --config config.yaml \
  --prompt "Summarize the latest AI news" \
  --model "gpt-4" \
  --provider gollm \
  --find-best-model
```

The optimizer iterates over every provider/model in your configuration and reports the best-performing prompt/model pair.

## Step 3: Register a Template

Create a reusable prompt template:

```bash
./route-switch template register \
  --template-id support-flow \
  --name "Support Onboarding Flow" \
  --prompt-text "Help {customer_name} with their {issue}. Ticket: {ticket_id}" \
  --variables customer_name,ticket_id,issue \
  --config config.yaml
```

This persists the template under `data/prompts/support-flow/manifest.yaml`.

## Step 4: Start the Gateway

Launch the OpenAI-compatible gateway:

```bash
./route-switch --config config.yaml \
  --gateway \
  --provider gollm \
  --prompt "Default onboarding prompt" \
  --model gpt-4 \
  --addr :8080
```

The gateway exposes these endpoints:

| Endpoint | Description |
|----------|-------------|
| `POST /v1/chat/completions` | OpenAI-compatible inference |
| `GET /health` | Readiness probe |
| `GET /status` | Prompt/model combinations status |
| `GET /v1/prompts/{id}/stats` | Per-prompt statistics |
| `GET /v1/system/analytics` | Global analytics |

## Step 5: Make a Request

Send a request to your gateway:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Please help."}],
    "variables": {
      "customer_name": "Jordan",
      "issue": "account locked"
    }
  }'
```

Route-Switch will:

1. Render the template with your variables
2. Select a model using the configured strategy
3. Forward the request to the provider
4. Log the invocation for future optimization

## Step 6: Check Status

View the operational status of your gateway:

```bash
curl http://localhost:8080/status
```

Example response:

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

## Next Steps

- Learn about [Prompt Templates](../user-guide/templates.md) in depth
- Configure the [Gateway](../user-guide/gateway.md) for production
- Set up [Portable Packages](../user-guide/packages.md) for deployment
