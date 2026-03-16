# Prompt Templates

Prompt templates are the foundation of Route-Switch. They define reusable prompts with variables that can be optimized and deployed across multiple models.

## Overview

A prompt template consists of:

- **Template ID** - Unique identifier for the template
- **Name** - Human-readable name
- **Prompt text** - The prompt with variable placeholders
- **Variables** - Named placeholders that are replaced at runtime
- **Default model/provider** - Baseline configuration

## Registering Templates

### Using the CLI

Register a template from a file:

```bash
./route-switch template register \
  --template-id support-flow \
  --name "Support Onboarding Flow" \
  --prompt-file prompts/support.txt \
  --variables customer_name,ticket_id,issue \
  --config config.yaml
```

Or inline:

```bash
./route-switch template register \
  --template-id support-flow \
  --name "Support Onboarding Flow" \
  --prompt-text "Help {customer_name} with their {issue}. Ticket: {ticket_id}" \
  --variables customer_name,ticket_id,issue \
  --default-model gpt-4 \
  --default-provider gollm \
  --config config.yaml
```

### Template Options

| Flag | Description |
|------|-------------|
| `--template-id` | Unique identifier (required) |
| `--name` | Display name (required) |
| `--prompt-file` | Path to prompt text file |
| `--prompt-text` | Inline prompt text |
| `--variables` | Comma-separated variable names |
| `--default-model` | Default model for this template |
| `--default-provider` | Default provider for this template |

## Variable Syntax

Variables use curly brace syntax: `{variable_name}`

```
Help {customer_name} with their {issue}.
Their ticket ID is {ticket_id}.
Please respond in a {tone} manner.
```

## Listing Templates

View all registered templates:

```bash
./route-switch template list --config config.yaml
```

## Using Templates

### In CLI Mode

Use a registered template with `--template-id`:

```bash
./route-switch --config config.yaml \
  --template-id support-flow \
  --optimize-prompt
```

This loads the stored prompt, default model, and provider from the manifest.

### In Gateway Mode

Include a `variables` object in your request:

```json
{
  "model": "gpt-4",
  "messages": [{"role": "user", "content": "Please help."}],
  "variables": {
    "customer_name": "Jordan",
    "ticket_id": "TKT-12345",
    "issue": "account locked"
  }
}
```

Route-Switch replaces the placeholders before invoking the provider.

### Targeting Specific Templates

Include metadata to target a specific template:

```json
{
  "model": "gpt-4",
  "messages": [{"role": "user", "content": "Help needed"}],
  "metadata": {
    "template_id": "support-flow"
  }
}
```

## Manifest Structure

Templates are stored as YAML manifests:

```yaml
id: support-flow
name: Support Onboarding Flow
prompt: |
  Help {customer_name} with their {issue}.
  Ticket: {ticket_id}
variables:
  - customer_name
  - ticket_id
  - issue
default_model: gpt-4
default_provider: gollm
metadata:
  optimized: false
  evaluation_strategy: Similarity
created_at: 2024-12-08T10:00:00Z
```

Manifests are stored at `data/prompts/<template-id>/manifest.yaml`.

## Dataset Integration

Every template has an associated SQLite database that stores:

- Rendered prompt input/output
- Variable bindings supplied at invocation
- Provider/model metadata
- Success flags and cost estimates
- Timestamps

This dataset powers the MIPROv2 optimization loop.

## Best Practices

1. **Use descriptive variable names** - `{customer_name}` is better than `{name}`
2. **Keep prompts focused** - One template per use case
3. **Include context variables** - Add variables for tone, format, etc.
4. **Test with mock provider** - Verify variable rendering before production
