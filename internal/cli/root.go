package cli

import (
	"fmt"
	"os"

	"github.com/skelf-research/route-switch/internal/core"
	"github.com/spf13/cobra"
)

var (
	prompt          string
	model           string
	optimizePrompt  bool
	findBestModel   bool
	help            bool
)

var rootCmd = &cobra.Command{
	Use:   "route-switch",
	Short: "Route-Switch optimizes prompts and finds the best models using MIPROv2",
	Long: `Route-Switch is a tool that implements MIPROv2 for prompt optimization and model switching.
It can optimize your existing prompt or find the best model for your prompt while keeping cost in mind.`,
	Run: func(cmd *cobra.Command, args []string) {
		if help {
			cmd.Help()
			return
		}

		if prompt == "" {
			fmt.Println("Error: prompt is required")
			cmd.Help()
			os.Exit(1)
		}

		// Initialize the optimizer service
		service := core.NewService()

		// Handle different operation modes
		switch {
		case optimizePrompt:
			result, err := service.OptimizePrompt(prompt, model)
			if err != nil {
				fmt.Printf("Error optimizing prompt: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Optimized Prompt: %s\n", result.OptimizedPrompt)
			fmt.Printf("Model: %s\n", result.Model)
		case findBestModel:
			result, err := service.FindBestModel(prompt, model)
			if err != nil {
				fmt.Printf("Error finding best model: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Optimized Prompt: %s\n", result.OptimizedPrompt)
			fmt.Printf("Best Model: %s\n", result.Model)
			fmt.Printf("Cost: $%.4f\n", result.Cost)
		default:
			fmt.Println("Please specify an operation mode: --optimize-prompt or --find-best-model")
			cmd.Help()
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.Flags().StringVarP(&prompt, "prompt", "p", "", "The input prompt to optimize")
	rootCmd.Flags().StringVarP(&model, "model", "m", "", "The initial model to work with")
	rootCmd.Flags().BoolVarP(&optimizePrompt, "optimize-prompt", "o", false, "Optimize the given prompt for the specified model")
	rootCmd.Flags().BoolVarP(&findBestModel, "find-best-model", "f", false, "Find the best model and optimized prompt combination")
	rootCmd.Flags().BoolVarP(&help, "help", "h", false, "Display help information")
}

func Execute() error {
	return rootCmd.Execute()
}