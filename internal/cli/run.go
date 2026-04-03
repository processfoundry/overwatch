package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the local self-hosted runtime",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("overwatch run: not yet implemented")
		return nil
	},
}
