package config

import (
	"fmt"
	"os"

	"github.com/christianmscott/overwatch/internal/auth"
	"github.com/christianmscott/overwatch/pkg/spec"
	"gopkg.in/yaml.v3"
)

func WriteStarterWithJoinToken(path string) error {
	var cfg spec.Config
	if err := yaml.Unmarshal([]byte(StarterConfig), &cfg); err != nil {
		return fmt.Errorf("parsing starter config: %w", err)
	}

	token, err := auth.GenerateJoinToken(cfg.Server.TokenAddress())
	if err != nil {
		return err
	}
	cfg.Server.JoinToken = token

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

const StarterConfig = `# Overwatch configuration
# See: https://github.com/christianmscott/overwatch

server:
  bind_address: 127.0.0.1
  bind_port: 3030
  # external_address: overwatch.example.com  # hostname or IP clients use to reach this server
  concurrency: 4

checks:
  - name: example-http
    type: http
    target: https://example.com
    interval: 60s
    timeout: 10s
    expected_status: 200

  # - name: example-tcp
  #   type: tcp
  #   target: localhost:5432
  #   interval: 30s
  #   timeout: 5s

  # - name: example-tls
  #   type: tls
  #   target: example.com:443
  #   interval: 1h
  #   timeout: 10s

  # - name: example-dns
  #   type: dns
  #   target: example.com
  #   interval: 5m
  #   timeout: 5s

  # - name: nightly-backup
  #   type: checkin
  #   max_silence: 25h
  #   interval: 1m
  #   timeout: 5s

alerts:
  webhooks: []
  # - name: slack
  #   url: https://hooks.slack.com/services/...
  #   method: POST
  #   timeout: 10s

  # smtp:
  #   host: smtp.example.com
  #   port: 587
  #   tls: true
  #   username: user
  #   password: pass
  #   from: overwatch@example.com
  #   recipients:
  #     - oncall@example.com
`
