package teams

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

func (s *Sender) Name() string { return "teams" }

func (s *Sender) Send(ctx context.Context, msg spec.AlertMessage) error {
	card := map[string]any{
		"type":    "message",
		"summary": fmt.Sprintf("%s is %s", msg.CheckName, msg.Status),
		"attachments": []map[string]any{{
			"contentType": "application/vnd.microsoft.card.adaptive",
			"content": map[string]any{
				"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
				"type":    "AdaptiveCard",
				"version": "1.4",
				"body": []map[string]any{
					{"type": "TextBlock", "size": "Medium", "weight": "Bolder",
						"text": fmt.Sprintf("%s is %s", msg.CheckName, msg.Status)},
					{"type": "TextBlock", "isSubtle": true,
						"text": fmt.Sprintf("Previous: %s | Time: %s", msg.PreviousStatus, msg.Timestamp.Format("15:04:05"))},
				},
			},
		}},
	}
	if msg.Detail != "" {
		content := card["attachments"].([]map[string]any)[0]["content"].(map[string]any)
		body := content["body"].([]map[string]any)
		body = append(body, map[string]any{"type": "TextBlock", "text": msg.Detail, "wrap": true})
		content["body"] = body
	}

	body, _ := json.Marshal(card)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("teams: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("teams: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("teams: %s", resp.Status)
	}
	return nil
}
