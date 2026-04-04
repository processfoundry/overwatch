package cli

import (
	"github.com/christianmscott/overwatch/internal/config"
	"github.com/christianmscott/overwatch/internal/runtime"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the local self-hosted runtime",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgFile
		if path == "" {
			path = config.DefaultPath
		}

		cfg, err := config.Load(path)
		if err != nil {
			return err
		}

		return runtime.NewEngine(cfg).Run(cmd.Context())
	},
}
