package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start the managed API-polled worker",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("overwatch worker: not yet implemented")
		return nil
	},
}
