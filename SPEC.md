# FleetDesk — Fleet Management TUI

## Overview

`fleetdesk` is a terminal UI application built in Go with Bubble Tea that provides a k9s-style navigable interface to manage a fleet of Linux VMs over SSH. It centralizes systemd service management and Podman container inspection across multiple hosts — no agents, no server, no infrastructure. Just a single binary that reads a host list, connects via SSH, and gives you a unified view of your fleet.

Think Cockpit, but as a TUI. Think k9s, but for VMs instead of Kubernetes. The layout follows the same pattern as mkx (github.com/Gaetan-Jaminon/mkx): list views with breadcrumb headers and keybind hint bars.

## Stack

- Language: Go
- TUI framework: Bubble Tea (github.com/charmbracelet/bubbletea)
- Styling: Lip Gloss (github.com/charmbracelet/lipgloss)
- SSH: golang.org/x/crypto/ssh + kevinburke/ssh_config
- Distribution: Single binary via `go build`

## Design Decisions

- **Custom `tea.ExecCommand`** for terminal handover — not `tea.ExecProcess`. Lesson from mkx: custom ExecCommand owns stdin/stdout/stderr directly, which is required for proper terminal control when Alt Screen is active.
- **SSH library for data, `ssh` binary for interactive** — `golang.org/x/crypto/ssh` handles all data-gathering commands (probe, list units, list containers). For interactive sessions (logs, exec, inspect), we shell out to the local `ssh` binary via `tea.Exec` to get proper PTY allocation and inherit the user's SSH agent/config natively.
- **`systemd_mode` defaults to `system`** — most target VMs (AAP, GitLab) run services at system level. `user` mode available as override for rootless setups.

## Core Concepts

### Host

A remote Linux VM accessible via SSH. Hosts are defined in a configuration file with minimal metadata — connection details only. Authentication is delegated entirely to the user's SSH setup.

### Resource Types

Two resource types are available per host:

- **Services** — systemd units managed via `systemctl`
- **Containers** — Podman containers managed via `podman`

### Navigation Model

Four-level drill-down, k9s-style:

```
Fleet Picker -> Host List -> Resource Type Picker -> Resource List (services or containers)
```

Every view uses the same layout pattern as mkx: list with cursor navigation, breadcrumb header, and keybind hint bar at the bottom.

## SSH

### Authentication Resolution Order

FleetDesk does not manage credentials. It relies on the user's existing SSH configuration:

1. SSH agent — if `SSH_AUTH_SOCK` is set, try keys from the agent first
2. ~/.ssh/config — respect `Host` entries including `IdentityFile`, `User`, `Port`, `ProxyJump`
3. Default key paths — `~/.ssh/id_ed25519`, `~/.ssh/id_rsa`, `~/.ssh/id_ecdsa`
4. Password fallback — if all key methods fail, prompt the user for a password via an inline TUI input field (masked) (v0.3.0)

### Connection Management

- On startup, FleetDesk opens one SSH connection per host in parallel using goroutines
- Connections are kept alive for the duration of the session (persistent)
- If a connection drops, the host status updates to `unreachable` and FleetDesk retries on manual refresh (`r`)
- Timeout: configurable, default 10 seconds per host

### Command Execution

All remote data-gathering commands are executed over the persistent SSH session via `golang.org/x/crypto/ssh`. FleetDesk never shells out to the local `ssh` binary for data collection.

For interactive terminal handover (logs, exec, inspect, status detail), FleetDesk uses `tea.Exec` with a custom `ExecCommand` that runs the local `ssh` binary. This gives proper PTY allocation and inherits the user's SSH agent and config.

## UX Flow

### Startup

1. Scan `~/.config/fleetdesk/` for `.yaml` fleet files
2. If only one fleet file exists, skip the picker and load it directly
3. If multiple fleet files exist, display the Fleet Picker

### View 1 — Fleet Picker

```
+------------------------------------------------------------------+
|  fleetdesk                                                       |
|                                                                  |
|  FLEET                      TYPE     HOSTS                       |
|  --------------------------------------------------------        |
|  > AAP Production           vm       6                           |
|    AAP Staging              vm       4                           |
|    GitLab                   vm       2                           |
|    Monitoring               vm       3                           |
|                                                                  |
|  --------------------------------------------------------        |
|  Enter select   e edit   r reload   q quit                       |
+------------------------------------------------------------------+
```

