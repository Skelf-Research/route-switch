# Architecture

Route-Switch is a modular prompt function platform. A single prompt template (with typed variables) becomes the source of truth, production traffic seeds the dataset, and every dependent service (optimizer, router, analytics, packaging) consumes the same captured data.

## System Overview
1. **Prompt registration** – the CLI/API stores a template, variable schema, and primary provider configuration.
2. **Dataset capture** – every invocation is persisted in a per-prompt SQLite store.
3. **Optimization** – the faithful MIPROv2 pipeline replays dataset samples to tune prompts for other models.
4. **Gateway** – requests arrive at a single endpoint; the gateway renders the template, selects a model using the configured strategy, and forwards the call.
5. **Analytics** – invocation metadata flows into a DuckDB-backed analytics store for dashboards and status APIs.
6. **Packaging** – prompts can be exported (manifest + config + datasets + analytics snapshot) for migration to other clusters or disaster recovery.

## Modules

### Prompt Templates & Registration
- Defines template text, variable schema, allowed providers, and operational metadata (weights, tags, SLOs).
- CLI `template register` command persists manifests under the dataset base path so each prompt carries its metadata wherever the dataset travels.
- Stores configuration alongside gateway rules so new models inherit the current routing strategy automatically.

### Dataset Storage
- Implemented by `internal/storage/dataset` using one SQLite database per prompt under `dataset.base_path`.
- Records include rendered input, output, variable payload, cost/success flags, and timestamps.
- Retention is bounded via `dataset.max_records` so datasets stay lightweight but useful for calibration.

### Optimizer & Evaluation
- `internal/optimizer` implements a faithful MIPROv2 loop:
  1. Bootstrap samples from the per-prompt dataset.
  2. Generate instruction candidates through provider calls (or mocks when testing).
  3. Run Bayesian optimization (goptuna) across instruction/example combinations.
  4. Evaluate candidates by replaying dataset rows and applying the configured evaluation strategy (similarity, keyword, exact-match, or custom).
- The optimizer produces an updated prompt, score, and metadata which the gateway can immediately deploy.

### Gateway, Proxy & Load Balancer
- `internal/gateway` maintains a prompt registry, load balancer, and OpenAI-compatible proxy server.
- Supports multiple strategies (round-robin, weighted, performance-based, future least-connections) with deterministic ordering to avoid flaky routing.
- Injects optimized prompts into the user request, calls the appropriate provider (`internal/models`), and records the result in both the dataset store and analytics sink.
- Background optimizer periodically refreshes prompts using fresh production traces.

### Analytics & Observability
- An `AnalyticsStore` abstraction (DuckDB by default) collects request/response summaries, latency, error rates, token/cost usage, and optimization history.
- Status APIs expose per-prompt dashboards (success rate, cost, winning models) and cluster-level health signals.
- The same analytics database powers packaging snapshots and cross-environment comparisons.

### Portable Packages
- Each prompt directory bundles:
  - Manifest (`package.yaml`) with template metadata, variable schema, provider configs, and optimization history.
  - SQLite dataset snapshot and recent logs.
  - Optional DuckDB analytics excerpt and git metadata.
- Packages are git repositories by default so every change to templates or configs is versioned.

## Data Flow
```
user -> gateway (
  render template,
  select model,
  call provider
) -> analytics store
  \-> dataset store (per prompt)
  \-> MIPROv2 optimizer -> gateway registry update
```

## Provider Integration
- `internal/models` exposes a `ModelProvider` interface with concrete adapters: OpenAI, Gollm unified provider, mock provider, etc.
- Providers handle token counting, cost estimation, and lifecycle hooks so the optimizer and gateway stay agnostic.
- Additional providers can be added without touching core logic by implementing the interface and wiring configuration entries.

## Background Optimization
- A `BackgroundOptimizer` goroutine per gateway periodically selects prompts needing refresh (based on `LastOptimized`) and reruns MIPROv2 with the latest dataset slice.
- Results are written back to the registry, logged to analytics, and optionally committed inside the prompt’s package repository.

This architecture keeps prompt quality, routing, and observability tightly integrated while allowing each component (providers, analytics, storage) to be swapped or extended independently.
