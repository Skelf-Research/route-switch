package packaging

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/skelf-research/route-switch/internal/config"
	"github.com/skelf-research/route-switch/internal/storage/dataset"
	"github.com/skelf-research/route-switch/internal/templates"
)

func TestExporterExport(t *testing.T) {
	baseDir := t.TempDir()
	cfg := &config.Config{
		Dataset: config.DatasetConfig{
			BasePath:   filepath.Join(baseDir, "datasets"),
			MaxRecords: 100,
		},
		Analytics: config.AnalyticsConfig{
			Driver: "duckdb",
			Path:   filepath.Join(baseDir, "analytics", "metrics.duckdb"),
		},
	}

	manager, err := templates.NewManager(cfg.Dataset.BasePath)
	if err != nil {
		t.Fatalf("templates.NewManager: %v", err)
	}
	_, err = manager.Register(&templates.Manifest{
		ID:           "support-flow",
		Name:         "Support Flow",
		Prompt:       "Hello {name}",
		DefaultModel: "gpt-4",
	})
	if err != nil {
		t.Fatalf("register manifest: %v", err)
	}

	store, err := dataset.NewSQLiteStore(cfg.Dataset.BasePath, cfg.Dataset.MaxRecords)
	if err != nil {
		t.Fatalf("dataset.NewSQLiteStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	record := &dataset.Record{
		PromptID:  "support-flow",
		Model:     "gpt-4",
		Input:     "Hello Jordan",
		Output:    "Hi Jordan",
		Success:   true,
		CreatedAt: time.Now(),
	}
	if err := store.AddRecord(ctx, "support-flow", record); err != nil {
		t.Fatalf("AddRecord: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(cfg.Analytics.Path), 0o755); err != nil {
		t.Fatalf("mkdir analytics: %v", err)
	}
	if err := os.WriteFile(cfg.Analytics.Path, []byte("duckdb"), 0o644); err != nil {
		t.Fatalf("write analytics file: %v", err)
	}

	exporter, err := NewExporter(cfg)
	if err != nil {
		t.Fatalf("NewExporter: %v", err)
	}

	info, err := exporter.Export(ctx, ExportOptions{
		TemplateID:       "support-flow",
		OutputDir:        filepath.Join(baseDir, "packages"),
		IncludeAnalytics: true,
		LogsLimit:        10,
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if _, err := os.Stat(info.OutputPath); err != nil {
		t.Fatalf("package path missing: %v", err)
	}
	if info.DatasetFile == "" || info.LogsFile == "" {
		t.Fatalf("expected dataset/logs paths")
	}

	if _, err := os.Stat(filepath.Join(info.OutputPath, "package.yaml")); err != nil {
		t.Fatalf("package manifest missing: %v", err)
	}

	if _, err := os.Stat(filepath.Join(info.OutputPath, "manifest.yaml")); err != nil {
		t.Fatalf("template manifest missing: %v", err)
	}
}
