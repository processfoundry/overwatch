package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/christianmscott/overwatch/pkg/spec"
	"github.com/spf13/cobra"
)

type apiCheckStatus struct {
	spec.CheckSpec
	LastResult *spec.CheckResult `json:"last_result,omitempty"`
}

type apiStatusResponse struct {
	Checks []apiCheckStatus `json:"checks"`
	Alerts spec.AlertsConfig `json:"alerts"`
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show all configured checks and alerts with live status",
	Long:  "Query the running server for live check results and configuration. Falls back to config file if the server is unreachable.",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if !hasServerConfig() && !hasClientConfig() {
			return fmt.Errorf("no configuration found — run 'overwatch init' to get started")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, err := serverAddr()
		if err != nil && !hasClientConfig() {
			return err
		}

		cfg, _ := loadCfg()

		if addr == "" && cfg != nil {
			addr = fmt.Sprintf("http://%s:%d", cfg.Server.BindAddress, cfg.Server.BindPort)
		}
		resp, apiErr := fetchStatus(addr)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

		if apiErr == nil {
			fmt.Fprintln(w, "CHECKS")
			fmt.Fprintln(w, "NAME\tTYPE\tTARGET\tSTATUS\tLATENCY\tINTERVAL\tTIMEOUT\tLAST CHECK")
			for _, c := range resp.Checks {
				status, latency, lastCheck := "-", "-", "-"
				if c.LastResult != nil {
					status = string(c.LastResult.Status)
					latency = c.LastResult.Duration.Round(time.Millisecond).String()
					lastCheck = c.LastResult.Timestamp.Format("15:04:05")
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					c.Name, c.Type, c.Target, status, latency, c.Interval.Duration, c.Timeout.Duration, lastCheck)
			}
			w.Flush()

			fmt.Println()
			w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ALERTS")
			fmt.Fprintln(w, "NAME\tTRANSPORT\tDESTINATION\tMETHOD\tTIMEOUT")
			for _, wh := range resp.Alerts.Webhooks {
				fmt.Fprintf(w, "%s\twebhook\t%s\t%s\t%s\n", wh.Name, wh.URL, wh.Method, wh.Timeout.Duration)
			}
			if smtp := resp.Alerts.SMTP; smtp != nil {
				fmt.Fprintf(w, "smtp\tsmtp\t%s:%d → %s\t-\t-\n",
					smtp.Host, smtp.Port, strings.Join(smtp.Recipients, ","))
			}
			if len(resp.Alerts.Webhooks) == 0 && resp.Alerts.SMTP == nil {
				fmt.Fprintln(w, "(none)")
			}
			w.Flush()
		} else {
			if cfg == nil {
				return fmt.Errorf("server not reachable (%s) and no local config available", apiErr)
			}
			fmt.Fprintf(os.Stderr, "server not reachable (%s), showing config only\n\n", apiErr)

			fmt.Fprintln(w, "CHECKS")
			fmt.Fprintln(w, "NAME\tTYPE\tTARGET\tSTATUS\tINTERVAL\tTIMEOUT")
			for _, c := range cfg.Checks {
				fmt.Fprintf(w, "%s\t%s\t%s\t-\t%s\t%s\n",
					c.Name, c.Type, c.Target, c.Interval.Duration, c.Timeout.Duration)
			}
			w.Flush()

			fmt.Println()
			w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ALERTS")
			fmt.Fprintln(w, "NAME\tTRANSPORT\tDESTINATION\tMETHOD\tTIMEOUT")
			for _, wh := range cfg.Alerts.Webhooks {
				fmt.Fprintf(w, "%s\twebhook\t%s\t%s\t%s\n", wh.Name, wh.URL, wh.Method, wh.Timeout.Duration)
			}
			if smtp := cfg.Alerts.SMTP; smtp != nil {
				fmt.Fprintf(w, "smtp\tsmtp\t%s:%d → %s\t-\t-\n",
					smtp.Host, smtp.Port, strings.Join(smtp.Recipients, ","))
			}
			if len(cfg.Alerts.Webhooks) == 0 && cfg.Alerts.SMTP == nil {
				fmt.Fprintln(w, "(none)")
			}
			w.Flush()
		}

		fmt.Printf("\nServer: %s\n", addr)

		return nil
	},
}

func fetchStatus(addr string) (*apiStatusResponse, error) {
	resp, err := apiDo("GET", addr+"/api/status", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %s", resp.Status)
	}
	var out apiStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
