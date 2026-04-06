# FleetDesk — Fleet Management TUI

## Overview

`fleetdesk` is a terminal UI application built in Go with Bubble Tea that provides a k9s-style navigable interface to manage a fleet of Linux VMs over SSH. It centralizes systemd service management and Podman container inspection across multiple hosts — no agents, no server, no infrastructure. Just a single binary that reads a host list, connects via SSH, and gives you a unified view of your fleet.

Think Cockpit, but as a TUI. Think k9s, but for VMs instead of Kubernetes. The layout follows the same pattern as mkx (github.com/Gaetan-Jaminon/mkx): list views with breadcrumb headers and keybind hint bars.

## Stack

- Language: Go
- TUI framework: Bubble Tea (github.com/charmbracelet/bubbletea)
- Styling: Lip Gloss (github.com/charmbracelet/lipgloss)
- SSH: golang.org/x/crypto/ssh + kevinburke/ssh_config
- Config: gopkg.in/yaml.v3
- Distribution: Single binary via `go build` + GoReleaser

## Design Decisions

- **Custom `tea.ExecCommand`** for terminal handover — not `tea.ExecProcess`. Lesson from mkx: custom ExecCommand owns stdin/stdout/stderr directly, which is required for proper terminal control when Alt Screen is active.
- **SSH library for data, `ssh` binary for interactive** — `golang.org/x/crypto/ssh` handles all data-gathering commands (probe, list units, list containers). For interactive sessions (logs, exec, inspect), we shell out to the local `ssh` binary via `tea.Exec` to get proper PTY allocation and inherit the user's SSH agent/config natively.
- **Each SSH auth method tried individually** — avoids `MaxAuthTries` exhaustion on servers with low limits. Keys are tried one at a time in separate connection attempts.
- **Service actions use `sudo systemctl`** via terminal handover — allows password prompt for sudo when needed. Actions show result confirmation + `systemctl status` output before returning.
- **`systemd_mode` defaults to `system`** — most target VMs (AAP, GitLab) run services at system level. `user` mode available as override for rootless setups.
- **Password never cached** — prompted once per session, used immediately for all hosts needing it, then zeroed from memory. Persistent SSH connections mean no re-prompting.
- **All dates in DD/MM/YYYY format** — European format consistent across all views.

## Core Concepts

### Host

A remote Linux VM accessible via SSH. Hosts are defined in a configuration file with minimal metadata — connection details only. Authentication is delegated entirely to the user's SSH setup.

### Resource Types

Six resource types available per host:

- **Services** — systemd units managed via `systemctl` (implemented)
- **Containers** — Podman containers managed via `podman` (implemented)
- **Cron Jobs** — scheduled tasks from `crontab` and `/etc/cron.d/` (read-only, planned v0.4.0)
- **Error Logs** — recent errors from `journalctl -p err` last 24h (planned v0.4.0)
- **Updates** — pending package updates from `dnf check-update` (read-only, planned v0.4.0)
- **Disk** — partition usage from `df` (read-only, planned v0.4.0)

### Navigation Model

Four-level drill-down, k9s-style:

```
Fleet Picker -> Host List -> Resource Type Picker -> Resource List
```

Every view uses the same layout pattern as mkx: bordered table with cursor navigation, breadcrumb header, and keybind hint bar at the bottom.

## SSH

### Authentication Resolution Order

FleetDesk does not manage credentials. It relies on the user's existing SSH configuration:

1. SSH agent — if `SSH_AUTH_SOCK` is set, try keys from the agent first
2. ~/.ssh/config — respect `Host` entries including `IdentityFile`, `User`, `Port`, `ProxyJump`
3. Default key paths — `~/.ssh/id_ed25519`, `~/.ssh/id_rsa`, `~/.ssh/id_ecdsa`
4. Password fallback — if all key methods fail, inline masked prompt in TUI; password reused for all hosts with same user, then cleared from memory

### Connection Management

- On fleet selection, FleetDesk opens one SSH connection per host in parallel using goroutines
- Connections are kept alive for the duration of the session (persistent)
- If a connection drops, the host status updates to `unreachable` and FleetDesk retries on manual refresh (`r`)
- Timeout: configurable per fleet defaults and per host, default 10 seconds

### Command Execution

All remote data-gathering commands are executed over the persistent SSH session via `golang.org/x/crypto/ssh`. FleetDesk never shells out to the local `ssh` binary for data collection.

