package cli

import (
	"context"
	"fmt"

	"github.com/skelf-research/route-switch/internal/config"
	"github.com/skelf-research/route-switch/internal/packaging"
	"github.com/spf13/cobra"
)

var (
	packageTemplateID             string
	packageOutputDir              string
	packageIncludeAnalytics       = true
	packageLogsLimit              int
	packageImportPath             string
	packageImportRestoreAnalytics = true
	packageImportOverwrite        bool
)

var packageCmd = &cobra.Command{
	Use:   "package",
	Short: "Manage portable prompt packages",
}

var packageExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export a prompt template + dataset bundle",
	RunE: func(cmd *cobra.Command, args []string) error {
		if packageTemplateID == "" {
			return fmt.Errorf("--template-id is required")
		}

		cfgManager := config.NewSimpleConfigManager()
		if configFile != "" {
			if err := cfgManager.Load(configFile); err != nil {
				return fmt.Errorf("load config: %w", err)
			}
		}

		exporter, err := packaging.NewExporter(cfgManager.GetConfig())
		if err != nil {
			return err
		}

		info, err := exporter.Export(context.Background(), packaging.ExportOptions{
			TemplateID:       packageTemplateID,
			OutputDir:        packageOutputDir,
			IncludeAnalytics: packageIncludeAnalytics,
			LogsLimit:        packageLogsLimit,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Package for %s created at %s\n", info.TemplateID, info.OutputPath)
		if info.AnalyticsFile != "" {
			fmt.Printf("Analytics snapshot: %s\n", info.AnalyticsFile)
		}
		fmt.Printf("Dataset: %s\nLogs: %s\n", info.DatasetFile, info.LogsFile)
		return nil
	},
}

var packageImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import a portable prompt package",
	RunE: func(cmd *cobra.Command, args []string) error {
		if packageImportPath == "" {
			return fmt.Errorf("--path is required")
		}

		cfgManager := config.NewSimpleConfigManager()
		if configFile != "" {
			if err := cfgManager.Load(configFile); err != nil {
				return fmt.Errorf("load config: %w", err)
			}
		}

		importer := packaging.NewImporter(cfgManager.GetConfig())
		result, err := importer.Import(context.Background(), packaging.ImportOptions{
			PackagePath:      packageImportPath,
			RestoreAnalytics: packageImportRestoreAnalytics,
			Overwrite:        packageImportOverwrite,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Template %s imported.\n", result.TemplateID)
		fmt.Printf("Manifest: %s\n", result.ManifestPath)
		fmt.Printf("Dataset: %s\n", result.DatasetPath)
		if result.LogsPath != "" {
			fmt.Printf("Logs: %s\n", result.LogsPath)
		}
		if result.AnalyticsPath != "" {
			fmt.Printf("Analytics: %s\n", result.AnalyticsPath)
		}
		return nil
	},
}

func init() {
	packageExportCmd.Flags().StringVar(&packageTemplateID, "template-id", "", "ID of the template to package")
	packageExportCmd.Flags().StringVar(&packageOutputDir, "output-dir", "packages", "Directory to place exported packages")
	packageExportCmd.Flags().BoolVar(&packageIncludeAnalytics, "include-analytics", true, "Copy the analytics DuckDB snapshot")
	packageExportCmd.Flags().IntVar(&packageLogsLimit, "logs-limit", 100, "Number of recent records to include in logs")

	packageCmd.AddCommand(packageExportCmd)
	packageImportCmd.Flags().StringVar(&packageImportPath, "path", "", "Path to the package directory")
	packageImportCmd.Flags().BoolVar(&packageImportRestoreAnalytics, "restore-analytics", true, "Restore the packaged analytics snapshot")
	packageImportCmd.Flags().BoolVar(&packageImportOverwrite, "overwrite", false, "Overwrite existing files (manifest, dataset, analytics)")
	packageCmd.AddCommand(packageImportCmd)

	rootCmd.AddCommand(packageCmd)
}
