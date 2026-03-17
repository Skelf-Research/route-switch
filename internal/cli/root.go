package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/skelf-research/route-switch/internal/analytics"
	"github.com/skelf-research/route-switch/internal/config"
	"github.com/skelf-research/route-switch/internal/core"
	"github.com/skelf-research/route-switch/internal/gateway"
	"github.com/skelf-research/route-switch/internal/models"
	"github.com/skelf-research/route-switch/internal/optimizer"
	"github.com/skelf-research/route-switch/internal/storage/dataset"
	"github.com/skelf-research/route-switch/internal/templates"
	"github.com/spf13/cobra"
)

var (
	prompt         string
	model          string
	modelProvider  string
	templateRef    string
	configFile     string
	optimizePrompt bool
	findBestModel  bool
	startGateway   bool
	gatewayAddr    string
	help           bool
	evalStrategy   string
)

var rootCmd = &cobra.Command{
	Use:   "route-switch",
	Short: "Route-Switch optimizes prompts and finds the best models using MIPROv2",
	Long: `Route-Switch is a tool that implements MIPROv2 for prompt optimization and model switching.
It can optimize your existing prompt or find the best model for your prompt while keeping cost in mind.
Route-Switch can also run as an advanced gateway for multiple prompt+model combinations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if help {
			return cmd.Help()
		}

		// Load configuration
		configManager := config.NewSimpleConfigManager()
		if configFile != "" {
			if err := configManager.Load(configFile); err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
		}

		prompt, model, modelProvider, templateManifest, err := applyTemplateDefaults(cmd, configManager.GetConfig(), prompt, model, modelProvider)
		if err != nil {
			return fmt.Errorf("applying template defaults: %w", err)
		}

		if startGateway {
			return runGatewayE(cmd, configManager, templateManifest, prompt, model, modelProvider)
		}

		// Validate required parameters for command-line operations
		if prompt == "" {
			cmd.Help()
			return fmt.Errorf("prompt is required (use --template-id or --prompt)")
		}

		if model == "" {
			cmd.Help()
			return fmt.Errorf("model is required (use --template-id or --model)")
		}

		datasetStore, err := dataset.NewSQLiteStore(configManager.GetConfig().Dataset.BasePath, configManager.GetConfig().Dataset.MaxRecords)
		if err != nil {
			return fmt.Errorf("initializing dataset store: %w", err)
		}
		defer datasetStore.Close()

		selectedStrategy := evalStrategy
		if selectedStrategy == "" && templateManifest != nil && templateManifest.Metadata != nil {
			if val, ok := templateManifest.Metadata["evaluation_strategy"].(string); ok {
				selectedStrategy = val
			}
		}
		if selectedStrategy == "" {
			selectedStrategy = configManager.GetConfig().Evaluation.DefaultStrategy
		}

		provider, err := newModelProvider(modelProvider, configManager.GetConfig())
		if err != nil {
			return fmt.Errorf("initializing model provider: %w", err)
		}

		// Initialize evaluation strategy
		evaluator, err := models.NewEvaluationStrategy(selectedStrategy)
		if err != nil {
			return fmt.Errorf("initializing evaluation strategy: %w", err)
		}

		// Initialize Bayesian optimizer
		bayesianOpt, err := optimizer.NewGoptunaBayesianOptimizer(map[string]interface{}{
			"num_trials": configManager.GetConfig().MiproV2.NumTrials,
		})
		if err != nil {
			return fmt.Errorf("initializing Bayesian optimizer: %w", err)
		}

		// Initialize the MIPROv2 optimizer with all dependencies
		opt := optimizer.NewMIPROv2(provider, evaluator, bayesianOpt, configManager.GetConfig().MiproV2)

		// Set up service configuration
		serviceConfig := &core.ServiceConfig{
			ModelProvider: provider,
			Evaluator:     evaluator,
			Optimizer:     opt,
			Config:        configManager.GetConfig(),
			DatasetStore:  datasetStore,
		}

		// Initialize the optimizer service
		service := core.NewService(serviceConfig)

		// Handle different operation modes
		switch {
		case optimizePrompt:
			templateID := ""
			if templateManifest != nil {
				templateID = templateManifest.ID
			}

			result, err := service.OptimizePromptWithTemplate(prompt, model, templateID)
			if err != nil {
				return fmt.Errorf("optimizing prompt: %w", err)
			}
			fmt.Printf("Optimized Prompt: %s\n", result.OptimizedPrompt)
			fmt.Printf("Model: %s\n", result.Model)
			if result.Cost > 0 {
				fmt.Printf("Cost: $%.6f\n", result.Cost)
			}
		case findBestModel:
			templateID := ""
			if templateManifest != nil {
				templateID = templateManifest.ID
			}

			result, err := service.FindBestModelWithTemplate(prompt, model, templateID)
			if err != nil {
				return fmt.Errorf("finding best model: %w", err)
			}
			fmt.Printf("Optimized Prompt: %s\n", result.OptimizedPrompt)
			fmt.Printf("Best Model: %s\n", result.Model)
			fmt.Printf("Cost: $%.6f\n", result.Cost)
			if result.ImprovementScore > 0 {
				fmt.Printf("Improvement Score: %.4f\n", result.ImprovementScore)
			}
		default:
			cmd.Help()
			return fmt.Errorf("please specify an operation mode: --optimize-prompt or --find-best-model")
		}

		// Close the provider when done
		provider.Close()
		return nil
	},
}

// runGatewayE starts the gateway server and returns any errors
func runGatewayE(cmd *cobra.Command, configManager *config.SimpleConfigManager, templateManifest *templates.Manifest, prompt, model, providerAlias string) error {
	appConfig := configManager.GetConfig()

	provider, err := newModelProvider(providerAlias, appConfig)
	if err != nil {
		return fmt.Errorf("initializing model provider: %w", err)
	}

	datasetStore, err := dataset.NewSQLiteStore(appConfig.Dataset.BasePath, appConfig.Dataset.MaxRecords)
	if err != nil {
		return fmt.Errorf("setting up dataset store: %w", err)
	}
	defer datasetStore.Close()

	var analyticsStore analytics.AnalyticsStore
	switch strings.ToLower(appConfig.Analytics.Driver) {
	case "", "duckdb":
		analyticsStore, err = analytics.NewDuckDBStore(appConfig.Analytics.Path)
	default:
		err = fmt.Errorf("unsupported analytics driver %q", appConfig.Analytics.Driver)
	}
	if err != nil {
		return fmt.Errorf("initializing analytics store: %w", err)
	}
	defer analyticsStore.Close()

	selectedStrategy := evalStrategy
	if selectedStrategy == "" && templateManifest != nil && templateManifest.Metadata != nil {
		if val, ok := templateManifest.Metadata["evaluation_strategy"].(string); ok {
			selectedStrategy = val
		}
	}
	if selectedStrategy == "" {
		selectedStrategy = configManager.GetConfig().Evaluation.DefaultStrategy
	}

	evaluator, err := models.NewEvaluationStrategy(selectedStrategy)
	if err != nil {
		return fmt.Errorf("initializing evaluation strategy: %w", err)
	}

	bayesianOpt, err := optimizer.NewGoptunaBayesianOptimizer(map[string]interface{}{
		"num_trials": configManager.GetConfig().MiproV2.NumTrials,
	})
	if err != nil {
		return fmt.Errorf("initializing Bayesian optimizer: %w", err)
	}

	opt := optimizer.NewMIPROv2(provider, evaluator, bayesianOpt, configManager.GetConfig().MiproV2)

	serviceConfig := &core.ServiceConfig{
		ModelProvider: provider,
		Evaluator:     evaluator,
		Optimizer:     opt,
		Config:        configManager.GetConfig(),
		DatasetStore:  datasetStore,
	}

	gatewayConfig := &gateway.GatewayConfig{
		Addr:                 gatewayAddr,
		LoadBalancerStrategy: gateway.LoadBalancerStrategy(configManager.GetConfig().Gateway.Strategy),
		OptimizationEnabled:  configManager.GetConfig().Gateway.Optimization.Enabled,
		OptimizationInterval: time.Duration(configManager.GetConfig().Gateway.Optimization.Interval) * time.Second,
	}

	if len(appConfig.Gateway.Combinations) == 0 && prompt != "" && model != "" {
		meta := map[string]interface{}{
			"source":          "cli_args",
			"original_prompt": prompt,
		}
		if templateManifest != nil {
			meta["template_id"] = templateManifest.ID
		}
		templateID := fmt.Sprintf("default-%d", time.Now().Unix())
		if templateManifest != nil && templateManifest.ID != "" {
			templateID = templateManifest.ID
		}
		defaultCombination := config.PromptCombinationConfig{
			ID:         fmt.Sprintf("default-%d", time.Now().Unix()),
			Name:       "default",
			TemplateID: templateID,
			Prompt:     prompt,
			Model:      model,
			Provider:   providerAlias,
			IsPrimary:  true,
			Weight:     10,
			Enabled:    true,
			Metadata:   meta,
		}

		appConfig.Gateway.Combinations = append(appConfig.Gateway.Combinations, defaultCombination)
	}

	gw, err := gateway.NewGateway(serviceConfig, gatewayConfig, appConfig, datasetStore, analyticsStore)
	if err != nil {
		return fmt.Errorf("creating gateway: %w", err)
	}

	gw.RegisterProvider(providerAlias, provider)

	fmt.Printf("Starting Route-Switch Gateway on %s\n", gatewayAddr)
	fmt.Println("Gateway is ready to handle requests...")

	if err := gw.Start(); err != nil {
		return fmt.Errorf("starting gateway: %w", err)
	}
	return nil
}

func newModelProvider(providerAlias string, cfg *config.Config) (models.ModelProvider, error) {
	alias := strings.ToLower(strings.TrimSpace(providerAlias))
	if alias == "mock" {
		return models.NewMockModelProvider(), nil
	}

	gollmCfg, err := buildGollmProviderConfig(cfg, alias)
	if err != nil {
		return nil, err
	}

	provider := models.NewGollmProvider()
	if err := provider.Initialize(gollmCfg); err != nil {
		return nil, fmt.Errorf("initialize gollm provider: %w", err)
	}
	return provider, nil
}

func buildGollmProviderConfig(cfg *config.Config, alias string) (map[string]interface{}, error) {
	includeAll := alias == "" || alias == "gollm"
	configMap := make(map[string]interface{})
	providerConfigs := make(map[string]map[string]interface{})

	for name, providerCfg := range cfg.ModelProviders {
		if !includeAll && strings.ToLower(name) != alias {
			continue
		}
		key := strings.ToLower(name)
		providerConfigs[key] = map[string]interface{}{
			"api_key":    providerCfg.APIKey,
			"base_url":   providerCfg.BaseURL,
			"models":     providerCfg.Models,
			"rate_limit": providerCfg.RateLimit,
			"options":    providerCfg.Options,
		}
		if providerCfg.APIKey != "" {
			configMap[fmt.Sprintf("%s_api_key", key)] = providerCfg.APIKey
		}
	}

	if len(providerConfigs) > 0 {
		configMap["providers"] = providerConfigs
	}

	if len(providerConfigs) == 0 {
		if includeAll {
			return nil, errors.New("no model providers configured for gollm")
		}
		return nil, fmt.Errorf("provider %s not found in configuration", alias)
	}

	return configMap, nil
}

func init() {
	rootCmd.Flags().StringVarP(&prompt, "prompt", "p", "", "The input prompt to optimize")
	rootCmd.Flags().StringVarP(&model, "model", "m", "", "The initial model to work with")
	rootCmd.Flags().StringVarP(&modelProvider, "provider", "r", "gollm", "Model provider alias (gollm, mock, or a configured provider name)")
	rootCmd.Flags().StringVar(&templateRef, "template-id", "", "Use a registered template ID for prompt/model defaults")
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file path")
	rootCmd.Flags().BoolVarP(&optimizePrompt, "optimize-prompt", "o", false, "Optimize the given prompt for the specified model")
	rootCmd.Flags().BoolVarP(&findBestModel, "find-best-model", "f", false, "Find the best model and optimized prompt combination")
	rootCmd.Flags().BoolVarP(&startGateway, "gateway", "g", false, "Start as a gateway server")
	rootCmd.Flags().StringVarP(&gatewayAddr, "addr", "a", ":8080", "Gateway server address")
	rootCmd.Flags().StringVar(&evalStrategy, "evaluation-strategy", "", "Override evaluation strategy (similarity, keyword, exact)")
	rootCmd.Flags().BoolVarP(&help, "help", "h", false, "Display help information")
}

func Execute() error {
	return rootCmd.Execute()
}

func applyTemplateDefaults(cmd *cobra.Command, cfg *config.Config, prompt, model, provider string) (string, string, string, *templates.Manifest, error) {
	if templateRef == "" {
		return prompt, model, provider, nil, nil
	}

	manager, err := templates.NewManager(cfg.Dataset.BasePath)
	if err != nil {
		return prompt, model, provider, nil, err
	}

	manifest, err := manager.Load(templateRef)
	if err != nil {
		return prompt, model, provider, nil, err
	}

	if prompt == "" {
		prompt = manifest.Prompt
	}
	if model == "" {
		model = manifest.DefaultModel
	}

	providerFlagSet := cmd != nil && cmd.Flags().Changed("provider")
	if !providerFlagSet && manifest.DefaultProvider != "" {
		provider = manifest.DefaultProvider
	}

	return prompt, model, provider, manifest, nil
}
