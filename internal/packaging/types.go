package packaging

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// PackageManifest describes the portable package contents.
type PackageManifest struct {
	TemplateID     string    `json:"template_id" yaml:"template_id"`
	ExportedAt     time.Time `json:"exported_at" yaml:"exported_at"`
	DatasetFile    string    `json:"dataset_file" yaml:"dataset_file"`
	LogsFile       string    `json:"logs_file" yaml:"logs_file"`
	AnalyticsFile  string    `json:"analytics_file,omitempty" yaml:"analytics_file,omitempty"`
	GitInitialized bool      `json:"git_initialized" yaml:"git_initialized"`
	Version        string    `json:"version" yaml:"version"`
}

func loadPackageManifest(path string) (*PackageManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read package manifest: %w", err)
	}
	var manifest PackageManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse package manifest: %w", err)
	}
	if manifest.TemplateID == "" {
		return nil, fmt.Errorf("package manifest missing template_id")
	}
	return &manifest, nil
}
