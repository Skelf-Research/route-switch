# API Reference

Route-Switch exposes both HTTP endpoints (through the gateway) and Go packages you can embed in your own services.

## HTTP Gateway

### `POST /v1/chat/completions`
- **Purpose**: primary inference endpoint. Accepts OpenAI-style chat payloads.
- **Body**:
  ```json
  {
    "model": "prompt-id-or-model",
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
- **Behavior**: the gateway locates the prompt combination, renders the template by replacing `{topic}`/`{tone}` placeholders with the supplied `variables`, selects a model based on the configured strategy, and returns an OpenAI-compatible response payload including usage stats. Each invocation (prompt, output, variables, metadata, result) is appended to the per-prompt dataset for future optimization.

### `GET /health`
- Basic readiness endpoint returning `{ "status": "healthy" }` when the proxy, registry, and optimizer are initialized.

### `GET /status`
- **Purpose**: operational snapshot of every registered prompt/model combo.
- **Response**:
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
        },
        "analytics": {
          "total_requests": 1042,
          "success_rate": 0.98,
          "avg_latency_ms": 1200,
          "avg_cost": 0.0023
        }
      }
    ]
  }
  ```

### `GET /v1/prompts/{id}/stats`
- **Purpose**: retrieve aggregated metrics for a single template ID or combination ID.
- **Path Parameters**: `id` – template ID (preferred) or combination ID.
- **Response**:
  ```json
  {
    "prompt_id": "support-flow",
    "template_id": "support-flow",
    "total_requests": 1042,
    "success_rate": 0.98,
    "avg_latency_ms": 1200,
    "avg_cost": 0.0023,
    "error_count": 21,
    "first_seen": "2024-12-05T18:13:00Z",
    "last_seen": "2024-12-08T22:05:14Z"
  }
  ```

### `GET /v1/system/analytics`
- **Purpose**: summarize request volume/cost across every template.
- **Response**:
  ```json
  {
    "total_prompts": 5,
    "total_requests": 5412,
    "success_rate": 0.94,
    "avg_latency_ms": 1320,
    "avg_cost": 0.0017
  }
  ```

## Go Packages

### Core Service (`internal/core`)
- `core.NewService(cfg *ServiceConfig) *Service`
- `service.OptimizePrompt(prompt, model string) (*Result, error)`
- `service.FindBestModel(prompt, baseModel string) (*Result, error)`

`Result` contains `OriginalPrompt`, `OptimizedPrompt`, `Model`, `Cost`, `ImprovementScore`, and arbitrary metadata.

### Dataset Store (`internal/storage/dataset`)
- `dataset.NewSQLiteStore(basePath string, maxRecords int) (*SQLiteStore, error)`
- `store.AddRecord(ctx, promptID string, record *dataset.Record) error`
- `store.ListRecent(ctx, promptID string, limit int) ([]*dataset.Record, error)`

`Record` captures model name, rendered input/output, variable bindings, metadata, and timestamps. Implement the `Store` interface to plug in other databases.

### Optimizer (`internal/optimizer`)
- `optimizer.NewMIPROv2(provider, evaluator, bayesianOpt, miproCfg)` returns `ExtendedPromptOptimizer`.
- `optimizer.GoptunaBayesianOptimizer` wraps the goptuna study API.

### Model Providers (`internal/models`)
- `ModelProvider` interface exposes `ListModels`, `GetModel`, `CallModel`, `EstimateCost`, `GetTokenCount`, `Initialize`, `Close`.
- Concrete providers: OpenAI, Gollm (multi-provider), Mock. Additional providers can be added by implementing the interface.

### Analytics (`internal/analytics`)
- `analytics.Store` defines `RecordInvocation`, `QueryPromptStats`, and `QuerySystemStats`.
- `analytics.NewDuckDBStore(path)` persists invocations and exposes the statistics powering `/status` and `/v1/system/analytics`.
- Implement the interface if you need to stream analytics data to another warehouse.

Use these APIs from your own Go binaries when you need to embed Route-Switch behavior without running the standalone gateway.
