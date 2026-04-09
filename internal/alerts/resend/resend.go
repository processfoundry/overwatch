package resend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/christianmscott/overwatch/pkg/spec"
)

var apiURL = "https://api.resend.com/emails"

type Sender struct {
	apiKey    string
	from      string
	recipient string
}

func New(apiKey, from, recipient string) *Sender {
	return &Sender{apiKey: apiKey, from: from, recipient: recipient}
}

func (s *Sender) Name() string { return "resend" }

func (s *Sender) Send(ctx context.Context, msg spec.AlertMessage) error {
	subject := fmt.Sprintf("[overwatch] %s is %s", msg.CheckName, msg.Status)

	payload, _ := json.Marshal(map[string]any{
		"from":    s.from,
		"to":      []string{s.recipient},
		"subject": subject,
		"text":    formatBody(msg),
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("resend: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("resend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("resend: %s — %s", resp.Status, string(body))
	}
	return nil
}

func formatBody(msg spec.AlertMessage) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Check:    %s\n", msg.CheckName)
	fmt.Fprintf(&b, "Status:   %s\n", msg.Status)
	fmt.Fprintf(&b, "Previous: %s\n", msg.PreviousStatus)
	fmt.Fprintf(&b, "Time:     %s\n", msg.Timestamp.Format(time.RFC3339))
	if msg.Detail != "" {
		fmt.Fprintf(&b, "Detail:   %s\n", msg.Detail)
	}
	return b.String()
}
