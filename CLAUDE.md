# FleetDesk — Project Instructions

## What is this

Go TUI application (Bubble Tea) managing fleets of Linux VMs (SSH), Azure resources (az CLI), and Kubernetes clusters (kubectl).

## Workflow

Source of truth: [AI Development Baseline — Step-by-Step Guide](https://linear.app/fleetdesk/document/ai-development-baseline-step-by-step-guide-40d44987b7b6)

| Step | When | Who | Output |
|------|------|-----|--------|
| 0 — Product discovery | Once per product | Claude Web + You | Product Brief |
| 1 — Plan & spec | Once per release | Claude Web + You | Release plan + Linear issues |
| 2 — Architecture | Per issue | CC Opus (Plan Mode) | Linear comment contract + ADR |
| 3 — Implementation | Per issue | CC Sonnet | Feature + tests + PR |
| 4 — Integration testing | Per issue | You | Test Run results |
| 5 — Pre-flight | Per push (auto) | Hook | Pass / fail |
| 6 — PR + merge | Per PR | Claude GitHub + CC | Reviewed + merged PR |
| 7 — Ship | Once per release | You + CC | Tagged release |

Key rules:
- Step 1 uses the Release Plan Template in Linear — Claude Web handles it. `/release-plan` is optional for CC-side grooming only.
- Step 2 posts the enriched spec + design + acceptance tests as a comment on the Linear issue — this is the contract between architect and developer. Claude GitHub checks the PR against it.
- Step 3: CC reads the Step 2 contract, writes failing tests first (TDD), then implements. Tests are committed before implementation.
- Step 6: CC presents review summary. **CC never merges without explicit human approval.**
- Use `/feature-dev` for Steps 2-3, `design-reviewer` subagent for architecture validation in Step 2.
- Use `/test-plan` to create Test Plan and Test Run documents in Linear (Step 2).
- Pre-flight hook runs automatically before every `git push` (Step 5).

### Post-implementation workflow

After implementation is complete, NEVER suggest manual testing directly. Follow the steps in order:

1. Push branch and create PR (`git push` + `gh pr create`)
2. Step 4 (pre-flight) runs automatically on push
3. Wait for Claude GitHub review (Step 5) — fix findings, loop until clean
4. Only THEN does the human do integration testing (Step 6)

Do not skip or reorder these steps. The human tests on reviewed code, not raw implementation.

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

### Testing tiers

| Level | What | How | CI? |
|-------|------|-----|-----|
| Unit | Parsers, formatters, config, helpers | `go test` | Yes |
| UI | Navigation, key bindings, state transitions, view rendering, flash messages, overlays | `go test` + `teatest` | Yes |
| Integration | Real infra — SSH, Azure, K8s | Manual on real hosts | No |

TDD: Opus (Step 2) designs *what to test* across all tiers. Sonnet (Step 3) translates unit + UI tests into Go test code. Integration tests go into the Test Plan/Run in Linear for Step 4.

## Security

- No hardcoded credentials, keys, or secrets — use environment variables
- Sanitize user input before shell execution (SSH commands, kubectl, az CLI)
- No secrets in error messages, log output, or debug logs
- Passwords: never cache, log, or persist beyond the session
- SSH auth: each method tried individually to avoid MaxAuthTries exhaustion
- API keys: from `$LINEAR_API_KEY`, `$ANTHROPIC_API_KEY` — never in code
- Proactively flag security implications during design review, even if the plan doesn't mention them

## Knowledge

- When corrected, update memory immediately — don't wait for end of session
- When making an architectural decision, draft an ADR in docs/adr/
- When a new convention or rule emerges, propose adding it to CLAUDE.md

## Git

- One PR per FLE (squash merge)
- Branch: `feature/fle-xx-description`
- Conventional commits: feat/fix/chore/refactor

## Linear

- API: curl + GraphQL (not MCP — saves tokens)
- API key: `$LINEAR_API_KEY`
- Team ID: `17a04ad2-7044-485f-b011-cf9ebeaa7eb2`
- Priority: Urgent (1), High (2), Medium (3), Low (4)
