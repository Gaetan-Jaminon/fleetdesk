# FleetDesk — Project Instructions

## What is this

Go TUI application (Bubble Tea) managing fleets of Linux VMs (SSH), Azure resources (az CLI), and Kubernetes clusters (kubectl).

## Workflow

Follow the AI Workflow: [docs/ai-workflow.md](docs/ai-workflow.md)

- Use `/release-plan` to plan releases
- Use `/feature-dev` for feature development (7-phase workflow)
- Use `/test-plan` to create test plans before user testing
- Use the `design-reviewer` agent before implementing architectural features
- Pre-flight hook runs automatically before `git push`

## Architecture

- Package structure: `internal/config/`, `internal/ssh/`, `internal/azure/`, `internal/k8s/`, `internal/app/`
- Backend packages (ssh/, azure/, k8s/) must NOT import Bubble Tea — pure data-fetching and parsing only
- Each view: fetch function -> message type -> model handler -> render function
- Destructive actions require [Y/n] confirmation prompts

## Action Engine

Generic transition system with poll/oneshot strategies using closures.

- Engine must NOT switch on resource type — use closures set by the caller for execution, polling, refresh, and state detection
- If a new backend requires editing the engine core, the abstraction is broken
- Bubble Tea Model is a value type — closures that mutate model state capture a stale snapshot

## Build & Test

```bash
make check    # build + test + lint (required before PR)
make build    # build binary
make test     # run unit tests
```

## Git

- One PR per FLE (squash merge)
- Branch: `feature/fle-xx-description`
- Conventional commits: feat/fix/chore/refactor
- Never push before user validates on real infra

## Linear

- API: curl + GraphQL (not MCP — saves tokens)
- API key: `$LINEAR_API_KEY`
- Team ID: `17a04ad2-7044-485f-b011-cf9ebeaa7eb2`
- Priority: Urgent (1), High (2), Medium (3), Low (4)
