This is a golang program that implements MIPROv2 for prompt optimisation and model switching

The user will provide a working prompt with an existing model, we will then attempt to optimise the prompt
The other usecase is when there is a working prompt with a given model, we will attempt to create a prompt + cheapest model

## Implementation Summary

We have successfully implemented a modular Go application that fulfills the above requirements with the following architecture:

### Modules:
1. **CLI** - Command-line interface using Cobra for user interaction
2. **Core** - Main service layer that orchestrates the functionality
3. **Models** - Model definitions and provider interfaces with a mock implementation
4. **Optimizer** - Full MIPROv2 implementation for prompt optimization
5. **Utils** - Utility functions including cost calculation

### Features Implemented:
- Complete MIPROv2 algorithm with all three steps:
  1. Bootstrap Few-Shot Examples
  2. Propose Instruction Candidates
  3. Find an Optimized Combination using Bayesian Optimization
- Prompt optimization using the full MIPROv2 algorithm
- Model switching to find the best (cheapest) model for a given prompt
- Cost calculation to determine the most cost-effective solution
- Comprehensive test coverage for all components

### MIPROv2 Implementation Details:

1. **Bootstrap Few-Shot Examples**: Randomly samples examples from a training set and validates them through the LM program to create valid few-shot example candidates.

2. **Propose Instruction Candidates**: Generates instruction candidates based on training dataset properties, LM program code, bootstrapped examples, and randomly sampled tips for generation.

3. **Optimize Combination**: Uses Bayesian Optimization to find the best combinations of instructions and demonstrations for each predictor in the program.

### Usage:
```bash
# Optimize a prompt for a specific model
./route-switch --prompt "Write a poem about programming" --model "gpt-4" --optimize-prompt

# Find the best model and optimized prompt combination
./route-switch --prompt "Write a poem about programming" --model "gpt-4" --find-best-model
```

The application is ready for extension with real model providers (OpenAI, Anthropic, etc.) and can be enhanced with more sophisticated optimization algorithms.
