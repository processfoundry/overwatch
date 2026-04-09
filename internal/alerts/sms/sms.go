package sms

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/christianmscott/overwatch/pkg/spec"
)

type Sender struct {
	phone string
}

func New(phone string) *Sender { return &Sender{phone: phone} }

func (s *Sender) Name() string { return "sms" }

func (s *Sender) Send(ctx context.Context, msg spec.AlertMessage) error {
	sid := os.Getenv("TWILIO_SID")
	token := os.Getenv("TWILIO_AUTH_TOKEN")
	from := os.Getenv("TWILIO_FROM")
	if sid == "" || token == "" || from == "" {
		return fmt.Errorf("sms: TWILIO_SID, TWILIO_AUTH_TOKEN, and TWILIO_FROM env vars required")
	}

	text := fmt.Sprintf("[overwatch] %s is %s (was %s)", msg.CheckName, msg.Status, msg.PreviousStatus)
	if msg.Detail != "" {
		text += " — " + msg.Detail
	}

	data := url.Values{}
	data.Set("To", s.phone)
	data.Set("From", from)
	data.Set("Body", text)

	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", sid)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("sms: %w", err)
	}
	req.SetBasicAuth(sid, token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sms: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("sms: twilio returned %s", resp.Status)
	}
	return nil
}
