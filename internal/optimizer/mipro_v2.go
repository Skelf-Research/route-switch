package optimizer

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/skelf-research/route-switch/internal/config"
	"github.com/skelf-research/route-switch/internal/models"
)

// MIPROv2Config holds configuration parameters for MIPROv2
type MIPROv2Config struct {
	NumCandidates            int // Number of few-shot example candidates to bootstrap
	MaxBootstrappedDemos     int // Maximum number of bootstrapped examples per candidate
	MaxLabeledDemos          int // Maximum number of basic examples per candidate
	NumTrials                int // Number of Bayesian optimization trials
	MinibatchSize            int // Size of minibatch for evaluation
	MinibatchFullEvalSteps   int // Evaluate on full validation set every N steps
	NumInstructionCandidates int // Number of instruction candidates to generate
}

// Example represents a few-shot example
type Example struct {
	Input  string
	Output string
}

// Prompt represents a complete prompt with instructions and examples
type Prompt struct {
	Instruction string
	Examples    []Example
	BasePrompt  string
}

// OptimizationResult holds the results of the MIPROv2 optimization
type OptimizationResult struct {
	BestPrompt        Prompt
	Score             float64
	EvaluationDetails []map[string]interface{}
}

// MIPROv2 implements the MIPROv2 optimization algorithm
type MIPROv2 struct {
	config        config.MiproV2Config
	modelProvider models.ModelProvider
	evaluator     models.EvaluationStrategy
	bayesianOpt   BayesianOptimizer
	rng           *rand.Rand
}

