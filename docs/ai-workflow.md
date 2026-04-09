# FleetDesk — AI Workflow

How we build FleetDesk using specialized AI agents, skills, and hooks.

## Model Strategy: Advisor Pattern

We use `opusplan` as the default model — Opus reasons during planning, Sonnet executes during implementation. This follows Anthropic's [Advisor Strategy](https://claude.com/blog/the-advisor-strategy): the smaller model drives execution, consulting the larger model only for hard decisions.

```
opusplan mode:
  Plan Mode (phases 1-4)  →  Opus    (architecture, design, trade-offs)
  Execution (phase 5+)    →  Sonnet  (code generation, file edits)

design-reviewer subagent  →  Opus    (called by Sonnet when design needs validation)
```

When a native advisor API becomes available in Claude Code CLI, it will replace the `opusplan` + subagent pattern with a built-in Sonnet-calls-Opus-for-guidance flow.

## Toolbox

| Name | Type | Model | When | Purpose |
|------|------|-------|------|---------|
| `/release-plan` | Skill | Sonnet | Human invokes at release start | Groom release file, propose sprints, create Linear items |
| `/feature-dev` | Plugin (Anthropic) | opusplan | Human invokes per FLE | 7-phase feature development with specialized agents |
| `code-explorer` | Agent (feature-dev) | Haiku | Phase 2 of feature-dev | Trace execution paths, find patterns |
| `code-architect` | Agent (feature-dev) | Opus | Phase 4 of feature-dev | Design approaches, compare trade-offs |
| `code-reviewer` | Agent (feature-dev) | Sonnet/Opus | Phase 6 of feature-dev | Review for bugs, quality, conventions |
| `design-reviewer` | Custom subagent | Opus | Between Phase 4 and 5 | Validate architecture — the "advisor" for design decisions |
| `/test-plan` | Skill | Sonnet | After implementation | Create test plan + test run in Linear |
| pre-flight | Hook | NEW | Automatic before git push | Build + test + lint + diff review |
| Claude GH | GitHub Action | Opus + /code-review | Automatic on PR | Multi-agent code review pipeline |

---

## Step 1: Release File

The human creates a release file — a wish list of what the next version should deliver. Just what you want and how important it is. The release-planner agent handles everything else (existing FLEs, dependencies, sprints, Linear creation).

**Who:** Human
**Output:** `releases/v0.X.Y.md`

### Format

```markdown
# v0.X.Y — Release Title

Short description of what this release delivers and why.

## Work Items

| Type | Priority | Title |
|------|----------|-------|
| Feature | High | Feature name |
| Bug | Urgent | Bug description |

## Notes

- Any context, risks, or concerns (optional)
```

**Priority** (Linear convention): Urgent (1), High (2), Medium (3), Low (4)
**Type** maps to Linear labels: Feature and Bug are labels on FLE issues, not separate entities.

### Example

```markdown
# v0.9.0 — K8s Actions & Multi-Backend

Extend K8s actions (deployment restart, scale) and add AWS EC2 as
a second backend to validate the closure-based engine.

## Work Items

| Type | Priority | Title |
|------|----------|-------|
| Feature | High | K8s deployment restart |
| Feature | High | K8s deployment scale |
| Feature | Medium | AWS EC2 fleet type |
| Feature | Low | Pod log JSON parser |
| Bug | High | AKS auto-detect stale closures |

## Notes

- AWS EC2 is large — may slip to v0.10.0
```

---

## Step 2: Release Grooming

The human invokes `/release-plan` to discuss the release file with Claude Code.

**Who:** Human + Claude Code (via `/release-plan` skill)
**Input:** `releases/v0.X.Y.md`
**Output:** `releases/v0.X.Y-plan.md` + Linear items created after approval

### What the skill does

1. **Reviews** the release file — challenges scope, flags risks, validates priorities
2. **Queries Linear** — checks existing FLEs, identifies blockers, what's already done
3. **Checks codebase** — rough complexity, readiness for proposed features
4. **Proposes sprints** — groups FLEs by dependency and priority
5. **Human validates** — adjusts sprints, priorities, scope
6. **Creates Linear items** (only after human says "go"):
   - Version project
   - Missing FLEs with descriptions and priorities
   - Dependencies (blocks/blocked by)

### Sprint grouping rules

A sprint is an ordered group of FLEs that:
- Have no blocking dependencies within the sprint
- Each produce a testable, mergeable increment
- Follow priority order (Urgent/High first)
- Each FLE = one feature branch = one PR (squash merge)

### Release plan format

```markdown
# v0.X.Y — Release Plan

Release file: releases/v0.X.Y.md
Linear project: v0.X.Y (id: xxx)
Created: YYYY-MM-DD

## Sprint 1 — Sprint Name

| Order | FLE | Title | Priority | Scope | Branch |
|-------|-----|-------|----------|-------|--------|
| 1 | FLE-xx | Feature name | High | small | feature/fle-xx |

## Sprint 2 — Sprint Name

| Order | FLE | Title | Priority | Scope | Branch |
|-------|-----|-------|----------|-------|--------|
| 2 | FLE-xx | Feature name | High | small | feature/fle-xx |
| 3 | FLE-xx | Feature name | High | small | feature/fle-xx |

## Decisions

- Key decisions made during grooming

## Linear

- Project created: v0.X.Y
- FLEs created: FLE-xx, FLE-xx
- Dependencies set: FLE-xx blocks FLE-xx
```