- Up/Down, j/k: Navigate fleet list
- Enter: Select fleet -> connect to hosts -> Host List
- e: Edit selected fleet file -> terminal handover to configured editor, reload on exit
- r: Reload — rescan config directory and reparse all fleet files
- q / Ctrl+C: Quit

Each entry shows the fleet name (from `name` field in YAML, or filename) and the number of hosts defined.

### Host Probe

After selecting a fleet, FleetDesk displays the Host List immediately with all hosts in `connecting...` state, then probes all hosts in parallel (one goroutine per host). As each host responds, its row updates with live data. Hosts that fail show `unreachable` with the error reason.

On connection, FleetDesk runs a single SSH command to gather host info in one roundtrip:

```bash
hostname -f && uptime -s && cat /etc/os-release | grep PRETTY_NAME | cut -d= -f2 | tr -d '"' && systemctl list-units --type=service --no-pager -q | wc -l && podman ps -q 2>/dev/null | wc -l
```

This returns: FQDN, up since date, OS name, service count, container count.

The probe uses the configured `systemd_mode` (default: `system`) for the systemctl call. If the command fails, it tries the opposite mode and stores which mode works for subsequent commands on that host.

### View 2 — Host List

```
+------------------------------------------------------------------+
|  fleetdesk > AAP Production                                      |
|                                                                  |
|  HOST                  OS              UP SINCE     SVC    CTN   |
|  --------------------------------------------------------        |
|  > aap-ctrl-01         RHEL 9.4        2024-03-12   18     22   |
|    aap-hub-01          RHEL 9.4        2024-03-12   6      8    |
|    aap-eda-01          RHEL 9.4        2024-03-15   12     14   |
|    gitlab-01           RHEL 9.4        2024-02-28   4      3    |
|    * monitoring-01     connecting...                             |
|    x old-server-03     unreachable (timeout)                     |
|                                                                  |
|  --------------------------------------------------------        |
|  Enter drill in   r refresh   Esc back   q quit                  |
+------------------------------------------------------------------+
```

- Up/Down, j/k: Navigate host list
- Enter: Drill into selected host -> Resource Type Picker
- r: Refresh — re-probe all hosts (or selected host)
- Esc: Back to Fleet Picker
- q / Ctrl+C: Quit

Status indicators:
- No icon: connected, healthy
- `*`: connecting / probing
- `x`: unreachable, with reason (timeout, auth failed, connection refused)

Only online hosts can be drilled into. Attempting to Enter on an unreachable host shows a flash message.

### View 3 — Resource Type Picker

Minimal view after selecting a host. Two options.

```
+------------------------------------------------------------------+
|  fleetdesk > AAP Production > aap-ctrl-01                        |
|                                                                  |
|  > Services              18 units                                |
|    Containers            22 running                              |
|                                                                  |
|  --------------------------------------------------------        |
|  Enter select   Esc back   q quit                                |
+------------------------------------------------------------------+
```

- Up/Down, j/k: Navigate
- Enter: Drill into selected resource type
- Esc: Back to Host List
- q / Ctrl+C: Quit

### View 4a — Service List

```
+------------------------------------------------------------------+
|  fleetdesk > AAP Production > aap-ctrl-01 > Services             |
|                                                                  |
|  SERVICE                          STATE       ENABLED            |
|  --------------------------------------------------------        |
|  > automation-controller-web      active      enabled            |
|    automation-controller-task     active      enabled            |
|    automation-controller-rsyslog  active      enabled            |
|    automation-eda-api             active      enabled            |
|    automation-eda-daphne          active      enabled            |
|    postgresql                     active      enabled            |
|    redis                          active      enabled            |
|    receptor                       active      enabled            |
|    x automation-hub-worker-2      failed      enabled            |
|                                                                  |
|  --------------------------------------------------------        |
|  s start  o stop  t restart  l logs  i status  Esc back          |
+------------------------------------------------------------------+
```

- Up/Down, j/k: Navigate
- s: Start selected service (`systemctl start <unit>`)
- o: Stop selected service (`systemctl stop <unit>`)
- t: Restart selected service (`systemctl restart <unit>`)
- l: View logs -> terminal handover to `ssh <host> journalctl -u <unit> -f`
- i: Show unit status detail (`systemctl status <unit>`) -> terminal handover
- Esc: Back to Resource Type Picker
- q / Ctrl+C: Quit

State indicators:
- `active` shown in green
- `inactive` in gray
- `failed` in red with x prefix

Data is fetched via:
```bash
systemctl list-units --type=service --all --no-pager --plain --no-legend
```

For `--user` mode hosts, all commands use `systemctl --user` and `journalctl --user-unit`.

