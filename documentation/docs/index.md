# Route-Switch

**A Go-powered prompt function manager that combines MIPROv2-style optimization, multi-provider routing, and production-aware analytics.**

Route-Switch lets you register a prompt template once, capture real invocations in lightweight per-prompt datasets, and continuously optimize and load-balance traffic across every model you trust.

---

## Key Features

<div class="grid cards" markdown>

-   :material-text-box-edit:{ .lg .middle } **Prompt Templates**

    ---

    Define prompts with variables plus a baseline model. Automatically capture every invocation for future optimization.

    [:octicons-arrow-right-24: Learn more](user-guide/templates.md)

-   :material-chart-line:{ .lg .middle } **MIPROv2 Optimization**

    ---

    Reuse production traces as the calibration set when adapting prompts to additional models or providers.

    [:octicons-arrow-right-24: Learn more](user-guide/optimization.md)

-   :material-server:{ .lg .middle } **Gateway & Inference**

    ---

    Expose a single OpenAI-compatible endpoint that handles prompt rendering, model selection, and failover.

    [:octicons-arrow-right-24: Learn more](user-guide/gateway.md)

-   :material-package-variant:{ .lg .middle } **Portable Packages**

    ---

    Export prompt packages with manifests, logs, and datasets for migration or disaster recovery.

    [:octicons-arrow-right-24: Learn more](user-guide/packages.md)

</div>

---

## Capabilities

| Feature | Description |
|---------|-------------|
| **Prompt registration & templates** | Define prompts with variables plus a baseline model and automatically capture every invocation |
| **Faithful MIPROv2 optimization** | Reuse production traces as the calibration set when adapting a prompt to additional models or providers |
| **Gateway & inference endpoint** | Expose a single OpenAI-compatible endpoint that handles prompt rendering, model selection, and failover |
| **Analytics & packaging** | Store request/response metadata in DuckDB and export portable prompt packages |
| **Extensible providers** | Connect OpenAI, Anthropic, Google, Ollama, Cohere, Mistral, Hugging Face, or mock providers |
| **Provider-aware throttling** | Enforce per-provider RPM limits so production traffic never trips upstream quotas |

---

## Quick Example

```bash
# Build the CLI
go build -o route-switch

# Optimize a prompt
./route-switch --config config.yaml \
  --prompt "Write a poem about programming" \
  --model "gpt-4" \
  --provider gollm \
  --optimize-prompt

# Start the gateway
./route-switch --config config.yaml \
  --gateway \
  --provider gollm \
  --prompt "Default onboarding prompt" \
  --model gpt-4 \
  --addr :8080
```

---

## Getting Started

Ready to get started? Follow these guides:

1. **[Installation](getting-started/installation.md)** - Set up Route-Switch on your system
2. **[Quick Start](getting-started/quickstart.md)** - Run your first optimization and gateway

---

## Supported Providers

Route-Switch connects to upstream providers through the [gollm](https://github.com/teilomillet/gollm) adapter:

- OpenAI (GPT-4, GPT-3.5-turbo, etc.)
- Anthropic (Claude)
- Google (Gemini)
- Ollama (local models)
- Cohere
- Mistral
- Hugging Face
- Mock provider (for testing)

---

## License

Route-Switch is open source software.
