# Route-Switch

Route-Switch is a Go implementation of MIPROv2 for prompt optimization and model switching.

## Features

- Prompt optimization using MIPROv2 algorithm
- Model switching to find the best model for your prompt
- Cost calculation to find the most cost-effective solution

## Installation

```bash
go build -o route-switch
```

## Usage

```bash
# Optimize a prompt for a specific model
./route-switch --prompt "Write a poem about programming" --model "gpt-4" --optimize-prompt

# Find the best model and optimized prompt combination
./route-switch --prompt "Write a poem about programming" --model "gpt-4" --find-best-model

# Display help
./route-switch --help
```

## Examples

### Optimizing a Prompt

```bash
./route-switch --prompt "Write a short story about a robot learning to cook" --model "gpt-4" --optimize-prompt
```

Output:
```
Step 1: Bootstrapping few-shot examples...
Step 2: Proposing instruction candidates...
Step 3: Finding optimized combination...
Optimized Prompt: Instruction: Begin with a brief summary before diving into details

Examples:
Example 1:
Input: Describe the water cycle
Output: The water cycle involves evaporation...

Example 2:
Input: Write a poem about nature
Output: Nature's beauty unfolds in morning light...

Example 3:
Input: Write a poem about nature
Output: Nature's beauty unfolds in morning light...

Task: Write a short story about a robot learning to cook
Model: gpt-4
```

### Finding the Best Model

```bash
./route-switch --prompt "Write a short story about a robot learning to cook" --model "gpt-4" --find-best-model
```

Output:
```
Step 1: Bootstrapping few-shot examples...
Step 2: Proposing instruction candidates...
Step 3: Finding optimized combination...
Optimized Prompt: Instruction: Begin with a brief summary before diving into details

Examples:
Example 1:
Input: Describe the water cycle
Output: The water cycle involves evaporation...

Example 2:
Input: Explain quantum computing
Output: Quantum computing uses quantum bits...

Example 3:
Input: Explain quantum computing
Output: Quantum computing uses quantum bits...

Task: Write a short story about a robot learning to cook
Best Model: gpt-3.5-turbo
Cost: $0.0035
```

## How MIPROv2 Works

MIPROv2 works by creating both few-shot examples and new instructions for each predictor in your LM program, and then searching over these using Bayesian Optimization to find the best combination of these variables for your program.

The steps are:

1) **Bootstrap Few-Shot Examples**: Randomly samples examples from your training set, and run them through your LM program. If the output from the program is correct for this example, it is kept as a valid few-shot example candidate.

2) **Propose Instruction Candidates**: The instruction proposer includes (1) a generated summary of properties of the training dataset, (2) a generated summary of your LM program's code and the specific predictor that an instruction is being generated for, (3) the previously bootstrapped few-shot examples to show reference inputs / outputs for a given predictor and (4) a randomly sampled tip for generation to help explore the feature space of potential instructions.

3) **Find an Optimized Combination**: Finally, we use Bayesian Optimization to choose which combinations of instructions and demonstrations work best for each predictor in our program.

## Architecture

The system is organized into the following modules:

1. **CLI** - Command-line interface for user interaction
2. **Core** - Main service layer that orchestrates the functionality
3. **Models** - Model definitions and provider interfaces
4. **Optimizer** - MIPROv2 implementation for prompt optimization
5. **Utils** - Utility functions including cost calculation

## Development

### Running Tests

```bash
go test ./...
```

### Project Structure

```
route-switch/
├── go.mod
├── main.go
├── README.md
├── SPECS.md
├── internal/
│   ├── cli/
│   │   └── root.go
│   ├── core/
│   │   ├── service.go
│   │   └── service_test.go
│   ├── models/
│   │   ├── mock_provider.go
│   │   ├── mock_provider_test.go
│   │   └── models.go
│   ├── optimizer/
│   │   ├── mipro.go
│   │   ├── mipro_test.go
│   │   └── mipro_v2.go
│   └── utils/
│       ├── cost.go
│       └── cost_test.go
```