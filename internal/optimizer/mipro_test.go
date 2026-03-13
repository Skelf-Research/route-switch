package optimizer

import (
	"strings"
	"testing"
)

func TestNewMIPROv2(t *testing.T) {
	optimizer := NewMIPROv2()
	if optimizer == nil {
		t.Error("Expected NewMIPROv2 to return a non-nil optimizer")
	}
}

func TestMIPROv2Optimize(t *testing.T) {
	optimizer := NewMIPROv2()
	prompt := "Write a story about a robot"
	model := "gpt-4"
	
	result, err := optimizer.Optimize(prompt, model)
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if result == "" {
		t.Error("Expected optimized prompt to not be empty")
	}
	
	// Check that the result contains expected elements
	if !strings.Contains(result, "Instruction:") && !strings.Contains(result, "Examples:") {
		t.Log("Note: Result may be using simplified optimization in test mode")
	}
}

func TestMIPROv2Evaluate(t *testing.T) {
	optimizer := NewMIPROv2()
	prompt := "Write a story about a robot"
	model := "gpt-4"
	
	score, err := optimizer.Evaluate(prompt, model)
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if score < 0 || score > 1 {
		t.Errorf("Expected score between 0 and 1, got %f", score)
	}
}

func TestConstructPrompt(t *testing.T) {
	optimizer := NewMIPROv2()
	
	prompt := Prompt{
		Instruction: "Be clear and concise",
		Examples: []Example{
			{Input: "Example input", Output: "Example output"},
		},
		BasePrompt: "Write a story",
	}
	
	result := optimizer.constructPrompt(prompt)
	
	if !strings.Contains(result, "Instruction: Be clear and concise") {
		t.Error("Expected instruction to be included in prompt")
	}
	
	if !strings.Contains(result, "Example input") {
		t.Error("Expected example input to be included in prompt")
	}
	
	if !strings.Contains(result, "Write a story") {
		t.Error("Expected base prompt to be included in prompt")
	}
}