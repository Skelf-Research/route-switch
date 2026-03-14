package templates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManager_Register(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}

	manifest := &Manifest{
		ID:           "test-template",
		Name:         "Test Template",
		Prompt:       "Hello {name}",
		Variables:    []string{"name"},
		DefaultModel: "gpt-4",
	}

	path, err := manager.Register(manifest)
	if err != nil {
		t.Fatalf("Register error: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected manifest file at %s: %v", path, err)
	}

	expectedDir := filepath.Join(dir, "test-template")
	if filepath.Dir(path) != expectedDir {
		t.Fatalf("expected manifest dir %s got %s", expectedDir, filepath.Dir(path))
	}
}

func TestManager_Load(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}

	manifest := &Manifest{
		ID:           "template-123",
		Name:         "Template 123",
		Prompt:       "Test prompt",
		DefaultModel: "gpt-4",
	}
	if _, err := manager.Register(manifest); err != nil {
		t.Fatalf("Register error: %v", err)
	}

	loaded, err := manager.Load("template-123")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if loaded.Prompt != "Test prompt" {
		t.Fatalf("expected prompt %q got %q", "Test prompt", loaded.Prompt)
	}
}

func TestParseVariables(t *testing.T) {
	vars := ParseVariables("name, topic , ,priority")
	if len(vars) != 3 {
		t.Fatalf("expected 3 variables, got %d", len(vars))
	}
	if vars[0] != "name" || vars[1] != "topic" || vars[2] != "priority" {
		t.Fatalf("unexpected variables: %#v", vars)
	}

	if result := ParseVariables(""); result != nil {
		t.Fatalf("expected nil result for empty input")
	}
}