### View 4b — Container List

```
+------------------------------------------------------------------+
|  fleetdesk > AAP Production > aap-ctrl-01 > Containers           |
|                                                                  |
|  CONTAINER                         IMAGE              STATUS     |
|  --------------------------------------------------------        |
|  > automation-controller-web       aap-controller     running    |
|    automation-controller-task      aap-controller     running    |
|    automation-eda-api              aap-eda            running    |
|    automation-eda-daphne           aap-eda            running    |
|    postgresql                      postgresql-13      running    |
|    redis                           redis-6            running    |
|    x automation-hub-worker-2       aap-hub            exited     |
|                                                                  |
|  --------------------------------------------------------        |
|  l logs  i inspect  e exec  Esc back                             |
+------------------------------------------------------------------+
```

- Up/Down, j/k: Navigate
- l: View logs -> terminal handover to `ssh <host> podman logs -f <container>`
- i: Inspect -> terminal handover to `ssh <host> podman inspect <container> | less`
- e: Exec shell -> terminal handover to `ssh -t <host> podman exec -it <container> /bin/bash`
- Esc: Back to Resource Type Picker
- q / Ctrl+C: Quit

Status indicators:
- `running` in green
- `exited` in gray
- `exited` with non-zero code in red with x prefix

Data is fetched via:
```bash
podman ps -a --format "{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.ID}}"
```

### Terminal Handover

For logs (`l`), inspect (`i`), exec (`e`), and status detail (`i` on services), FleetDesk uses `tea.Exec` with a custom `ExecCommand` implementation that shells out to the local `ssh` binary:

```go
cmd := exec.Command("ssh", "-t", host, "journalctl", "-u", unit, "-f")
```

The custom `ExecCommand` (not `tea.ExecProcess`) owns stdin/stdout/stderr directly, which is required when Bubble Tea runs in Alt Screen mode. This is the same pattern used in mkx — `tea.ExecProcess` does not work correctly because the subprocess output is not visible while the TUI holds the alternate screen buffer.

This reuses the user's SSH config and agent natively. The TUI disappears, the user interacts with the remote command directly, and when they exit (Ctrl+C or `q` in less), FleetDesk reclaims the terminal.

**Important**: for interactive commands like `podman exec`, the `-t` flag on `ssh` is essential to allocate a PTY.

## Fleet Configuration

### Config Directory

`~/.config/fleetdesk/` (XDG compliant). Each `.yaml` file in this directory represents a fleet.

```
~/.config/fleetdesk/
  config.yaml              <- global settings
  aap-production.yaml      <- fleet file
  aap-staging.yaml
  gitlab.yaml
  monitoring.yaml
```

### Global Configuration

`~/.config/fleetdesk/config.yaml` holds app-wide settings:

```yaml
# Editor for fleet file editing (e key)
# Resolution order: this setting -> $EDITOR -> $VISUAL -> vi
editor: nvim
```

Key bindings configuration is planned for v0.3.0.

### Startup Flow

1. Scan `~/.config/fleetdesk/` for `.yaml` files (excluding `config.yaml`)
2. If only one fleet file exists, skip the picker and load it directly
3. If multiple fleet files exist, display the Fleet Picker

### Fleet File Format

Each fleet file contains the hosts for that fleet:

```yaml
# ~/.config/fleetdesk/aap-production.yaml

# Optional display name (defaults to filename without extension)
name: AAP Production

# Fleet type: "vm" (default) or "k8s" (future)
type: vm

# Global defaults (can be overridden per host)
defaults:
  user: aap
  port: 22
  timeout: 10s
  systemd_mode: system    # "system" (default) or "user" for rootless setups

# Host groups
groups:
  - name: Control Plane
    hosts:
      - name: aap-ctrl-01
        hostname: aap-ctrl-01.fluxys.net

      - name: aap-ctrl-02
        hostname: aap-ctrl-02.fluxys.net

  - name: Hub
    hosts:
      - name: aap-hub-01
        hostname: aap-hub-01.fluxys.net
        user: hub-admin          # override default user

  - name: GitLab
    hosts:
      - name: gitlab-01
        hostname: gitlab-01.fluxys.net

# Ungrouped hosts also supported
hosts:
  - name: monitoring-01
    hostname: monitoring-01.fluxys.net
```

### Fleet-Level Fields

- `name` (optional, default: filename) — Display name in Fleet Picker
- `type` (optional, default: vm) — Fleet type. `vm` supported now, `k8s` reserved for future

