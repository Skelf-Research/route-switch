# API Reference

Route-Switch exposes HTTP endpoints through the gateway and Go packages for embedding.

## HTTP Endpoints

### POST /v1/chat/completions

Primary inference endpoint with OpenAI-compatible format.

**Request:**

```json
{
  "model": "gpt-4",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Tell me about renewable energy"}
  ],
  "stream": false,
  "temperature": 0.7,
  "max_tokens": 1000,
  "variables": {
    "topic": "solar power",
    "tone": "friendly"
  },
  "metadata": {
    "template_id": "support-flow"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `model` | string | Yes | Model or prompt ID |
| `messages` | array | Yes | Chat messages |
| `stream` | bool | No | Enable streaming (default: false) |
| `temperature` | float | No | Sampling temperature |
| `max_tokens` | int | No | Maximum tokens |
| `variables` | object | No | Template variable values |
| `metadata` | object | No | Request metadata |

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

**Streaming Response:**

When `stream: true`, returns Server-Sent Events:

```
data: {"id":"chatcmpl-abc","choices":[{"delta":{"content":"Hello"}}]}

data: {"id":"chatcmpl-abc","choices":[{"delta":{"content":" world"}}]}

data: {"id":"chatcmpl-abc","choices":[{"finish_reason":"stop"}],"usage":{"total_tokens":10}}

data: [DONE]
```

---

### GET /health

Basic readiness probe.

**Response:**

```json
{
  "status": "healthy"
}
```

---

### GET /health/storage

Verify dataset and analytics backends.

**Response:**

```json
{
  "status": "healthy",
  "dataset": "ok",
  "analytics": "ok"
}
```

---

### GET /status

Operational snapshot of all prompt/model combinations.

**Response:**

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
      "is_primary": true,
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

---

### GET /v1/prompts/{id}/stats

Per-prompt statistics.

**Path Parameters:**

| Parameter | Description |
|-----------|-------------|
| `id` | Template ID or combination ID |

**Response:**

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

---

### GET /v1/system/analytics

Global analytics summary.

**Response:**

```json
{
  "total_prompts": 5,
  "total_requests": 5412,
  "success_rate": 0.94,
  "avg_latency_ms": 1320,
  "avg_cost": 0.0017
}
```

---

## Request Metadata

### Targeting Templates

```json
{
  "metadata": {
    "template_id": "support-flow"
  }
}
```

### Targeting Combinations

```json
{
  "metadata": {
    "combination_id": "support-flow-gpt4"
  }
}
```

---

## Go Packages

### Core Service

`internal/core`

```go
import "route-switch/internal/core"

// Create service
svc := core.NewService(&core.ServiceConfig{
    Provider:    provider,
    Optimizer:   optimizer,
    Config:      cfg,
})

// Optimize prompt
result, err := svc.OptimizePrompt("Write a poem about {topic}", "gpt-4")

// Find best model
result, err := svc.FindBestModel("Summarize {content}", "gpt-4")
```

**Result struct:**

```go
type Result struct {
    OriginalPrompt   string
    OptimizedPrompt  string
    Model            string
    Cost             float64
    ImprovementScore float64
    Metadata         map[string]any
}
```

---

### Dataset Store

`internal/storage/dataset`

```go
import "route-switch/internal/storage/dataset"

// Create store
store, err := dataset.NewSQLiteStore("data/prompts", 1000)

// Add record
err = store.AddRecord(ctx, "support-flow", &dataset.Record{
    Input:     "Hello {name}",
    Output:    "Hi there!",
    Variables: map[string]string{"name": "Jordan"},
    Model:     "gpt-4",
    Provider:  "openai",
    Success:   true,
    Cost:      0.002,
})

// List recent records
records, err := store.ListRecent(ctx, "support-flow", 100)
```

**Record struct:**

```go
type Record struct {
    ID        string
    Input     string
    Output    string
    Variables map[string]string
    Model     string
    Provider  string
    Success   bool
    Cost      float64
    CreatedAt time.Time
}
```

---

### Optimizer

`internal/optimizer`

```go
import "route-switch/internal/optimizer"

// Create MIPROv2 optimizer
opt := optimizer.NewMIPROv2(
    provider,
    evaluator,
    bayesianOpt,
    &optimizer.MIPROv2Config{
        NumCandidates:            5,
        MaxBootstrappedDemos:     3,
        NumTrials:                20,
        NumInstructionCandidates: 4,
    },
)

// Run optimization
result, err := opt.Optimize(ctx, prompt, dataset)
```

---

### Model Providers

`internal/models`

```go
import "route-switch/internal/models"

type ModelProvider interface {
    ListModels() ([]Model, error)
    GetModel(name string) (Model, error)
    CallModel(ctx context.Context, model Model, prompt string) (string, error)
    EstimateCost(model Model, inputTokens, outputTokens int) float64
    GetTokenCount(text string) int
    Initialize(config map[string]any) error
    Close() error
}
```

**Available providers:**

- `GollmProvider` - Multi-provider via gollm
- `MockProvider` - Testing provider

---

### Analytics

`internal/analytics`

```go
import "route-switch/internal/analytics"

// Create store
store, err := analytics.NewDuckDBStore("data/analytics/metrics.duckdb")

// Record invocation
err = store.RecordInvocation(ctx, &analytics.Invocation{
    PromptID:  "support-flow",
    Model:     "gpt-4",
    Success:   true,
    LatencyMs: 1200,
    Cost:      0.002,
})

// Query stats
stats, err := store.QueryPromptStats(ctx, "support-flow")
systemStats, err := store.QuerySystemStats(ctx)
```

**Store interface:**

```go
type Store interface {
    RecordInvocation(ctx context.Context, inv *Invocation) error
    QueryPromptStats(ctx context.Context, promptID string) (*PromptStats, error)
    QuerySystemStats(ctx context.Context) (*SystemStats, error)
    Close() error
}
```

---

### Evaluation Strategies

`internal/models`

```go
type EvaluationStrategy interface {
    Evaluate(prompt, expected, actual string, model Model) (*EvaluationResult, error)
    Name() string
}

type EvaluationResult struct {
    Score   float64
    Correct bool
    Details map[string]any
}
```

**Built-in strategies:**

- `SimilarityEvaluator` - Token overlap scoring
- `ExactMatchEvaluator` - Exact string matching
- `KeywordMatchEvaluator` - Keyword presence checking
