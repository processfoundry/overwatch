package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var checksCmd = &cobra.Command{
	Use:   "checks",
	Short: "Manage and test checks",
}

var checksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured checks",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("overwatch checks list: not yet implemented")
		return nil
	},
}

var checksTestCmd = &cobra.Command{
	Use:   "test [name]",
	Short: "Run a check by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("overwatch checks test %s: not yet implemented\n", args[0])
		return nil
	},
}

func init() {
	checksCmd.AddCommand(checksListCmd)
	checksCmd.AddCommand(checksTestCmd)
}
