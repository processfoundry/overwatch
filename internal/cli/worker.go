package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:    "worker",
	Short:  "Run as a managed worker (Overwatch Cloud)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("managed worker mode is not yet implemented")
		return nil
	},
}
