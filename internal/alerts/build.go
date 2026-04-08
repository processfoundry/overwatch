package alerts

import (
	"github.com/christianmscott/overwatch/internal/alerts/discord"
	"github.com/christianmscott/overwatch/internal/alerts/pagerduty"
	"github.com/christianmscott/overwatch/internal/alerts/sms"
	"github.com/christianmscott/overwatch/internal/alerts/smtp"
	"github.com/christianmscott/overwatch/internal/alerts/teams"
	"github.com/christianmscott/overwatch/internal/alerts/webhook"
	"github.com/christianmscott/overwatch/pkg/spec"
)

func BuildSenders(cfg spec.AlertsConfig) []AlertSender {
	var senders []AlertSender

	for _, w := range cfg.Webhooks {
		senders = append(senders, webhook.New(w))
	}
	if cfg.SMTP != nil {
		senders = append(senders, smtp.New(*cfg.SMTP))
	}
	for _, d := range cfg.Discord {
		senders = append(senders, discord.New(d.WebhookURL))
	}
	for _, t := range cfg.Teams {
		senders = append(senders, teams.New(t.WebhookURL))
	}
	for _, p := range cfg.PagerDuty {
		senders = append(senders, pagerduty.New(p.IntegrationKey))
	}
	for _, s := range cfg.SMS {
		senders = append(senders, sms.New(s.Phone))
	}

	return senders
}
