package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/christianmscott/overwatch/internal/config"
	"github.com/christianmscott/overwatch/internal/runtime"
	"github.com/spf13/cobra"
)

var (
	bindAddress string
	bindPort    int
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the self-hosted monitoring server",
	Long:  "Run the Overwatch server: loads checks and alerts from YAML, executes monitors, sends alerts, and exposes an API.",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgFile
		if path == "" {
			path = config.DefaultPath
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			slog.Info("no config file found, creating starter config", "path", path)
			if err := os.WriteFile(path, []byte(config.StarterConfig), 0644); err != nil {
				return fmt.Errorf("writing starter config: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Created starter config at %s\n", path)
			fmt.Fprintf(os.Stderr, "Edit it to add your checks and alerts, then restart or send SIGHUP to reload.\n\n")
		}

		cfg, err := config.Load(path)
		if err != nil {
			return err
		}

		if cmd.Flags().Changed("bind-address") {
			cfg.Server.BindAddress = bindAddress
		}
		if cmd.Flags().Changed("bind-port") {
			cfg.Server.BindPort = bindPort
		}

		return runtime.NewEngine(cfg, path).Run(cmd.Context())
	},
}

func init() {
	serveCmd.Flags().StringVar(&bindAddress, "bind-address", "127.0.0.1", "address to bind the API server")
	serveCmd.Flags().IntVar(&bindPort, "bind-port", 3030, "port to bind the API server")
}
