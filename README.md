# Overwatch

**Infrastructure monitoring from the command line.** Know when services, endpoints, certificates, and scheduled jobs fail—without living in a browser.

Overwatch ships as a **single Go binary**. Run `overwatch serve` to start a self-hosted monitoring server, define checks and alerts (webhooks, email) in YAML, and manage everything from the CLI. Optionally connect to [Overwatch Cloud](https://overwatchapp.dev) for hosted monitoring with no server to run, multi-region checks, and integrations.

## How it works

**Self-hosted** — `overwatch serve` runs the monitoring server. Checks and alerts are defined in a YAML config file (the source of truth). The server executes checks on a schedule, sends alerts on state changes, and exposes an API that the CLI talks to. Edit the YAML directly and send SIGHUP (osr `POST /api/reload`) to reload, or use the CLI to add/remove/update checks and alerts (changes are persisted back to YAML).

**Client** — The CLI stores connection state (server address, Ed25519 keypair) under `~/.overwatch/`. Multiple machines can manage the same server; each joins with a token and gets its own keypair.

**Overwatch Cloud** — Same CLI, no server to run. The CLI talks to the hosted backend instead. The `overwatch init` flow handles setup for either mode.

---

## Install

### macOS (Homebrew)

```bash
brew install processfoundry/tap/overwatch
```

### Linux

```bash
curl -sLO "https://github.com/processfoundry/overwatch/releases/latest/download/overwatch_linux_amd64.tar.gz"
tar xzf overwatch_linux_amd64.tar.gz
sudo mv overwatch /usr/local/bin/
```

For ARM64, replace `amd64` with `arm64` in the URL above.

### Windows

```powershell
Invoke-WebRequest "https://github.com/processfoundry/overwatch/releases/latest/download/overwatch_windows_amd64.tar.gz" -OutFile overwatch.tar.gz
tar xzf overwatch.tar.gz
New-Item -ItemType Directory -Force -Path "C:\overwatch" | Out-Null
Move-Item overwatch.exe "C:\overwatch\overwatch.exe" -Force
# Add C:\overwatch to your PATH if not already present:
# Add C:\overwatch to your PATH if not already present:
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\overwatch", "User")
```

### From source

```bash
go install github.com/processfoundry/overwatch/cmd/overwatch@latest
```

---

## Quick start

### 1. Start the server

```bash
overwatch serve
```

Creates a starter `overwatch.yaml` if one doesn't exist, generates a **join token**, and starts the API + scheduler on `127.0.0.1:3030`. The join token is printed to stderr — copy it for the next step.

```bash
overwatch serve --bind-address 0.0.0.0 --bind-port 3030
```

### 2. Connect a client

On any machine that should manage this server:

```bash
overwatch init
```

Select "Setup a client", paste the join token. This generates an Ed25519 keypair under `~/.overwatch/`, registers it with the server, and saves the connection config. All subsequent CLI commands are signed automatically.

### 3. Add checks and alerts

```bash
overwatch check add my-api \
  --type http \
  --target https://api.example.com \
  --interval 30s

overwatch alert add slack \
  --url https://hooks.slack.com/services/T.../B.../xxx

overwatch check update my-api --alerts slack

overwatch status
```

`overwatch status` prints a verbose table of every check and alert with all parameters and live results — designed for both humans and scripts.

---

## Running as a service

Overwatch doesn't manage its own service lifecycle. Use your platform's process supervisor.

### Docker

The published image is available on GitHub Container Registry.

- Pick a **host directory** for config (for example `overwatch-data` beside where you run the command). Bind-mount that **folder** into the container — Docker creates an empty directory on the host on first use if needed. This avoids bind-mounting a single file path that does not exist yet, which Docker would create as a **directory** by mistake.
- Set `--config` to `overwatch.yaml` **inside** that mount. If that file is missing, `serve` writes a starter config on startup; if it already exists (e.g. you copied one in), it is loaded as usual. The join token is printed to stderr.

```bash
docker run -d \
  -p 3030:3030 \
  -v "$(pwd)/overwatch-data:/etc/overwatch" \
  ghcr.io/processfoundry/overwatch:latest \
  serve --bind-address 0.0.0.0 --config /etc/overwatch/overwatch.yaml
```

Replace `latest` with a specific version tag (e.g. `v0.3.0`) to pin releases.

### Docker Compose

Same pattern: mount a host directory, point `serve` at `overwatch.yaml` inside it.

```yaml
services:
  overwatch:
    image: ghcr.io/processfoundry/overwatch:latest
    ports:
      - "3030:3030"
    volumes:
      - ./overwatch-data:/etc/overwatch
    command:
      [
        "serve",
        "--bind-address",
        "0.0.0.0",
        "--config",
        "/etc/overwatch/overwatch.yaml",
      ]
```

### systemd (Linux)

```bash
sudo cp packaging/systemd/overwatch.service /etc/systemd/system/
sudo mkdir -p /etc/overwatch
sudo cp overwatch.yaml /etc/overwatch/overwatch.yaml
sudo systemctl daemon-reload
sudo systemctl enable --now overwatch
```

Reload config without restarting: `sudo systemctl reload overwatch`

### Windows (NSSM)

```powershell
nssm install Overwatch "C:\overwatch\overwatch.exe" "serve --bind-address 0.0.0.0 --config C:\overwatch\overwatch.yaml"
nssm set Overwatch AppDirectory "C:\overwatch"
nssm set Overwatch DisplayName "Overwatch Monitoring Server"
nssm set Overwatch Start SERVICE_AUTO_START
nssm start Overwatch
```

---

## Check types

| Type | What it checks | Key fields |
|------|---------------|------------|
| `http` | HTTP/HTTPS endpoint returns expected status | `target`, `expected_status`, `headers` |
| `tcp` | TCP port is accepting connections | `target` (host:port) |
| `tls` | TLS certificate validity and expiry | `target` (host:port) |
| `dns` | DNS name resolves | `target` (hostname) |
| `checkin` | Scheduled job reports in before a deadline | `max_silence` |

Every check has `interval` and `timeout`. All check types support `alerts` to bind specific alert destinations.

### Check-in webhooks

For cron jobs and scheduled tasks, use a `checkin` check. The server exposes a webhook endpoint for each one:

```bash
# Your cron job curls the check-in URL on success
curl -s http://overwatch.example.com:3030/api/checkin/nightly-backup

# Signal explicit failure
curl -s http://overwatch.example.com:3030/api/checkin/nightly-backup?status=fail
```

If no successful check-in arrives within `max_silence`, the check transitions to `down` and alerts fire.

---

## Alerts

**Webhooks** — Outbound HTTP requests (Slack, Teams, PagerDuty, generic endpoints). Supports custom headers for auth tokens.

**SMTP** — Email via your relay. TLS/STARTTLS, optional auth, multiple recipients.

Test an alert destination without waiting for a real failure:

```bash
overwatch alert test slack
overwatch check test my-api
```

---

## Commands

```text
overwatch                         # show help / setup prompt
overwatch init                    # interactive setup (server, client, or cloud)
overwatch serve                   # start the self-hosted server
overwatch status                  # verbose table of all checks & alerts + live results
overwatch check list|add|remove|update|test
overwatch alert list|add|remove|update|test
overwatch token                   # print the server's join token
overwatch config init             # generate a starter YAML config
overwatch config validate         # validate the config file
overwatch version                 # build/version metadata
```

Use `--help` on any command for flags and examples.

---

## Server configuration (YAML)

The YAML file is the source of truth. The CLI persists changes back to it; you can also edit it directly and reload.

```yaml
server:
  bind_address: 127.0.0.1
  bind_port: 3030
  external_address: overwatch.example.com  # what clients use to reach you
  external_port: 443                       # if behind a TLS proxy
  concurrency: 4

checks:
  - name: my-api
    type: http
    target: https://api.example.com
    interval: 30s
    timeout: 10s
    expected_status: 200
    headers:
      Authorization: Bearer tok123
    alerts: [slack]

  - name: db
    type: tcp
    target: localhost:5432
    interval: 30s
    timeout: 5s

  - name: cert
    type: tls
    target: example.com:443
    interval: 1h
    timeout: 10s

  - name: nightly-backup
    type: checkin
    max_silence: 25h
    interval: 1m
    alerts: [slack]

alerts:
  webhooks:
    - name: slack
      url: https://hooks.slack.com/services/...
      method: POST
      timeout: 10s
      headers:
        Content-Type: application/json

  smtp:
    host: smtp.example.com
    port: 587
    tls: true
    username: user
    password: pass
    from: overwatch@example.com
    recipients:
      - oncall@example.com
```

See [`examples/overwatch.yaml`](./examples/overwatch.yaml) for a complete reference with all fields documented.

---

## Repository layout

```text
cmd/overwatch/     main entrypoint
internal/          CLI, server, checks, alerts, auth, scheduler
pkg/spec/          config and API types (stable public surface)
packaging/         Docker, systemd, launchd assets
examples/          example configs
```

---

## Contributing

Issues and pull requests are welcome. Read [`CONTRIBUTING.md`](./CONTRIBUTING.md) first.

---

## License

[MIT License](./LICENSE)
