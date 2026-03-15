package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Manifest describes a registered prompt template.
type Manifest struct {
	ID              string                 `json:"id" yaml:"id"`
	Name            string                 `json:"name" yaml:"name"`
	Prompt          string                 `json:"prompt" yaml:"prompt"`
	Variables       []string               `json:"variables" yaml:"variables"`
	DefaultModel    string                 `json:"default_model" yaml:"default_model"`
	DefaultProvider string                 `json:"default_provider" yaml:"default_provider"`
	Metadata        map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at" yaml:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" yaml:"updated_at"`
}

// Manager persists template manifests on disk.
type Manager struct {
	basePath string
}

// NewManager creates a manager rooted at basePath.
func NewManager(basePath string) (*Manager, error) {
	if basePath == "" {
		return nil, fmt.Errorf("base path is required")
	}
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, fmt.Errorf("create templates directory: %w", err)
	}
	return &Manager{basePath: basePath}, nil
}

// Register writes a manifest for the template ID.
func (m *Manager) Register(manifest *Manifest) (string, error) {
	if manifest == nil {
		return "", fmt.Errorf("manifest is nil")
	}
	if manifest.ID == "" {
		return "", fmt.Errorf("template id is required")
	}
	if manifest.Prompt == "" {
		return "", fmt.Errorf("prompt text is required")
	}
	manifest.CreatedAt = time.Now()
	manifest.UpdatedAt = manifest.CreatedAt

	dir := filepath.Join(m.basePath, manifest.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create template dir: %w", err)
	}
	path := filepath.Join(dir, "manifest.yaml")
	if err := writeYAML(path, manifest); err != nil {
		return "", err
	}
	return path, nil
}

// Load reads an existing manifest by template ID.
func (m *Manager) Load(id string) (*Manifest, error) {
	if id == "" {
		return nil, fmt.Errorf("template id is required")
	}
	path := filepath.Join(m.basePath, id, "manifest.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &manifest, nil
}

// List returns all manifests stored under the base path.
func (m *Manager) List() ([]*Manifest, error) {
	entries, err := os.ReadDir(m.basePath)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	var manifests []*Manifest
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifest, err := m.Load(entry.Name())
		if err != nil {
			continue
		}
		manifests = append(manifests, manifest)
	}
	return manifests, nil
}

func writeYAML(path string, v interface{}) error {
	data, err := yamlMarshal(v)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

func yamlMarshal(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}

// ParseVariables converts a comma-separated string into a slice.
func ParseVariables(csv string) []string {
	if csv == "" {
		return nil
	}
	parts := strings.Split(csv, ",")
	var out []string
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
