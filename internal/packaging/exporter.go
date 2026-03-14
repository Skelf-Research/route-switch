package packaging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/skelf-research/route-switch/internal/config"
	"github.com/skelf-research/route-switch/internal/storage/dataset"
	"github.com/skelf-research/route-switch/internal/templates"
	"gopkg.in/yaml.v3"
)

// ExportOptions control how a prompt package is produced.
type ExportOptions struct {
	TemplateID       string
	OutputDir        string
	IncludeAnalytics bool
	LogsLimit        int
}

// PackageInfo captures the export result.
type PackageInfo struct {
	TemplateID     string
	OutputPath     string
	DatasetFile    string
	LogsFile       string
	AnalyticsFile  string
	GitInitialized bool
}

// Exporter handles package creation from config + template manager.
type Exporter struct {
	cfg       *config.Config
	templates *templates.Manager
}

// NewExporter builds an exporter backed by the provided configuration.
func NewExporter(cfg *config.Config) (*Exporter, error) {
	manager, err := templates.NewManager(cfg.Dataset.BasePath)
	if err != nil {
		return nil, err
	}
	return &Exporter{
		cfg:       cfg,
		templates: manager,
	}, nil
}

// Export writes a package for templateID and returns metadata about the result.
func (e *Exporter) Export(ctx context.Context, opts ExportOptions) (*PackageInfo, error) {
	if opts.TemplateID == "" {
		return nil, fmt.Errorf("template id is required")
	}
	if opts.LogsLimit <= 0 {
		opts.LogsLimit = 50
	}
	exportedAt := time.Now().UTC()

	manifest, err := e.templates.Load(opts.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("load template manifest: %w", err)
	}

	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = filepath.Join("packages", opts.TemplateID)
	}
	baseDir := filepath.Join(outputDir, fmt.Sprintf("%s-%s", opts.TemplateID, exportedAt.Format("20060102-150405")))
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create package directory: %w", err)
	}

	if err := e.writeTemplateManifest(baseDir, manifest); err != nil {
		return nil, err
	}

	datasetFile, err := e.copyDataset(opts.TemplateID, baseDir)
	if err != nil {
		return nil, err
	}

	logsFile, err := e.writeRecentLogs(ctx, opts.TemplateID, opts.LogsLimit, baseDir)
	if err != nil {
		return nil, err
	}

	var analyticsRelative string
	if opts.IncludeAnalytics {
		if file, err := e.copyAnalyticsSnapshot(baseDir); err != nil {
			return nil, err
		} else {
			analyticsRelative = file
		}
	}

	gitInitialized, err := initializeGitRepo(baseDir)
	if err != nil {
		return nil, err
	}

	if err := e.writePackageManifest(baseDir, &PackageManifest{
		TemplateID:     manifest.ID,
		ExportedAt:     exportedAt,
		DatasetFile:    datasetFile,
		LogsFile:       logsFile,
		AnalyticsFile:  analyticsRelative,
		GitInitialized: gitInitialized,
		Version:        "1.0",
	}); err != nil {
		return nil, err
	}

	var analyticsAbs string
	if analyticsRelative != "" {
		analyticsAbs = filepath.Join(baseDir, analyticsRelative)
	}

	return &PackageInfo{
		TemplateID:     manifest.ID,
		OutputPath:     baseDir,
		DatasetFile:    filepath.Join(baseDir, datasetFile),
		LogsFile:       filepath.Join(baseDir, logsFile),
		AnalyticsFile:  analyticsAbs,
		GitInitialized: gitInitialized,
	}, nil
}

func (e *Exporter) writeTemplateManifest(baseDir string, manifest *templates.Manifest) error {
	path := filepath.Join(baseDir, "manifest.yaml")
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

func (e *Exporter) copyDataset(templateID, baseDir string) (string, error) {
	src := filepath.Join(e.cfg.Dataset.BasePath, fmt.Sprintf("%s.db", templateID))
	if _, err := os.Stat(src); err != nil {
		return "", fmt.Errorf("dataset missing for %s: %w", templateID, err)
	}
	destDir := filepath.Join(baseDir, "dataset")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("create dataset folder: %w", err)
	}
	dest := filepath.Join(destDir, fmt.Sprintf("%s.db", templateID))
	if err := copyFile(src, dest); err != nil {
		return "", err
	}
	relative := filepath.Join("dataset", fmt.Sprintf("%s.db", templateID))
	return relative, nil
}

func (e *Exporter) writeRecentLogs(ctx context.Context, templateID string, limit int, baseDir string) (string, error) {
	store, err := dataset.NewSQLiteStore(e.cfg.Dataset.BasePath, e.cfg.Dataset.MaxRecords)
	if err != nil {
		return "", fmt.Errorf("init dataset store: %w", err)
	}
	defer store.Close()

	records, err := store.ListRecent(ctx, templateID, limit)
	if err != nil {
		return "", fmt.Errorf("list recent dataset entries: %w", err)
	}

	logsDir := filepath.Join(baseDir, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return "", fmt.Errorf("create logs dir: %w", err)
	}

	path := filepath.Join(logsDir, "recent.jsonl")
	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create logs file: %w", err)
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	for _, record := range records {
		entry := map[string]interface{}{
			"id":         record.ID,
			"model":      record.Model,
			"input":      record.Input,
			"output":     record.Output,
			"success":    record.Success,
			"cost":       record.Cost,
			"metadata":   record.Metadata,
			"variables":  record.Variables,
			"created_at": record.CreatedAt,
		}
		if err := enc.Encode(entry); err != nil {
			return "", fmt.Errorf("write log entry: %w", err)
		}
	}

	return filepath.Join("logs", "recent.jsonl"), nil
}

func (e *Exporter) copyAnalyticsSnapshot(baseDir string) (string, error) {
	driver := strings.ToLower(e.cfg.Analytics.Driver)
	if driver == "" {
		driver = "duckdb"
	}
	if driver != "duckdb" {
		return "", fmt.Errorf("analytics driver %s not supported for packaging", e.cfg.Analytics.Driver)
	}
	if e.cfg.Analytics.Path == "" {
		return "", fmt.Errorf("analytics path not configured")
	}
	if _, err := os.Stat(e.cfg.Analytics.Path); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("stat analytics db: %w", err)
	}
	destDir := filepath.Join(baseDir, "analytics")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("create analytics dir: %w", err)
	}
	dest := filepath.Join(destDir, filepath.Base(e.cfg.Analytics.Path))
	if err := copyFile(e.cfg.Analytics.Path, dest); err != nil {
		return "", err
	}
	return filepath.Join("analytics", filepath.Base(e.cfg.Analytics.Path)), nil
}

func (e *Exporter) writePackageManifest(baseDir string, manifest *PackageManifest) error {
	path := filepath.Join(baseDir, "package.yaml")
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal package manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write package manifest: %w", err)
	}
	return nil
}

func copyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("create copy dir: %w", err)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}
	return nil
}

func initializeGitRepo(path string) (bool, error) {
	cmd := exec.Command("git", "init")
	cmd.Dir = path
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("git init: %w", err)
	}
	return true, nil
}
