package pagerduty

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/christianmscott/overwatch/pkg/spec"
)

const eventsURL = "https://events.pagerduty.com/v2/enqueue"

type Sender struct {
	integrationKey string
}

func New(integrationKey string) *Sender { return &Sender{integrationKey: integrationKey} }

func (s *Sender) Name() string { return "pagerduty" }

func (s *Sender) Send(ctx context.Context, msg spec.AlertMessage) error {
	action := "trigger"
	severity := "error"
	if msg.Status == spec.StatusUp {
		action = "resolve"
		severity = "info"
	} else if msg.Status == spec.StatusDegraded {
		severity = "warning"
	}

	payload := map[string]any{
		"routing_key":  s.integrationKey,
		"event_action": action,
		"dedup_key":    "overwatch-" + msg.CheckName,
		"payload": map[string]any{
			"summary":   fmt.Sprintf("%s is %s (was %s)", msg.CheckName, msg.Status, msg.PreviousStatus),
			"source":    "overwatch",
			"severity":  severity,
			"timestamp": msg.Timestamp.Format(time.RFC3339),
			"custom_details": map[string]string{
				"check":           msg.CheckName,
				"status":          string(msg.Status),
				"previous_status": string(msg.PreviousStatus),
				"detail":          msg.Detail,
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, eventsURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("pagerduty: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("pagerduty: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("pagerduty: %s", resp.Status)
	}
	return nil
}