For interactive terminal handover (logs, exec, inspect, status detail, shell, service actions with sudo), FleetDesk uses `tea.Exec` with a custom `ExecCommand` that runs the local `ssh` binary. This gives proper PTY allocation and inherits the user's SSH agent and config.

## Implemented Views

### View 1 — Fleet Picker

Columns: FLEET, TYPE, HOSTS

- Up/Down, j/k: Navigate fleet list
- Enter: Select fleet -> connect to hosts -> Host List
- e: Edit selected fleet file -> terminal handover to $EDITOR/$VISUAL/vi, reload on exit
- r: Reload — rescan config directory and reparse all fleet files
- q / Ctrl+C: Quit

### Host Probe

After selecting a fleet, FleetDesk displays the Host List immediately with all hosts in `connecting...` state, then probes all hosts in parallel. As each host responds, its row updates with live data. Hosts that fail with auth errors trigger an inline password prompt.

The probe gathers in a single SSH roundtrip: FQDN, uptime, OS name, service count (total/running/failed), container count (total/running), last update date (from dnf history), last security patch date.

### View 2 — Host List

Columns: HOST, OS, UP SINCE, SVC, CTN, LAST UPDATE, LAST SECURITY

Host groups are shown with visual separator rows (blue, bold). Hosts within groups are listed under their group header.

- Up/Down, j/k: Navigate host list
- Enter: Drill into selected host -> Resource Type Picker
- x: SSH shell -> terminal handover to `ssh -t user@host`
- r: Refresh — re-probe all hosts
- Esc: Back to Fleet Picker
- q / Ctrl+C: Quit

### View 3 — Resource Type Picker

Columns: RESOURCE, TOTAL, RUNNING, FAILED

Pre-fetches filtered services and containers on entry for accurate counts.

- Up/Down, j/k: Navigate
- Enter: Drill into selected resource type
- Esc: Back to Host List
- q / Ctrl+C: Quit

### View 4a — Service List

Columns: SERVICE, STATE, ENABLED, DESCRIPTION

Sorted by state: failed -> running -> exited -> waiting -> inactive, then alphabetically. Filtered by service_filter patterns if configured.

- Up/Down, j/k: Navigate
- s: Start selected service (sudo systemctl start, terminal handover)
- o: Stop selected service (sudo systemctl stop, terminal handover)
- t: Restart selected service (sudo systemctl restart, terminal handover)
- l: View logs -> terminal handover to `ssh -t host sudo journalctl -u unit -f`
- i: Show status detail -> terminal handover to `ssh -t host sudo systemctl status unit`
- r: Refresh service list
- Esc: Back to Resource Type Picker

### View 4b — Container List

Columns: CONTAINER, IMAGE, STATUS

Sorted by state: Up -> Exited -> others, then alphabetically.

- Up/Down, j/k: Navigate
- l: View logs -> terminal handover to `ssh -t host podman logs -f container`
- i: Inspect -> terminal handover to `ssh -t host podman inspect container | less`
- e: Exec shell -> terminal handover to `ssh -t host podman exec -it container /bin/bash`
- r: Refresh container list
- Esc: Back to Resource Type Picker

## Planned Views (v0.4.0)

### View 4c — Cron Jobs

Columns: SCHEDULE, COMMAND, SOURCE

Read-only view. Data fetched from user `crontab -l` and system `/etc/cron.d/*`. Source column indicates origin (crontab or /etc/cron.d).

Key bindings: navigate, Esc back, q quit.

### View 4d — Error Logs

Columns: TIME, UNIT, MESSAGE

Shows errors from last 24 hours via `journalctl -p err --since "24 hours ago"`.

Key bindings: l view full log (terminal handover), r refresh, Esc back, q quit.

### View 4e — Updates

Columns: PACKAGE, VERSION, TYPE

Read-only view. Data from `dnf check-update` + `dnf updateinfo list --security`. Security updates highlighted in red. Type column: security/bugfix/enhancement.

Key bindings: r refresh, Esc back, q quit.

### View 4f — Disk

Columns: FILESYSTEM, SIZE, USED, AVAIL, USE%, MOUNT

Read-only view. Data from `df -h`. Partitions above 90% in red, above 80% in yellow.

Key bindings: r refresh, Esc back, q quit.

## Fleet Configuration

### Config Directory

`~/.config/fleetdesk/` (XDG compliant). Each `.yaml` file in this directory represents a fleet.

### Global Configuration

`~/.config/fleetdesk/config.yaml` holds app-wide settings:

