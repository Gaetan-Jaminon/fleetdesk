# FleetDesk

Fleet management TUI — manage Linux VMs over SSH with k9s-style navigation.

## Overview

`fleetdesk` is a terminal UI application that provides a unified view of your Linux VM fleet.
Connect via SSH, browse systemd services and Podman containers, and manage them interactively.

No agents, no server — just a single binary that reads a host list and connects via SSH.

## Features

- k9s-style navigation: Fleet → Host → Services/Containers
- Systemd service management: start, stop, restart, logs, status
- Podman container inspection: logs, inspect, exec
- Parallel SSH connections with async status updates
- Terminal handover for interactive commands
- Fleet configuration via YAML files

## Install

```bash
go install github.com/Gaetan-Jaminon/fleetdesk@latest
```

Or download a binary from [Releases](https://github.com/Gaetan-Jaminon/fleetdesk/releases).

## Configuration

Fleet files live in `~/.config/fleetdesk/`:

```yaml
name: My Fleet

defaults:
  user: ansible
  timeout: 10s
  systemd_mode: system

hosts:
  - name: server-01
    hostname: server-01.example.com
  - name: server-02
    hostname: server-02.example.com
```

## Key Bindings

- `Enter` — drill in / select
- `Esc` — go back
- `j/k` or `↑/↓` — navigate
- `r` — refresh
- `q` — quit

Service actions: `s` start, `o` stop, `t` restart, `l` logs, `i` status
Container actions: `l` logs, `i` inspect, `e` exec

## License

MIT
