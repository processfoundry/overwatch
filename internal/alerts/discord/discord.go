package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/christianmscott/overwatch/pkg/spec"
)

type Sender struct {
	webhookURL string
}

func New(webhookURL string) *Sender { return &Sender{webhookURL: webhookURL} }

func (s *Sender) Name() string { return "discord" }

func (s *Sender) Send(ctx context.Context, msg spec.AlertMessage) error {
	text := fmt.Sprintf("**[%s]** `%s` is **%s** (was %s)",
		msg.Timestamp.Format("15:04:05"), msg.CheckName, msg.Status, msg.PreviousStatus)
	if msg.Detail != "" {
		text += "\n" + msg.Detail
	}

	body, _ := json.Marshal(map[string]string{"content": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("discord: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("discord: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord: %s", resp.Status)
	}
	return nil
}
