package cli

import (
	"fmt"

	"github.com/christianmscott/overwatch/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("overwatch %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.Date)
	},
}
