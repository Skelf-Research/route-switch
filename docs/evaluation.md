# Evaluation & Optimization

Route-Switch reuses real production traffic as the calibration/evaluation set for MIPROv2. Every invocation stored in the per-prompt SQLite database can act as:
- **bootstrapped examples** – seed the optimizer with real user prompts and expected outputs.
- **validation set** – replay historical requests against candidate prompts to compute accuracy, cost, and latency scores.

## Workflow
1. Capture request/response pairs in `dataset.Record` (triggered automatically by CLI/gateway usage).
2. When optimizing for a new model, load the most recent records and split them into calibration vs. validation slices.
3. Evaluate each candidate prompt using one of the built-in strategies:
   - Similarity (default)
   - Exact match
   - Keyword match
   - Custom strategy
4. Feed evaluation results into the Bayesian optimizer, ensuring infeasible candidates (token limit, cost budget) are discarded.

## Built-in Strategies
| Strategy      | Use When                              | Notes |
|---------------|---------------------------------------|-------|
| Similarity    | Creative or long-form outputs         | Uses token overlap + length heuristics, threshold defaults to 0.7 |
| Exact Match   | Deterministic answers (unit tests)    | Trims whitespace before comparison |
| Keyword Match | Key concepts must appear in the reply | Accepts pre-defined keywords or derives them from expected output |

All strategies implement `EvaluationStrategy` and return `EvaluationResult { Score, Correct, Details }`.

## Configuration
```yaml
mipro_v2:
  evaluation_strategy: "Similarity"
evaluation:
  default_strategy: "Similarity"
  threshold: 0.7
  max_retries: 3
```
Set `mipro_v2.evaluation_strategy` to override the optimizer-specific strategy while retaining a global default for runtime health checks.

## Custom Strategies
Implement the `EvaluationStrategy` interface:
```go
type EvaluationStrategy interface {
    Evaluate(prompt, expected, actual string, model models.Model) (*EvaluationResult, error)
    Name() string
}
```
You can ingest domain-specific metrics (BLEU, ROUGE, classifier outputs, etc.) and plug them into the optimizer by updating the configuration or injecting the evaluator when constructing `core.Service`.
