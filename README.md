# Overwatch

**Infrastructure monitoring** with a terminal-first workflow: know when services, endpoints, certificates, and scheduled jobs fail—without living in a browser.

Overwatch ships as a **single Go binary**. **Self-hosted:** you run `overwatch serve` and own the YAML config on your machines. **Overwatch Cloud:** the same TUI and CLI connect to a hosted account so checks and alerts run without you operating the server.

---

## Features

| Capability | Description |
|------------|-------------|
| **Terminal UI** | Run `overwatch` for a live dashboard; first-time setup chooses self-hosted or cloud and stores client config under `~/.overwatch` (or the OS user profile equivalent). |
| **Automation CLI** | `overwatch check` and `overwatch alert` (`add`, `list`, `remove`, `update`) plus `overwatch status` for verbose, tabular config—ideal for scripts and automation. |
| **Self-hosted server** | `overwatch serve` runs the API and scheduler; bind address/port are first-class flags. Monitor definitions live in **YAML** as the **source of truth** (edits via TUI/CLI or on disk; reload via signal and on restart). |
| **Check types** | HTTP/HTTPS, TCP, TLS certificate expiry, DNS, and **scheduled-job check-in** (HTTP webhook your jobs `curl` on success, with “missed check-in” windows and optional failure signaling). |
| **Alerts** | Outbound **webhooks** and **SMTP** (your relay). |
| **Config** | `overwatch config init`, `overwatch config validate`, and `overwatch version`. |

Service supervision (systemd, Windows Service, Docker, etc.) is up to you—the binary does not install itself as a service.

See [`CONTRIBUTING.md`](./CONTRIBUTING.md) for how to contribute.

---

## Install

```bash
go install github.com/christianmscott/overwatch/cmd/overwatch@latest
```

Or clone and build:

```bash
git clone https://github.com/christianmscott/overwatch.git
cd overwatch
go build -o overwatch ./cmd/overwatch
```

Releases and Homebrew are intended as the project matures; see GitHub **Releases** when available.

---

## Quick start (self-hosted)

1. Start the server (creates starter config if missing):

   ```bash
   overwatch serve --bind-address 127.0.0.1 --bind-port 8080
   ```

2. In another terminal, launch the TUI or use the CLI:

   ```bash
   overwatch                 # TUI
   overwatch status          # full config snapshot
   overwatch check list
   ```

Exact flags and defaults may evolve; use `overwatch --help` and `overwatch serve --help` for the current interface.

---

## Repository layout

- `cmd/overwatch` — main entrypoint  
- `internal/` — implementation (CLI, TUI, server, checks, alerts, …)  
- `pkg/spec` — shared config and API types where stable  
- `packaging/` — examples (systemd, Docker, …)  

---

## Contributing

Issues and pull requests are welcome. Read [`CONTRIBUTING.md`](./CONTRIBUTING.md) first.

---

## License

[MIT License](./LICENSE)
