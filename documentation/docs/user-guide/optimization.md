# Optimization

Route-Switch uses MIPROv2-style optimization to improve prompt performance. Production traffic serves as the calibration/evaluation set, enabling continuous improvement.

## How It Works

1. **Capture** - Every invocation is stored in a per-prompt SQLite database
2. **Bootstrap** - Load recent records as calibration data
3. **Generate** - Create instruction candidates through provider calls
4. **Optimize** - Run Bayesian optimization across combinations
5. **Evaluate** - Replay dataset rows with the configured evaluation strategy
6. **Deploy** - Update the gateway with the optimized prompt

## CLI Optimization

### Optimize a Prompt

```bash
./route-switch --config config.yaml \
  --prompt "Explain {topic} to a 10 year old" \
  --model gpt-4 \
  --provider gollm \
  --optimize-prompt
```

The CLI outputs:

- Bootstrapping progress
- Instruction search status
- Bayesian trial results
- Final optimized prompt
- Improvement score and cost estimate

### Find Best Model

Search across all configured models:

```bash
./route-switch --config config.yaml \
  --prompt "Summarize the latest AI trends" \
  --model gpt-4 \
  --provider gollm \
  --find-best-model
```

## Evaluation Strategies

Route-Switch includes three built-in evaluation strategies:

### Similarity (Default)

Best for creative or long-form outputs.

- Uses token overlap and length heuristics
- Default threshold: 0.7
- Handles variable output formats well

### Exact Match

Best for deterministic answers.

- Trims whitespace before comparison
- Use for unit test-style validations
- Binary pass/fail scoring

### Keyword Match

Best when key concepts must appear.

- Accepts pre-defined keywords
- Can derive keywords from expected output
- Partial scoring based on keyword presence

## Configuration

```yaml
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
```

### MIPROv2 Parameters

| Parameter | Description |
|-----------|-------------|
| `num_candidates` | Number of final prompt candidates to evaluate |
| `max_bootstrapped_demos` | Maximum examples from dataset for bootstrapping |
| `max_labeled_demos` | Maximum labeled examples for training |
| `num_trials` | Number of Bayesian optimization trials |
| `minibatch_size` | Size of evaluation minibatches |
| `minibatch_full_eval_steps` | Full evaluation frequency |
| `num_instruction_candidates` | Instruction proposals to generate |
| `evaluation_strategy` | Strategy for this optimization run |

### Evaluation Parameters

| Parameter | Description |
|-----------|-------------|
| `default_strategy` | Global default (`Similarity`, `Keyword`, `ExactMatch`) |
| `threshold` | Success/failure threshold (0-1) |
| `max_retries` | Retry count for failed evaluations |

## CLI Override

Override the strategy per run:

```bash
./route-switch --config config.yaml \
  --prompt "Answer: {question}" \
  --model gpt-4 \
  --optimize-prompt \
  --evaluation-strategy exact
```

## Custom Strategies

Implement the `EvaluationStrategy` interface:

```go
type EvaluationStrategy interface {
    Evaluate(prompt, expected, actual string, model models.Model) (*EvaluationResult, error)
    Name() string
}
```

You can integrate domain-specific metrics like BLEU, ROUGE, or classifier outputs.

## Background Optimization

The gateway can run optimization automatically:

```yaml
gateway:
  optimization:
    enabled: true
    interval_seconds: 1800
```

The background optimizer:

1. Identifies prompts needing refresh (based on `LastOptimized`)
2. Loads fresh production traces
3. Runs MIPROv2
4. Updates the prompt registry
5. Logs results to analytics

## Dataset Storage

Each prompt's dataset stores:

| Field | Description |
|-------|-------------|
| `rendered_input` | Final prompt sent to model |
| `output` | Model response |
| `variables` | Variable bindings used |
| `model` | Model name |
| `provider` | Provider name |
| `success` | Whether the call succeeded |
| `cost` | Estimated cost |
| `created_at` | Timestamp |

Records are automatically trimmed to `dataset.max_records`.

## Optimization Workflow

```
User Request → Gateway → Provider Response
                 ↓
           Dataset Store
                 ↓
         Background Optimizer
                 ↓
           MIPROv2 Loop
                 ↓
         Updated Prompt
                 ↓
         Gateway Registry
```

## Best Practices

1. **Accumulate data first** - Run at least 100 requests before optimizing
2. **Match strategies to use case** - Use Exact Match for deterministic tasks
3. **Set appropriate thresholds** - Lower thresholds for creative tasks
4. **Monitor improvement scores** - Track optimization effectiveness over time
5. **Use background optimization** - Let the system improve continuously