```yaml
# Editor for fleet file editing (e key)
# Resolution order: this setting -> $EDITOR -> $VISUAL -> vi
editor: nvim
```

### Startup Flow

1. Scan `~/.config/fleetdesk/` for `.yaml` files (excluding `config.yaml`)
2. If only one fleet file exists, skip the picker and load it directly
3. If multiple fleet files exist, display the Fleet Picker

### Fleet File Format

```yaml
name: AAP Production
type: vm

defaults:
  user: aap
  port: 22
  timeout: 10s
  systemd_mode: system
  service_filter:
    - "automation-*"
    - "postgresql*"

groups:
  - name: Control Plane
    service_filter:        # group-level override
      - "automation-*"
    hosts:
      - name: aap-ctrl-01
        hostname: aap-ctrl-01.fluxys.net

      - name: aap-ctrl-02
        hostname: aap-ctrl-02.fluxys.net

  - name: Hub
    hosts:
      - name: aap-hub-01
        hostname: aap-hub-01.fluxys.net
        user: hub-admin
        service_filter:    # host-level override
          - "pulpcore*"
          - "nginx*"

hosts:
  - name: monitoring-01
    hostname: monitoring-01.fluxys.net
```

### Fleet-Level Fields

- `name` (optional, default: filename) — Display name in Fleet Picker
- `type` (optional, default: vm) — Fleet type. `vm` supported, `k8s` reserved for future

### Host Fields

- `name` (required) — Display name in the TUI
- `hostname` (required) — FQDN or IP for SSH connection
- `user` (optional, from defaults or ~/.ssh/config) — SSH user
- `port` (optional, default: 22) — SSH port
- `timeout` (optional, default: 10s) — Connection timeout
- `systemd_mode` (optional, default: system) — `system` or `user` for systemctl scope
- `service_filter` (optional, from group or defaults) — Glob patterns for service filtering

Service filter inheritance: host -> group -> defaults. Most specific wins.

### Ansible Inventory Import (future)

Not implemented. Future enhancement: `fleetdesk --inventory /path/to/inventory.yml`.

## Release History

### v0.1.0 — MVP

Core functionality: config parsing, SSH connections, 5 views (fleet picker, host list, resource picker, service list, container list), service actions (start/stop/restart with sudo terminal handover), container actions (logs/inspect/exec), edit fleet file, host probe, CI/CD with GoReleaser.

### v0.2.0 — Config & Polish

Service filter patterns (glob, per-fleet defaults), host groups with visual separators, sort services and containers by state (failed first).

### v0.3.0 — Auth & Resources

Per-host/group service filter inheritance, password fallback prompt (masked inline, session-scoped, auto-cleared from memory), resource picker redesigned as bordered table with TOTAL/RUNNING/FAILED columns, expanded probe (service running/failed counts, container running/total, last update/security dates from dnf history).

### v0.3.1

SSH shell into host via `x` key in Host List.

### v0.3.2

README with 5 screenshots, CI/release/Go/MIT badges, group header color fix.

### v0.4.0 — Extended Resources (planned)

Four new read-only resource views:
- Cron Jobs (crontab + /etc/cron.d)
- Error Logs (journalctl -p err, last 24h)
- Updates (dnf check-update + updateinfo)
- Disk (df -h with usage highlighting)

Extended host probe to include cron/error/update/disk counts in resource picker.

## Non-Goals

- Kubernetes fleet support (`type: k8s` — field reserved, not implemented)
- Ansible inventory import
- Multi-host parallel actions (e.g., restart a service on all hosts)
- Container start/stop/rm actions (inspect-only; manage via systemd)
- Persistent history or audit log
- Notifications or alerting
- Config file hot-reload
- Configurable key bindings

## CI/CD

### CI — `.github/workflows/ci.yml`

Runs on every push and pull request to `main`.

Steps: checkout, setup Go, test (`go test ./... -race`), build.
Matrix: `ubuntu-latest` + `macos-latest`. No lint (golangci-lint Go 1.25 compat issue).

### CD — `.github/workflows/release.yml`

Runs on tag push matching `v*`. Uses GoReleaser to cross-compile for Linux/macOS (amd64/arm64), generate checksums, create GitHub Release.

### Versioning

Semantic versioning. Tag format: `v{major}.{minor}.{patch}`.
Binary embeds version info via ldflags, accessible via `fleetdesk --version`.
Single static binary with zero runtime dependencies.
