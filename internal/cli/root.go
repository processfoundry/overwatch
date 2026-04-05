package cli

import (
	"fmt"
	"os"

	"github.com/christianmscott/overwatch/internal/tui"
	"github.com/spf13/cobra"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "overwatch",
	Short: "Monitoring tool for sites, services, jobs, and more",
	Long:  "Overwatch is an open source monitoring tool that runs health checks and alerts on failures.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !hasServerConfig() && !hasClientConfig() {
			fmt.Println("No configuration found. Run 'overwatch init' to get started.")
			return nil
		}
		return tui.Run()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: overwatch.yaml)")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(alertCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(workerCmd)
	rootCmd.AddCommand(versionCmd)
}
