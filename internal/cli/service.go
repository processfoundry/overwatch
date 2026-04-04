package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage system service installation",
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install overwatch as a system service",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("overwatch service install: not yet implemented")
		return nil
	},
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the overwatch system service",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("overwatch service uninstall: not yet implemented")
		return nil
	},
}

func init() {
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
}
