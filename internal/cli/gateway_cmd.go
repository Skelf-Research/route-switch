package cli

import (
	"fmt"

	"github.com/skelf-research/route-switch/internal/config"
	"github.com/spf13/cobra"
)

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Gateway management helpers",
}

var gatewayCombosCmd = &cobra.Command{
	Use:   "combinations",
	Short: "List configured prompt combinations",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgManager := config.NewSimpleConfigManager()
		if configFile != "" {
			if err := cfgManager.Load(configFile); err != nil {
				return fmt.Errorf("load config: %w", err)
			}
		}
		appConfig := cfgManager.GetConfig()
		if len(appConfig.Gateway.Combinations) == 0 {
			fmt.Println("No gateway combinations configured")
			return nil
		}
		for _, combo := range appConfig.Gateway.Combinations {
			status := "disabled"
			if combo.Enabled {
				status = "enabled"
			}
			fmt.Printf("- %s (%s) model=%s provider=%s weight=%d [%s]\n",
				combo.ID, combo.Name, combo.Model, combo.Provider, combo.Weight, status)
		}
		return nil
	},
}

func init() {
	gatewayCmd.AddCommand(gatewayCombosCmd)
	rootCmd.AddCommand(gatewayCmd)
}
