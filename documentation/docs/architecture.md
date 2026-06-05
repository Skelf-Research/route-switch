# Architecture

Route-Switch is a modular prompt function platform. A single prompt template becomes the source of truth, production traffic seeds the dataset, and every service (optimizer, router, analytics, packaging) consumes the same captured data.

## System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Route-Switch                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐  │
│  │   CLI/API    │───▶│   Gateway    │───▶│  Model Providers     │  │
│  │              │    │  (Proxy)     │    │  (OpenAI, Anthropic) │  │
│  └──────────────┘    └──────┬───────┘    └──────────────────────┘  │
│                             │                                        │
│         ┌───────────────────┼───────────────────┐                   │
│         ▼                   ▼                   ▼                   │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐          │
│  │   Template   │    │   Dataset    │    │  Analytics   │          │
│  │   Registry   │    │    Store     │    │    Store     │          │
│  │   (YAML)     │    │  (SQLite)    │    │  (DuckDB)    │          │
│  └──────────────┘    └──────┬───────┘    └──────────────┘          │
│                             │                                        │
│                             ▼                                        │
│                      ┌──────────────┐                               │
│                      │   MIPROv2    │                               │
│                      │  Optimizer   │                               │
│                      └──────────────┘                               │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Prompt Registration

- Stores template text, variable schema, and provider configuration
- CLI `template register` persists manifests under the dataset base path
- Templates travel with their metadata wherever the dataset goes

**Location:** `internal/templates`

### 2. Dataset Storage

- One SQLite database per prompt under `dataset.base_path`
- Records: rendered input, output, variables, cost/success flags, timestamps
- Retention bounded via `dataset.max_records`

**Location:** `internal/storage/dataset`

### 3. Optimizer & Evaluation

The MIPROv2 optimization loop:

1. Bootstrap samples from the per-prompt dataset
2. Generate instruction candidates through provider calls
3. Run Bayesian optimization (goptuna) across combinations
4. Evaluate candidates by replaying dataset rows
5. Produce optimized prompt, score, and metadata

**Location:** `internal/optimizer`

### 4. Gateway & Proxy

- Maintains prompt registry, load balancer, and OpenAI-compatible proxy
- Strategies: round-robin, weighted, performance-based
- Injects optimized prompts, calls providers, records results

**Location:** `internal/gateway`

### 5. Analytics & Observability

- DuckDB-backed analytics store
- Collects request/response summaries, latency, error rates, cost
- Powers `/status` and `/v1/system/analytics` endpoints

**Location:** `internal/analytics`

### 6. Portable Packages

Each prompt directory bundles:

- `manifest.yaml` - template metadata
- `package.yaml` - bundle manifest
- SQLite dataset snapshot
- Recent logs
- Optional DuckDB analytics excerpt

**Location:** `internal/packaging`

## Data Flow

### Request Flow

```
User Request
     │
     ▼
┌─────────────┐
│   Gateway   │
│   Proxy     │
└─────┬───────┘
      │
      ├──────────────────────┐
      ▼                      ▼
┌─────────────┐       ┌─────────────┐
│   Render    │       │    Load     │
│   Template  │       │  Balancer   │
└─────┬───────┘       └─────┬───────┘
      │                     │
      └──────────┬──────────┘
                 ▼
          ┌─────────────┐
          │   Provider  │
          │    Call     │
          └─────┬───────┘
                │
      ┌─────────┴─────────┐
      ▼                   ▼
┌─────────────┐    ┌─────────────┐
│   Dataset   │    │  Analytics  │
│    Store    │    │    Store    │
└─────────────┘    └─────────────┘
```

### Optimization Flow

```
Dataset Store
     │
     ▼
┌─────────────┐
│  Bootstrap  │
│   Samples   │
└─────┬───────┘
      │
      ▼
┌─────────────┐
│  Generate   │
│ Candidates  │
└─────┬───────┘
      │
      ▼
┌─────────────┐
│  Bayesian   │
│   Search    │
└─────┬───────┘
      │
      ▼
┌─────────────┐
│  Evaluate   │
│  & Score    │
└─────┬───────┘
      │
      ▼
┌─────────────┐
│   Update    │
│  Registry   │
└─────────────┘
```

## Module Structure

```
internal/
├── analytics/          # DuckDB analytics store
│   ├── store.go        # Interface
│   └── duckdb_store.go # Implementation
│
├── cli/                # CLI commands
│   ├── root.go         # Main CLI
│   ├── template_cmd.go # Template commands
│   ├── package_cmd.go  # Package commands
│   └── gateway_cmd.go  # Gateway commands
│
├── config/             # Configuration
│   ├── config.go       # Types
│   └── manager.go      # Loading/validation
│
├── core/               # Core service
│   └── service.go      # Business logic
│
├── gateway/            # HTTP gateway
│   ├── gateway.go      # Main gateway
│   ├── proxy_server.go # OpenAI proxy
│   ├── load_balancer.go# Load balancing
│   └── prompt_registry.go
│
├── models/             # Model providers + evaluators
│   ├── models.go       # Interfaces
│   ├── gollm_provider.go
│   ├── mock_provider.go
│   ├── evaluation_factory.go
│   ├── similarity_eval.go
│   ├── keyword_match_eval.go
│   └── exact_match_eval.go
│
├── optimizer/          # Optimization
│   ├── interfaces.go   # Interfaces
│   ├── mipro_v2.go     # MIPROv2 impl
│   └── bayesian_optimizer.go
│
├── packaging/          # Package export/import
│   ├── exporter.go
│   ├── importer.go
│   └── types.go
│
├── storage/
│   └── dataset/        # Per-prompt datasets
│       ├── store.go    # Interface
│       └── sqlite_store.go
│
├── templates/          # Template management
│   └── manager.go
│
└── utils/              # Utilities
    ├── cost.go
    └── logging.go
```

## Provider Integration

The `ModelProvider` interface abstracts all LLM providers:

```go
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

Providers handle:

- Token counting
- Cost estimation
- Lifecycle hooks

New providers implement this interface without modifying core logic.

## Background Optimization

`BackgroundOptimizer` periodically:

1. Selects prompts needing refresh (based on `LastOptimized`)
2. Reruns MIPROv2 with latest dataset slice
3. Writes results to registry
4. Logs to analytics
5. Optionally commits to package repository

## Design Principles

1. **Single source of truth** - Templates define everything
2. **Production-driven** - Real traffic drives optimization
3. **Modular** - Components can be swapped independently
4. **Observable** - Analytics built into every operation
5. **Portable** - Packages enable migration and recovery