// NewMIPROv2 creates a new instance of MIPROv2 optimizer
func NewMIPROv2(provider models.ModelProvider, evaluator models.EvaluationStrategy, bayesianOpt BayesianOptimizer, config config.MiproV2Config) ExtendedPromptOptimizer {
	return &MIPROv2{
		config:        config,
		modelProvider: provider,
		evaluator:     evaluator,
		bayesianOpt:   bayesianOpt,
		rng:           rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Optimize implements the MIPROv2 prompt optimization algorithm
func (m *MIPROv2) Optimize(prompt string, modelName string) (string, error) {
	// Get model info from provider
	var model models.Model
	if m.modelProvider != nil {
		var err error
		model, err = m.modelProvider.GetModel(modelName)
		if err != nil {
			// Use a basic model struct if provider lookup fails
			model = models.Model{Name: modelName}
		}
	} else {
		model = models.Model{Name: modelName}
	}

	// Step 1: Bootstrap Few-Shot Examples
	fmt.Println("Step 1: Bootstrapping few-shot examples...")
	fewShotCandidates, err := m.bootstrapFewShotExamples(prompt, modelName, nil)
	if err != nil {
		return "", fmt.Errorf("failed to bootstrap few-shot examples: %w", err)
	}

	// Step 2: Propose Instruction Candidates
	fmt.Println("Step 2: Proposing instruction candidates...")
	instructionCandidates, err := m.proposeInstructionCandidates(prompt, modelName, fewShotCandidates)
	if err != nil {
		return "", fmt.Errorf("failed to propose instruction candidates: %w", err)
	}

	// Step 3: Find an Optimized Combination using Bayesian Optimization
	fmt.Println("Step 3: Finding optimized combination...")
	result, err := m.optimizeCombination(prompt, model, fewShotCandidates, instructionCandidates, nil)
	if err != nil {
		return "", fmt.Errorf("failed to optimize combination: %w", err)
	}

	// Construct the final optimized prompt
	optimizedPrompt := m.constructPrompt(result.BestPrompt)
	return optimizedPrompt, nil
}

// bootstrapFewShotExamples implements Step 1 of MIPROv2
func (m *MIPROv2) bootstrapFewShotExamples(basePrompt, modelName string, seedExamples []Example) ([][]Example, error) {
	var candidates [][]Example

	pool := make([]Example, 0, len(seedExamples))
	pool = append(pool, seedExamples...)
	if len(pool) == 0 {
		pool = []Example{
			{Input: "Write a poem about nature", Output: "Nature's beauty unfolds in morning light..."},
			{Input: "Explain quantum computing", Output: "Quantum computing uses quantum bits..."},
			{Input: "Describe the water cycle", Output: "The water cycle involves evaporation..."},
			{Input: "Summarize the Industrial Revolution", Output: "The Industrial Revolution transformed manufacturing..."},
			{Input: "Explain photosynthesis", Output: "Photosynthesis converts light energy into chemical energy..."},
		}
	}

	numCandidates := m.config.NumCandidates
	if numCandidates <= 0 {
		numCandidates = 3
	}

	maxDemos := m.config.MaxBootstrappedDemos
	if maxDemos <= 0 {
		maxDemos = 3
	}

	for i := 0; i < numCandidates; i++ {
		numExamples := m.rng.Intn(maxDemos) + 1
		var candidate []Example

		for j := 0; j < numExamples && len(pool) > 0; j++ {
			idx := m.rng.Intn(len(pool))
			candidate = append(candidate, pool[idx])
		}

		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

// proposeInstructionCandidates implements Step 2 of MIPROv2
func (m *MIPROv2) proposeInstructionCandidates(basePrompt, modelName string, fewShotCandidates [][]Example) ([]string, error) {
	target := m.config.NumInstructionCandidates
	if target <= 0 {
		target = 4
	}

	candidates := make([]string, 0, target)

	if m.modelProvider != nil {
		promptBuilder := strings.Builder{}
		promptBuilder.WriteString("You are helping optimize prompt templates. ")
		promptBuilder.WriteString("Given the base task description below, produce short instruction candidates (one per line).\n\n")
		promptBuilder.WriteString("Base task:\n")
		promptBuilder.WriteString(basePrompt)
		promptBuilder.WriteString("\n\nFormat each instruction as a standalone sentence.")

		response, err := m.modelProvider.CallModel(modelName, promptBuilder.String())
		if err == nil {
			candidates = append(candidates, parseInstructionLines(response)...)
		}
	}

	fallback := []string{
		"Be concise and focus on key points",
		"Use technical language appropriate for experts",
		"Provide examples to illustrate concepts",
		"Structure the response with clear headings",
		"Begin with a brief summary before diving into details",
		"Use analogies to explain complex topics",
		"Address potential counterarguments",
		"Include relevant statistics and data",
	}

	for _, inst := range fallback {
		candidates = append(candidates, inst)
	}

	cleaned := dedupeAndTrimInstructions(candidates)
	if len(cleaned) == 0 {
		cleaned = []string{"Provide a clear, structured response."}
	}

	if len(cleaned) > target {
		cleaned = cleaned[:target]
	}

	return cleaned, nil
}

// optimizeCombination implements Step 3 of MIPROv2 using Bayesian optimization
func (m *MIPROv2) optimizeCombination(basePrompt string, model models.Model, fewShotCandidates [][]Example, instructionCandidates []string, evalExamples []Example) (*OptimizationResult, error) {
	if len(instructionCandidates) == 0 {
		instructionCandidates = []string{""}
	}
	if len(fewShotCandidates) == 0 {
		fewShotCandidates = [][]Example{{}}
	}

	buildPrompt := func(instructionIdx, exampleIdx int) Prompt {
		if instructionIdx < 0 || instructionIdx >= len(instructionCandidates) {
			instructionIdx = 0
		}
		if exampleIdx < 0 || exampleIdx >= len(fewShotCandidates) {
			exampleIdx = 0
		}

		return Prompt{
			Instruction: instructionCandidates[instructionIdx],
			Examples:    fewShotCandidates[exampleIdx],
			BasePrompt:  basePrompt,
		}
	}

	evaluate := func(instructionIdx, exampleIdx int) (float64, []map[string]interface{}, error) {
		prompt := buildPrompt(instructionIdx, exampleIdx)
		return m.evaluatePromptCandidate(prompt, model, evalExamples)
	}

	if m.bayesianOpt != nil && (len(instructionCandidates) > 1 || len(fewShotCandidates) > 1) {
		searchSpace := make(map[string]interface{})
		if len(instructionCandidates) > 1 {
			searchSpace["instruction_idx"] = map[string]interface{}{
				"type": "int",
				"low":  0,
				"high": len(instructionCandidates) - 1,
			}
		}
		if len(fewShotCandidates) > 1 {
			searchSpace["example_idx"] = map[string]interface{}{
				"type": "int",
				"low":  0,
				"high": len(fewShotCandidates) - 1,
			}
		}

		bestParams, _, err := m.bayesianOpt.Optimize(searchSpace, func(params map[string]interface{}) (float64, error) {
			score, _, evalErr := evaluate(getIndexParam(params["instruction_idx"]), getIndexParam(params["example_idx"]))
			return score, evalErr
		})
		if err == nil {
			instructionIdx := getIndexParam(bestParams["instruction_idx"])
			exampleIdx := getIndexParam(bestParams["example_idx"])
			score, evalDetails, evalErr := evaluate(instructionIdx, exampleIdx)
			if evalErr != nil {
				return nil, evalErr
			}
			return &OptimizationResult{
				BestPrompt:        buildPrompt(instructionIdx, exampleIdx),
				Score:             score,
				EvaluationDetails: evalDetails,
			}, nil
		}
	}

	// Fallback to brute-force search if Bayesian optimization is unavailable or fails
	bestScore := -1.0
	var bestPrompt Prompt
	var evalDetails []map[string]interface{}

	for instIdx := range instructionCandidates {
		for exIdx := range fewShotCandidates {
			score, details, err := evaluate(instIdx, exIdx)
			if err != nil {
				continue
			}
			if score > bestScore {
				bestScore = score
				bestPrompt = buildPrompt(instIdx, exIdx)
				evalDetails = details
			}
		}
	}

	if bestScore < 0 {
		// As a last resort, return the base prompt without modifications
		base := buildPrompt(0, 0)
		score, details, _ := evaluate(0, 0)
		return &OptimizationResult{
			BestPrompt:        base,
			Score:             score,
			EvaluationDetails: details,
		}, nil
	}

	return &OptimizationResult{
		BestPrompt:        bestPrompt,
		Score:             bestScore,
		EvaluationDetails: evalDetails,
	}, nil
}

// evaluatePrompt evaluates the quality of a prompt (simplified implementation)
func (m *MIPROv2) evaluatePrompt(prompt Prompt, modelName string) float64 {
	// In a real implementation, this would:
	// 1. Run the prompt on a validation set
	// 2. Compare outputs to ground truth
	// 3. Return a quality score

	// For this implementation, we'll return a random score weighted by some factors
	factor := 1.0

	// More examples might be better (up to a point)
	if len(prompt.Examples) > 0 {
		factor += 0.1 * float64(len(prompt.Examples))
	}

	// Longer instructions might be better (up to a point)
	if len(prompt.Instruction) > 20 {
		factor += 0.05
	}

	// Generate a base score between 0.7 and 0.9, then apply factors
	baseScore := 0.7 + m.rng.Float64()*0.2
	score := baseScore * factor

	// Ensure score is between 0 and 1
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// evaluatePromptCandidate replays evaluation examples through the provider/evaluator when available.
func (m *MIPROv2) evaluatePromptCandidate(prompt Prompt, model models.Model, evalExamples []Example) (float64, []map[string]interface{}, error) {
	if len(evalExamples) == 0 || m.modelProvider == nil || m.evaluator == nil {
		return m.evaluatePrompt(prompt, model.Name), nil, nil
	}

	limit := m.config.MinibatchSize
	if limit <= 0 || limit > len(evalExamples) {
		limit = len(evalExamples)
	}

	if limit == 0 {
		return m.evaluatePrompt(prompt, model.Name), nil, nil
	}

	totalScore := 0.0
	summaries := make([]map[string]interface{}, 0, limit)

	for i := 0; i < limit; i++ {
		example := evalExamples[i]
		rendered := m.renderPromptForExample(prompt, example.Input)

		actualOutput, err := m.modelProvider.CallModel(model.Name, rendered)
		if err != nil {
			return 0, nil, fmt.Errorf("model call during evaluation failed: %w", err)
		}

		evalResult, err := m.evaluator.Evaluate(rendered, example.Output, actualOutput, model)
		if err != nil {
			return 0, nil, fmt.Errorf("evaluation failed: %w", err)
		}

		totalScore += evalResult.Score
		summaries = append(summaries, map[string]interface{}{
			"input":   example.Input,
			"score":   evalResult.Score,
			"correct": evalResult.Correct,
		})
	}

	return totalScore / float64(limit), summaries, nil
}

// OptimizePrompt implements the ExtendedPromptOptimizer interface
func (m *MIPROv2) OptimizePrompt(basePrompt string, model models.Model, examples []models.Example) (*PromptOptimizationResult, error) {
	// Convert models.Example to our internal Example type
	internalExamples := make([]Example, len(examples))
	for i, ex := range examples {
		internalExamples[i] = Example{
			Input:  ex.Input,
			Output: ex.Output,
		}
	}

	// Step 1: Bootstrap Few-Shot Examples
	fmt.Println("Step 1: Bootstrapping few-shot examples...")
	fewShotCandidates, err := m.bootstrapFewShotExamples(basePrompt, model.Name, internalExamples)
	if err != nil {
		return nil, fmt.Errorf("failed to bootstrap few-shot examples: %w", err)
	}

	// Add the provided examples to the candidates
	if len(internalExamples) > 0 {
		fewShotCandidates = append(fewShotCandidates, internalExamples)
	}

	// Step 2: Propose Instruction Candidates
	fmt.Println("Step 2: Proposing instruction candidates...")
	instructionCandidates, err := m.proposeInstructionCandidates(basePrompt, model.Name, fewShotCandidates)
	if err != nil {
		return nil, fmt.Errorf("failed to propose instruction candidates: %w", err)
	}

	// Step 3: Find an Optimized Combination using Bayesian Optimization
	fmt.Println("Step 3: Finding optimized combination...")
	result, err := m.optimizeCombination(basePrompt, model, fewShotCandidates, instructionCandidates, internalExamples)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize combination: %w", err)
	}

	// Construct the final optimized prompt
	optimizedPrompt := m.constructPrompt(result.BestPrompt)

	metadata := map[string]interface{}{
		"model":        model.Name,
		"instruction":  result.BestPrompt.Instruction,
		"num_examples": len(result.BestPrompt.Examples),
	}

	if len(result.EvaluationDetails) > 0 {
		evalMeta := map[string]interface{}{
			"examples": result.EvaluationDetails,
		}
		if m.evaluator != nil {
			evalMeta["strategy"] = m.evaluator.Name()
		}
		metadata["evaluation"] = evalMeta
	}

	return &PromptOptimizationResult{
		Prompt:   optimizedPrompt,
		Score:    result.Score,
		Metadata: metadata,
	}, nil
}

// EvaluatePrompt implements the ExtendedPromptOptimizer interface
func (m *MIPROv2) EvaluatePrompt(prompt string, model models.Model, examples []models.Example) (float64, error) {
	internalExamples := toInternalExamples(examples)
	candidate := Prompt{BasePrompt: prompt}
	score, _, err := m.evaluatePromptCandidate(candidate, model, internalExamples)
	return score, err
}

// GetBestCandidate implements the ExtendedPromptOptimizer interface
func (m *MIPROv2) GetBestCandidate() *PromptOptimizationResult {
	// This would typically return the best candidate found during optimization
	// For this implementation, we return nil since we don't maintain state between calls
	return nil
}

// constructPrompt builds a final prompt string from a Prompt struct
func (m *MIPROv2) constructPrompt(prompt Prompt) string {
	result := ""

	// Add instruction
	if prompt.Instruction != "" {
		result += fmt.Sprintf("Instruction: %s\n\n", prompt.Instruction)
	}

	// Add examples
	if len(prompt.Examples) > 0 {
		result += "Examples:\n"
		for i, example := range prompt.Examples {
			result += fmt.Sprintf("Example %d:\nInput: %s\nOutput: %s\n\n", i+1, example.Input, example.Output)
		}
	}

	// Add base prompt
	result += fmt.Sprintf("Task: %s", prompt.BasePrompt)

	return result
}

func (m *MIPROv2) renderPromptForExample(prompt Prompt, userInput string) string {
	base := m.constructPrompt(prompt)
	if strings.TrimSpace(userInput) == "" {
		return base
	}
	return fmt.Sprintf("%s\n\nUser Request: %s", base, userInput)
}

// Evaluate assesses the quality of a prompt for a given model
func (m *MIPROv2) Evaluate(prompt string, modelName string) (float64, error) {
	// Create a simple prompt structure for evaluation
	p := Prompt{
		BasePrompt: prompt,
	}

	// Evaluate the prompt
	score := m.evaluatePrompt(p, modelName)
	return score, nil
}

func toInternalExamples(examples []models.Example) []Example {
	if len(examples) == 0 {
		return nil
	}
	out := make([]Example, 0, len(examples))
	for _, ex := range examples {
		if ex.Input == "" || ex.Output == "" {
			continue
		}
		out = append(out, Example{
			Input:  ex.Input,
			Output: ex.Output,
		})
	}
	return out
}

func parseInstructionLines(text string) []string {
	lines := strings.Split(text, "\n")
	var out []string
	for _, line := range lines {
		clean := strings.TrimSpace(line)
		clean = strings.TrimLeft(clean, "-•0123456789. ")
		clean = strings.TrimSpace(clean)
		if clean != "" {
			out = append(out, clean)
		}
	}
	return out
}

func dedupeAndTrimInstructions(instructions []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, inst := range instructions {
		clean := strings.TrimSpace(inst)
		if clean == "" {
			continue
		}
		key := strings.ToLower(clean)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, clean)
	}
	return out
}

func getIndexParam(value interface{}) int {
	switch v := value.(type) {
	case int:
		if v >= 0 {
			return v
		}
	case float64:
		if v >= 0 {
			return int(v)
		}
	}
	return 0
}
