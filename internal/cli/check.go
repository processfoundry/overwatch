package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/christianmscott/overwatch/internal/checks"
	"github.com/christianmscott/overwatch/pkg/spec"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Manage checks (add, list, remove, update, test)",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if !hasServerConfig() && !hasClientConfig() {
			return fmt.Errorf("no configuration found — run 'overwatch init' to get started")
		}
		return nil
	},
}

// --- list ---

var checkListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured checks",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, err := serverAddr()
		if err != nil {
			return err
		}
		resp, err := apiDo("GET", addr+"/api/checks", nil)
		if err != nil {
			return fmt.Errorf("cannot reach server at %s: %w\nIs 'overwatch serve' running?", addr, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return apiReadError(resp)
		}

		var list []spec.CheckSpec
		if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tTARGET\tINTERVAL\tTIMEOUT\tALERTS")
		for _, c := range list {
			al := "-"
			if len(c.Alerts) > 0 {
				al = strings.Join(c.Alerts, ",")
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				c.Name, c.Type, c.Target, c.Interval.Duration, c.Timeout.Duration, al)
		}
		return w.Flush()
	},
}

// --- add ---

var (
	checkAddType     string
	checkAddTarget   string
	checkAddInterval string
	checkAddTimeout  string
	checkAddAlerts   []string
	checkAddExpected int
	checkAddSilence  string
)

var checkAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new check",
	Long: `Add a new check to the running server.

Examples:
  overwatch check add my-api --type http --target https://api.example.com --interval 30s
  overwatch check add db --type tcp --target localhost:5432 --timeout 5s
  overwatch check add cert --type tls --target example.com:443 --interval 1h
  overwatch check add ns --type dns --target example.com --interval 5m
  overwatch check add nightly-job --type checkin --max-silence 25h --interval 1m`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, err := serverAddr()
		if err != nil {
			return err
		}

		c := spec.CheckSpec{
			Name:           args[0],
			Type:           spec.CheckType(checkAddType),
			Target:         checkAddTarget,
			ExpectedStatus: checkAddExpected,
			Alerts:         checkAddAlerts,
		}
		if d, err := time.ParseDuration(checkAddInterval); err != nil {
			return fmt.Errorf("invalid --interval: %w", err)
		} else {
			c.Interval = spec.Duration{Duration: d}
		}
		if d, err := time.ParseDuration(checkAddTimeout); err != nil {
			return fmt.Errorf("invalid --timeout: %w", err)
		} else {
			c.Timeout = spec.Duration{Duration: d}
		}
		if checkAddSilence != "" {
			d, err := time.ParseDuration(checkAddSilence)
			if err != nil {
				return fmt.Errorf("invalid --max-silence: %w", err)
			}
			c.MaxSilence = spec.Duration{Duration: d}
		}

		resp, err := apiDo(http.MethodPost, addr+"/api/checks", c)
		if err != nil {
			return fmt.Errorf("cannot reach server at %s: %w\nIs 'overwatch serve' running?", addr, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			return apiReadError(resp)
		}
		fmt.Printf("added check %q\n", args[0])
		return nil
	},
}

// --- remove ---

var checkRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a check by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, err := serverAddr()
		if err != nil {
			return err
		}
		resp, err := apiDo(http.MethodDelete, addr+"/api/checks/"+args[0], nil)
		if err != nil {
			return fmt.Errorf("cannot reach server at %s: %w\nIs 'overwatch serve' running?", addr, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return apiReadError(resp)
		}
		fmt.Printf("removed check %q\n", args[0])
		return nil
	},
}

// --- update ---

var (
	checkUpdateTarget   string
	checkUpdateInterval string
	checkUpdateTimeout  string
	checkUpdateAlerts   []string
	checkUpdateExpected int
	checkUpdateSilence  string
)

var checkUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Update an existing check",
	Long: `Update fields on an existing check. Only flags you pass are changed.

Examples:
  overwatch check update my-api --interval 30s
  overwatch check update my-api --target https://new-api.example.com --timeout 5s
  overwatch check update my-api --alerts slack,pagerduty
  overwatch check update nightly-job --max-silence 48h`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, err := serverAddr()
		if err != nil {
			return err
		}

		patch := spec.CheckSpec{}
		if cmd.Flags().Changed("target") {
			patch.Target = checkUpdateTarget
		}
		if cmd.Flags().Changed("interval") {
			d, err := time.ParseDuration(checkUpdateInterval)
			if err != nil {
				return fmt.Errorf("invalid --interval: %w", err)
			}
			patch.Interval = spec.Duration{Duration: d}
		}
		if cmd.Flags().Changed("timeout") {
			d, err := time.ParseDuration(checkUpdateTimeout)
			if err != nil {
				return fmt.Errorf("invalid --timeout: %w", err)
			}
			patch.Timeout = spec.Duration{Duration: d}
		}
		if cmd.Flags().Changed("expected-status") {
			patch.ExpectedStatus = checkUpdateExpected
		}
		if cmd.Flags().Changed("alerts") {
			patch.Alerts = checkUpdateAlerts
		}
		if cmd.Flags().Changed("max-silence") {
			d, err := time.ParseDuration(checkUpdateSilence)
			if err != nil {
				return fmt.Errorf("invalid --max-silence: %w", err)
			}
			patch.MaxSilence = spec.Duration{Duration: d}
		}

		resp, err := apiDo(http.MethodPut, addr+"/api/checks/"+args[0], patch)
		if err != nil {
			return fmt.Errorf("cannot reach server at %s: %w\nIs 'overwatch serve' running?", addr, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return apiReadError(resp)
		}
		fmt.Printf("updated check %q\n", args[0])
		return nil
	},
}

// --- test ---

var checkTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Run a check immediately and print the result",
	Long: `Execute a check once and display the result. Reads the check definition
from the config file and runs it locally (does not require a running server).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadCfg()
		if err != nil {
			return err
		}
		name := args[0]
		for _, c := range cfg.Checks {
			if c.Name == name {
				result := checks.Run(cmd.Context(), c)
				fmt.Printf("check:    %s\n", result.CheckName)
				fmt.Printf("status:   %s\n", result.Status)
				fmt.Printf("duration: %s\n", result.Duration)
				if result.Error != "" {
					fmt.Printf("error:    %s\n", result.Error)
				}
				if result.Status == spec.StatusDown {
					os.Exit(1)
				}
				return nil
			}
		}
		return fmt.Errorf("check %q not found in config", name)
	},
}

func init() {
	checkAddCmd.Flags().StringVar(&checkAddType, "type", "http", "check type: http, tcp, tls, dns, checkin")
	checkAddCmd.Flags().StringVar(&checkAddTarget, "target", "", "target (URL, host:port, or hostname depending on type)")
	checkAddCmd.Flags().StringVar(&checkAddInterval, "interval", "60s", "how often to run (e.g. 30s, 5m, 1h)")
	checkAddCmd.Flags().StringVar(&checkAddTimeout, "timeout", "10s", "max time per check (e.g. 5s, 30s)")
	checkAddCmd.Flags().StringSliceVar(&checkAddAlerts, "alerts", nil, "alert names to notify on failure (comma-separated)")
	checkAddCmd.Flags().IntVar(&checkAddExpected, "expected-status", 0, "expected HTTP status code (http type only)")
	checkAddCmd.Flags().StringVar(&checkAddSilence, "max-silence", "", "max time without check-in before alerting (e.g. 25h; checkin type only)")

	checkUpdateCmd.Flags().StringVar(&checkUpdateTarget, "target", "", "new target (URL, host:port, or hostname)")
	checkUpdateCmd.Flags().StringVar(&checkUpdateInterval, "interval", "", "new interval (e.g. 30s, 5m, 1h)")
	checkUpdateCmd.Flags().StringVar(&checkUpdateTimeout, "timeout", "", "new timeout (e.g. 5s, 30s)")
	checkUpdateCmd.Flags().StringSliceVar(&checkUpdateAlerts, "alerts", nil, "alert names to notify (comma-separated)")
	checkUpdateCmd.Flags().IntVar(&checkUpdateExpected, "expected-status", 0, "expected HTTP status code")
	checkUpdateCmd.Flags().StringVar(&checkUpdateSilence, "max-silence", "", "new max silence (e.g. 25h; checkin type)")

	checkCmd.AddCommand(checkListCmd)
	checkCmd.AddCommand(checkAddCmd)
	checkCmd.AddCommand(checkRemoveCmd)
	checkCmd.AddCommand(checkUpdateCmd)
	checkCmd.AddCommand(checkTestCmd)
}