For MVP, only `type: vm` is functional. If `type: k8s` is specified, FleetDesk displays the fleet in the picker but shows "K8s support coming soon" on Enter.

### Host Fields

- `name` (required) — Display name in the TUI
- `hostname` (required) — FQDN or IP for SSH connection
- `user` (optional, from defaults or ~/.ssh/config) — SSH user
- `port` (optional, default: 22) — SSH port
- `timeout` (optional, default: 10s) — Connection timeout
- `systemd_mode` (optional, default: system) — `system` or `user` for systemctl scope

Groups are purely for visual organization in the Host List — they add a group header row.

### Ansible Inventory Import (future)

Not in MVP. Future enhancement: `fleetdesk --inventory /path/to/inventory.yml` to import hosts from a standard Ansible inventory file.

## Service Filtering (v0.2.0)

By default, FleetDesk shows all systemd services on a host. This can be noisy. The config file supports optional filter patterns per group or host:

```yaml
defaults:
  service_filter:
    - "automation-*"
    - "postgresql*"
    - "redis*"
    - "receptor*"
```

If `service_filter` is set, only matching services are shown. If not set, all services are shown.

## Key Bindings Summary

Default key bindings (configurable via config file in v0.3.0):

- Fleet Picker: Up/Down/j/k navigate, Enter select, e edit, r reload, q quit
- Host List: Up/Down/j/k navigate, Enter drill in, r refresh, Esc back, q quit
- Resource Type Picker: Up/Down/j/k navigate, Enter select, Esc back, q quit
- Service List: Up/Down/j/k navigate, s start, o stop, t restart, l logs, i status, Esc back, q quit
- Container List: Up/Down/j/k navigate, l logs, i inspect, e exec, Esc back, q quit
- Terminal handover: all keys passed to SSH subprocess

## Non-Goals (MVP)

- Kubernetes fleet support (`type: k8s` — field reserved, not implemented)
- Ansible inventory import
- Multi-host parallel actions (e.g., restart a service on all hosts)
- Container start/stop/rm actions (MVP is inspect-only for containers; manage via systemd)
- Resource usage dashboards (CPU, memory, disk)
- Persistent history or audit log
- Notifications or alerting
- Config file hot-reload
- Configurable key bindings (v0.3.0)
- Password fallback prompt (v0.3.0)

## Milestones

### v0.1.0 — MVP

Core functionality: connect, navigate, view, act. CI/CD from day one.

1. Project scaffolding (go mod, main, Bubble Tea)
2. CI workflow: build matrix (ubuntu + macos)
3. Fleet config directory scanner
4. Fleet file parser (per-fleet YAML)
5. Fleet Picker view
6. SSH connection manager (parallel connect, keep-alive)
7. Host probe (single-command info gathering)
8. Host List view with async status updates
9. Resource Type Picker view
10. Service List view (systemctl list-units)
11. Service actions: start, stop, restart
12. Container List view (podman ps)
13. Terminal handover for logs, inspect, exec, status
14. Keybind hint bar (k9s-style bottom bar)
15. Edit fleet file via editor handover (e key)
16. Reload config directory (r key on Fleet Picker)
17. Basic error handling (unreachable, auth failed, timeout)
18. --version flag with embedded build info
19. GoReleaser config + CD workflow
20. README.md

### v0.2.0 — Config & Polish

21. Service filter patterns from config
22. Host groups with visual separators
23. Lip Gloss styling pass (colors, status indicators)
24. Responsive layout on terminal resize

### v0.3.0 — Config & Keybinds

25. Configurable key bindings
26. Per-host systemd_mode override
27. Per-host/group service_filter
28. Connection timeout configuration
29. Password fallback prompt in TUI

## CI/CD

### CI — `.github/workflows/ci.yml`

Runs on every push and pull request to `main`.

Steps:
1. Checkout code
2. Setup Go
3. Test — `go test ./... -race -coverprofile=coverage.out`
4. Build — `go build -o fleetdesk .`

Matrix: `ubuntu-latest` + `macos-latest`. No lint (golangci-lint Go 1.25 compatibility issue, same as mkx).

### CD — `.github/workflows/release.yml`

Runs on tag push matching `v*`.

Uses GoReleaser to:
1. Cross-compile for Linux (amd64, arm64), macOS (amd64, arm64)
2. Generate checksums
3. Create a GitHub Release with all binaries attached

### Versioning

Semantic versioning. Tag format: `v{major}.{minor}.{patch}`.
Binary embeds version info via ldflags, accessible via `fleetdesk --version`.
