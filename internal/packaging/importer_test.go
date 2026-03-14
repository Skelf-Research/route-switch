package packaging

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/skelf-research/route-switch/internal/config"
	"github.com/skelf-research/route-switch/internal/templates"
	"gopkg.in/yaml.v3"
)

func TestImporterImport(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	pkgDir := filepath.Join(root, "package")
	if err := os.MkdirAll(filepath.Join(pkgDir, "dataset"), 0o755); err != nil {
		t.Fatalf("mkdir dataset: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(pkgDir, "logs"), 0o755); err != nil {
		t.Fatalf("mkdir logs: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(pkgDir, "analytics"), 0o755); err != nil {
		t.Fatalf("mkdir analytics: %v", err)
	}

	templateManifest := templates.Manifest{
		ID:           "support-flow",
		Name:         "Support Flow",
		Prompt:       "Hello {name}",
		DefaultModel: "gpt-4",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	writeYAMLHelper(t, filepath.Join(pkgDir, "manifest.yaml"), templateManifest)

	pkgManifest := PackageManifest{
		TemplateID:    "support-flow",
		ExportedAt:    time.Now(),
		DatasetFile:   "dataset/support-flow.db",
		LogsFile:      "logs/recent.jsonl",
		AnalyticsFile: "analytics/metrics.duckdb",
		Version:       "1.0",
	}
	writeYAMLHelper(t, filepath.Join(pkgDir, "package.yaml"), pkgManifest)

	if err := os.WriteFile(filepath.Join(pkgDir, pkgManifest.DatasetFile), []byte("db"), 0o644); err != nil {
		t.Fatalf("write dataset: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, pkgManifest.LogsFile), []byte("log"), 0o644); err != nil {
		t.Fatalf("write logs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, pkgManifest.AnalyticsFile), []byte("duckdb"), 0o644); err != nil {
		t.Fatalf("write analytics: %v", err)
	}

	targetDataset := filepath.Join(root, "datasets")
	targetAnalytics := filepath.Join(root, "analytics", "metrics.duckdb")
	cfg := &config.Config{
		Dataset: config.DatasetConfig{
			BasePath:   targetDataset,
			MaxRecords: 100,
		},
		Analytics: config.AnalyticsConfig{
			Driver: "duckdb",
			Path:   targetAnalytics,
		},
	}

	importer := NewImporter(cfg)
	result, err := importer.Import(ctx, ImportOptions{
		PackagePath:      pkgDir,
		RestoreAnalytics: true,
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if result.TemplateID != "support-flow" {
		t.Fatalf("unexpected template id: %s", result.TemplateID)
	}
	if _, err := os.Stat(result.ManifestPath); err != nil {
		t.Fatalf("manifest not copied: %v", err)
	}
	if _, err := os.Stat(result.DatasetPath); err != nil {
		t.Fatalf("dataset not copied: %v", err)
	}
	if _, err := os.Stat(result.LogsPath); err != nil {
		t.Fatalf("logs not copied: %v", err)
	}
	if _, err := os.Stat(result.AnalyticsPath); err != nil {
		t.Fatalf("analytics not copied: %v", err)
	}
}

func TestImporterImportRequiresOverwrite(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	pkgDir := filepath.Join(root, "pkg2")
	if err := os.MkdirAll(filepath.Join(pkgDir, "dataset"), 0o755); err != nil {
		t.Fatalf("mkdir dataset: %v", err)
	}

	templateManifest := templates.Manifest{
		ID:         "support-flow",
		Prompt:     "Hi",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	writeYAMLHelper(t, filepath.Join(pkgDir, "manifest.yaml"), templateManifest)

	pkgManifest := PackageManifest{
		TemplateID:  "support-flow",
		ExportedAt:  time.Now(),
		DatasetFile: "dataset/support-flow.db",
		Version:     "1.0",
	}
	writeYAMLHelper(t, filepath.Join(pkgDir, "package.yaml"), pkgManifest)
	if err := os.WriteFile(filepath.Join(pkgDir, pkgManifest.DatasetFile), []byte("db"), 0o644); err != nil {
		t.Fatalf("write dataset: %v", err)
	}

	targetDataset := filepath.Join(root, "datasets")
	if err := os.MkdirAll(targetDataset, 0o755); err != nil {
		t.Fatalf("mkdir target dataset: %v", err)
	}
	existing := filepath.Join(targetDataset, "support-flow.db")
	if err := os.WriteFile(existing, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write existing dataset: %v", err)
	}

	cfg := &config.Config{
		Dataset: config.DatasetConfig{
			BasePath:   targetDataset,
			MaxRecords: 100,
		},
	}
	importer := NewImporter(cfg)
	_, err := importer.Import(ctx, ImportOptions{
		PackagePath: pkgDir,
	})
	if err == nil {
		t.Fatalf("expected error without overwrite")
	}
}

func writeYAMLHelper(t *testing.T, path string, v interface{}) {
	t.Helper()
	data, err := yaml.Marshal(v)
	if err != nil {
		t.Fatalf("marshal yaml: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for yaml: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
}
