package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/christianmscott/overwatch/internal/alerts"
	"github.com/christianmscott/overwatch/pkg/spec"
	"github.com/spf13/cobra"
)

var alertCmd = &cobra.Command{
	Use:   "alert",
	Short: "Manage alert destinations (add, list, remove, update, test)",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if !hasServerConfig() && !hasClientConfig() {
			return fmt.Errorf("no configuration found — run 'overwatch init' to get started")
		}
		return nil
	},
}

// --- list ---

var alertListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured alert destinations",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, err := serverAddr()
		if err != nil {
			return err
		}
		resp, err := apiDo("GET", addr+"/api/alerts", nil)
		if err != nil {
			return fmt.Errorf("cannot reach server at %s: %w\nIs 'overwatch serve' running?", addr, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return apiReadError(resp)
		}

		var ac spec.AlertsConfig
		if err := json.NewDecoder(resp.Body).Decode(&ac); err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTRANSPORT\tDESTINATION\tMETHOD\tTIMEOUT")
		for _, wh := range ac.Webhooks {
			fmt.Fprintf(w, "%s\twebhook\t%s\t%s\t%s\n", wh.Name, wh.URL, wh.Method, wh.Timeout.Duration)
		}
		if smtp := ac.SMTP; smtp != nil {
			fmt.Fprintf(w, "smtp\tsmtp\t%s:%d → %s\t-\t-\n",
				smtp.Host, smtp.Port, strings.Join(smtp.Recipients, ","))
		}
		if len(ac.Webhooks) == 0 && ac.SMTP == nil {
			fmt.Fprintln(w, "(none)")
		}
		return w.Flush()
	},
}

// --- add ---

var (
	alertAddTransport string
	alertAddURL       string
	alertAddMethod    string
	alertAddTimeout   string
	alertAddHeaders   []string
)

var alertAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new webhook alert destination",
	Long: `Add a webhook alert destination to the running server.

Examples:
  overwatch alert add slack --url https://hooks.slack.com/services/T.../B.../xxx
  overwatch alert add pagerduty --url https://events.pagerduty.com/v2/enqueue --method POST --timeout 15s
  overwatch alert add custom --url https://my.endpoint/hook --headers "Authorization:Bearer tok123"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, err := serverAddr()
		if err != nil {
			return err
		}

		wh := spec.WebhookConfig{
			Name:   args[0],
			URL:    alertAddURL,
			Method: alertAddMethod,
		}
		if alertAddTimeout != "" {
			d, err := time.ParseDuration(alertAddTimeout)
			if err != nil {
				return fmt.Errorf("invalid --timeout: %w", err)
			}
			wh.Timeout = spec.Duration{Duration: d}
		}
		if len(alertAddHeaders) > 0 {
			wh.Headers = make(map[string]string)
			for _, h := range alertAddHeaders {
				k, v, ok := strings.Cut(h, ":")
				if !ok {
					return fmt.Errorf("invalid header %q (expected Key:Value)", h)
				}
				wh.Headers[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
		}

		resp, err := apiDo(http.MethodPost, addr+"/api/alerts", wh)
		if err != nil {
			return fmt.Errorf("cannot reach server at %s: %w\nIs 'overwatch serve' running?", addr, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			return apiReadError(resp)
		}
		fmt.Printf("added alert %q (webhook)\n", args[0])
		return nil
	},
}

// --- remove ---

var alertRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an alert destination by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, err := serverAddr()
		if err != nil {
			return err
		}
		resp, err := apiDo(http.MethodDelete, addr+"/api/alerts/"+args[0], nil)
		if err != nil {
			return fmt.Errorf("cannot reach server at %s: %w\nIs 'overwatch serve' running?", addr, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return apiReadError(resp)
		}
		fmt.Printf("removed alert %q\n", args[0])
		return nil
	},
}

// --- update ---

var (
	alertUpdateURL     string
	alertUpdateMethod  string
	alertUpdateTimeout string
	alertUpdateHeaders []string
)

var alertUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Update an existing alert destination",
	Long: `Update fields on an existing webhook alert. Only flags you pass are changed.

Examples:
  overwatch alert update slack --url https://hooks.slack.com/services/NEW/URL
  overwatch alert update slack --timeout 30s
  overwatch alert update custom --headers "Authorization:Bearer newtoken"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, err := serverAddr()
		if err != nil {
			return err
		}

		patch := spec.WebhookConfig{}
		if cmd.Flags().Changed("url") {
			patch.URL = alertUpdateURL
		}
		if cmd.Flags().Changed("method") {
			patch.Method = alertUpdateMethod
		}
		if cmd.Flags().Changed("timeout") {
			d, err := time.ParseDuration(alertUpdateTimeout)
			if err != nil {
				return fmt.Errorf("invalid --timeout: %w", err)
			}
			patch.Timeout = spec.Duration{Duration: d}
		}
		if cmd.Flags().Changed("headers") {
			patch.Headers = make(map[string]string)
			for _, h := range alertUpdateHeaders {
				k, v, ok := strings.Cut(h, ":")
				if !ok {
					return fmt.Errorf("invalid header %q (expected Key:Value)", h)
				}
				patch.Headers[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
		}

		resp, err := apiDo(http.MethodPut, addr+"/api/alerts/"+args[0], patch)
		if err != nil {
			return fmt.Errorf("cannot reach server at %s: %w\nIs 'overwatch serve' running?", addr, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return apiReadError(resp)
		}
		fmt.Printf("updated alert %q\n", args[0])
		return nil
	},
}

// --- test ---

var alertTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Send a test alert through all configured senders",
	Long:  "Read alert config from the YAML file and dispatch a test message to every configured sender.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadCfg()
		if err != nil {
			return err
		}

		senders := alerts.BuildSenders(cfg.Alerts)
		if len(senders) == 0 {
			return fmt.Errorf("no alert senders configured; add webhooks or smtp to your config")
		}

		router := alerts.NewRouter(senders)
		router.SendTest()

		fmt.Printf("test alert sent to %d sender(s)\n", len(senders))
		return nil
	},
}

func init() {
	alertAddCmd.Flags().StringVar(&alertAddURL, "url", "", "webhook URL (required)")
	alertAddCmd.Flags().StringVar(&alertAddMethod, "method", "POST", "HTTP method (GET, POST, PUT)")
	alertAddCmd.Flags().StringVar(&alertAddTimeout, "timeout", "10s", "send timeout (e.g. 5s, 30s)")
	alertAddCmd.Flags().StringSliceVar(&alertAddHeaders, "headers", nil, "custom headers (Key:Value, comma-separated or repeated)")
	alertAddCmd.MarkFlagRequired("url")

	alertUpdateCmd.Flags().StringVar(&alertUpdateURL, "url", "", "new webhook URL")
	alertUpdateCmd.Flags().StringVar(&alertUpdateMethod, "method", "", "new HTTP method (GET, POST, PUT)")
	alertUpdateCmd.Flags().StringVar(&alertUpdateTimeout, "timeout", "", "new send timeout (e.g. 5s, 30s)")
	alertUpdateCmd.Flags().StringSliceVar(&alertUpdateHeaders, "headers", nil, "new headers (Key:Value)")

	alertCmd.AddCommand(alertListCmd)
	alertCmd.AddCommand(alertAddCmd)
	alertCmd.AddCommand(alertRemoveCmd)
	alertCmd.AddCommand(alertUpdateCmd)
	alertCmd.AddCommand(alertTestCmd)
}
