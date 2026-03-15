package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/skelf-research/route-switch/internal/config"
	"github.com/skelf-research/route-switch/internal/templates"
	"github.com/spf13/cobra"
)

var (
	templateID              string
	templateName            string
	templatePromptText      string
	templatePromptFile      string
	templateVariables       string
	templateDefaultModel    string
	templateDefaultProvider string
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage prompt templates",
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered prompt templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgManager := config.NewSimpleConfigManager()
		if configFile != "" {
			if err := cfgManager.Load(configFile); err != nil {
				return fmt.Errorf("load config: %w", err)
			}
		}
		manager, err := templates.NewManager(cfgManager.GetConfig().Dataset.BasePath)
		if err != nil {
			return err
		}
		manifests, err := manager.List()
		if err != nil {
			return err
		}
		if len(manifests) == 0 {
			fmt.Println("No templates registered")
			return nil
		}
		for _, manifest := range manifests {
			fmt.Printf("- %s (%s)\n", manifest.ID, manifest.Name)
		}
		return nil
	},
}

var templateRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new prompt template",
	RunE: func(cmd *cobra.Command, args []string) error {
		if templateID == "" {
			return fmt.Errorf("--template-id is required")
		}

		promptText := templatePromptText
		if templatePromptFile != "" {
			data, err := os.ReadFile(templatePromptFile)
			if err != nil {
				return fmt.Errorf("read prompt file: %w", err)
			}
			promptText = string(data)
		}

		if strings.TrimSpace(promptText) == "" {
			return fmt.Errorf("prompt text is required")
		}

		cfgManager := config.NewSimpleConfigManager()
		if configFile != "" {
			if err := cfgManager.Load(configFile); err != nil {
				return fmt.Errorf("load config: %w", err)
			}
		}
		appConfig := cfgManager.GetConfig()

		manager, err := templates.NewManager(appConfig.Dataset.BasePath)
		if err != nil {
			return err
		}

		manifest := &templates.Manifest{
			ID:              templateID,
			Name:            templateName,
			Prompt:          promptText,
			Variables:       templates.ParseVariables(templateVariables),
			DefaultModel:    templateDefaultModel,
			DefaultProvider: templateDefaultProvider,
			Metadata: map[string]interface{}{
				"source": "cli",
			},
		}

		path, err := manager.Register(manifest)
		if err != nil {
			return err
		}

		fmt.Printf("Template %q registered at %s\n", templateID, path)
		return nil
	},
}

func init() {
	templateRegisterCmd.Flags().StringVar(&templateID, "template-id", "", "Unique ID for the template")
	templateRegisterCmd.Flags().StringVar(&templateName, "name", "", "Human-readable template name")
	templateRegisterCmd.Flags().StringVar(&templatePromptText, "prompt-text", "", "Prompt text (use --prompt-file for files)")
	templateRegisterCmd.Flags().StringVar(&templatePromptFile, "prompt-file", "", "Path to file containing prompt text")
	templateRegisterCmd.Flags().StringVar(&templateVariables, "variables", "", "Comma-separated list of variable names (e.g. user,topic)")
	templateRegisterCmd.Flags().StringVar(&templateDefaultModel, "default-model", "", "Default model for this template")
	templateRegisterCmd.Flags().StringVar(&templateDefaultProvider, "default-provider", "", "Default provider alias for this template")

	templateCmd.AddCommand(templateRegisterCmd)
	templateCmd.AddCommand(templateListCmd)
	rootCmd.AddCommand(templateCmd)
}
