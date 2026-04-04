package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current health and state",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("overwatch status: not yet implemented")
		return nil
	},
}