---

## Step 3: Feature Development

For each FLE, the human invokes `/feature-dev` — Anthropic's 7-phase workflow.

**Who:** Human + Claude Code (via `/feature-dev` plugin)
**Input:** FLE from the release plan
**Output:** Implemented feature with tests

### The 7 phases

| Phase | What | Agent | Human gate? |
|-------|------|-------|-------------|
| 1. Discovery | Clarify requirements | Claude Code | Yes — confirm understanding |
| 2. Explore | Trace code, find patterns | code-explorer (2-3x parallel) | No |
| 3. Clarify | Resolve ambiguities | Claude Code | Yes — answer questions |
| 4. Architect | Design multiple approaches | code-architect (2-3x parallel) | Yes — choose approach |
| 5. Implement | TDD: tests → code | general-purpose agents | No |
| 6. Review | Check quality | code-reviewer (3x parallel) | Yes — decide what to fix |
| 7. Summary | Document what was built | Claude Code | No |

### Design validation (between Phase 4 and 5)

For features that claim to be "generic" or touch 3+ resource types, the **design-reviewer** subagent validates the chosen architecture before implementation starts.

The subagent checks:
- Are abstractions real or just renames (switch-on-type)?
- Can a new backend be added without modifying shared code?
- Bubble Tea stale model capture in closures?
- Proper separation of concerns?

**Skip** for simple features, bug fixes, or single-file changes.

### Key rules

- **Never skip Phase 4 (Architect)** for non-trivial features
- One FLE per session — fresh context per feature
- `make check` before leaving Phase 5

---

## Step 4: User Testing

After implementation, the human invokes `/test-plan` to create structured test documentation.

**Who:** Human + Claude Code (via `/test-plan` skill)
**Input:** Implemented feature
**Output:** Test Plan + Test Run in Linear

### What the skill does

1. Reads the FLE description and implementation diff
2. Creates **Test Plan** (reusable template) in Linear:
   - Tests grouped by sequence
   - Preconditions for each test
   - Negative tests, parallel scenarios, edge cases
3. Human reviews the test plan
4. Creates **Test Run** (instance) in Linear
5. Human tests on real infrastructure
6. Claude Code updates the test run with results
7. Bugs found → fix on same branch → retest

### Rules

- No push before user validates on real infra
- Every action/UI feature needs a test plan
- Bugs found during testing are fixed on the same branch

---

## Step 5: Push + PR

The pre-flight hook runs automatically. Then push and create PR.

**Who:** Claude Code + pre-flight hook + Claude GH

### Pre-flight hook (automatic)

Runs before every `git push`:
- `make check` (build + test + lint)
- Flags: debug logs, dead code, secrets, scope creep

### PR creation

Claude Code pushes the feature branch and creates a PR.

### Claude GH review (automatic)

Triggered on PR open/sync. Uses Opus + `/code-review` skill:
1. Haiku — pre-flight checks
2. Sonnet x5 — parallel review (CLAUDE.md compliance, bugs, git history, past PRs, code comments)
3. Haiku — validate each issue (score 0-100, drop below 80)
4. Post findings

Architecture rules in CLAUDE.md are enforced:
- Engine must NOT switch on resource type
- Polymorphism via closures/interfaces
- Bubble Tea stale model capture detection

Claude GH can also be consulted via PR comments:
```
@claude What do you think about this design approach?
```

### Rules

- Address all review comments before merge
- One PR per FLE (squash merge)

---

## Step 6: Ship

Merge, tag, update Linear.

**Who:** Human + Claude Code

1. Merge PR (squash)
2. When all FLEs in the release plan are merged:
   - `git tag v0.X.Y && git push origin v0.X.Y`
3. Update Linear:
   - FLEs → Done
   - Close version project
4. Update memory with lessons learned

---

## Implementation Checklist

What we need to create:

| Item | Type | Location | Status |
|------|------|----------|--------|
| `/release-plan` | Skill | `.claude/skills/release-plan/SKILL.md` | Done |
| `design-reviewer` | Subagent | `.claude/agents/design-reviewer.md` | Done |
| `/test-plan` | Skill | `.claude/skills/test-plan/SKILL.md` | Done |
| pre-flight | Hook | `.claude/settings.json` | Done |
| Claude GH config | GitHub Action | `.github/workflows/claude-review.yml` | Done |
| feature-dev | Plugin | Anthropic built-in | Available |
| code-review | Plugin | Anthropic built-in | Available |

---

## Rules Summary

- One PR per FLE, one FLE per session
- Never push before user validates on real infra
- Never skip Phase 4 (Architect) for non-trivial features
- Generic = closures/interfaces, not switch on type
- Linear API via curl + GraphQL (not MCP)
- Document lessons in memory immediately when corrected
