package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var alertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Manage and test alerts",
}

var alertsTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Send a test alert",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("overwatch alerts test: not yet implemented")
		return nil
	},
}

func init() {
	alertsCmd.AddCommand(alertsTestCmd)
}
