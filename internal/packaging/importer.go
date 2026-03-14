package packaging

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/skelf-research/route-switch/internal/config"
	"github.com/skelf-research/route-switch/internal/templates"
	"gopkg.in/yaml.v3"
)

// ImportOptions controls how package import behaves.
type ImportOptions struct {
	PackagePath      string
	RestoreAnalytics bool
	Overwrite        bool
}

// ImportResult summarizes imported artifacts.
type ImportResult struct {
	TemplateID    string
	ManifestPath  string
	DatasetPath   string
	LogsPath      string
	AnalyticsPath string
}

// Importer hydrates prompt packages onto the local filesystem.
type Importer struct {
	cfg *config.Config
}

// NewImporter returns an importer for a configuration.
func NewImporter(cfg *config.Config) *Importer {
	return &Importer{cfg: cfg}
}

// Import copies the manifest, dataset, logs, and optional analytics snapshot into place.
func (i *Importer) Import(ctx context.Context, opts ImportOptions) (*ImportResult, error) {
	_ = ctx
	if opts.PackagePath == "" {
		return nil, fmt.Errorf("package path is required")
	}
	info, err := os.Stat(opts.PackagePath)
	if err != nil {
		return nil, fmt.Errorf("stat package path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("package path must be a directory")
	}

	pkgManifest, err := loadPackageManifest(filepath.Join(opts.PackagePath, "package.yaml"))
	if err != nil {
		return nil, err
	}

	templateManifest, err := readTemplateManifest(filepath.Join(opts.PackagePath, "manifest.yaml"))
	if err != nil {
		return nil, err
	}

	templateDir := filepath.Join(i.cfg.Dataset.BasePath, templateManifest.ID)
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		return nil, fmt.Errorf("create template directory: %w", err)
	}

	destManifest := filepath.Join(templateDir, "manifest.yaml")
	if err := copyWithOverwrite(filepath.Join(opts.PackagePath, "manifest.yaml"), destManifest, opts.Overwrite); err != nil {
		return nil, err
	}

	datasetSrc := filepath.Join(opts.PackagePath, pkgManifest.DatasetFile)
	destDataset := filepath.Join(i.cfg.Dataset.BasePath, fmt.Sprintf("%s.db", templateManifest.ID))
	if err := copyWithOverwrite(datasetSrc, destDataset, opts.Overwrite); err != nil {
		return nil, err
	}

	var logsDest string
	if pkgManifest.LogsFile != "" {
		logsDir := filepath.Join(templateDir, "logs")
		if err := os.MkdirAll(logsDir, 0o755); err != nil {
			return nil, fmt.Errorf("create logs dir: %w", err)
		}
		logsDest = filepath.Join(logsDir, filepath.Base(pkgManifest.LogsFile))
		if err := copyWithOverwrite(filepath.Join(opts.PackagePath, pkgManifest.LogsFile), logsDest, opts.Overwrite); err != nil {
			return nil, err
		}
	}

	var analyticsDest string
	if opts.RestoreAnalytics && pkgManifest.AnalyticsFile != "" {
		if i.cfg.Analytics.Path == "" {
			return nil, fmt.Errorf("analytics path not configured in config")
		}
		if err := os.MkdirAll(filepath.Dir(i.cfg.Analytics.Path), 0o755); err != nil {
			return nil, fmt.Errorf("create analytics dir: %w", err)
		}
		if err := copyWithOverwrite(filepath.Join(opts.PackagePath, pkgManifest.AnalyticsFile), i.cfg.Analytics.Path, opts.Overwrite); err != nil {
			return nil, err
		}
		analyticsDest = i.cfg.Analytics.Path
	}

	return &ImportResult{
		TemplateID:    templateManifest.ID,
		ManifestPath:  destManifest,
		DatasetPath:   destDataset,
		LogsPath:      logsDest,
		AnalyticsPath: analyticsDest,
	}, nil
}

func readTemplateManifest(path string) (*templates.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read template manifest: %w", err)
	}
	var manifest templates.Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse template manifest: %w", err)
	}
	if manifest.ID == "" {
		return nil, fmt.Errorf("template manifest missing id")
	}
	return &manifest, nil
}

func copyWithOverwrite(src, dest string, overwrite bool) error {
	if !overwrite {
		if _, err := os.Stat(dest); err == nil {
			return fmt.Errorf("%s already exists (use --overwrite to replace)", dest)
		}
	}
	return copyFile(src, dest)
}
