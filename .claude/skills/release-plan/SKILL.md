---
name: release-plan
description: Groom a release file, propose sprints, and create Linear items. Use when the user wants to plan a new release version.
disable-model-invocation: true
---

# Release Planning

Groom a release file and create a release plan with sprints and Linear items.

## Input

The user provides a path to a release file: `$ARGUMENTS`

Read the file. It contains a version title, a goal description, and a table of work items (features and bugs) with priorities.

## Process

### 1. Review the release file

- Read the release file
- Summarize what you understand: the goal, the work items, priorities
- Flag any concerns: scope too large, unclear items, missing context

### 2. Check existing backlog

Query Linear for open FLEs:

```bash
curl -s -X POST https://api.linear.app/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: $LINEAR_API_KEY" \
  -d '{"query":"{ team(id: \"17a04ad2-7044-485f-b011-cf9ebeaa7eb2\") { issues(filter: { state: { name: { nin: [\"Done\", \"Canceled\"] } } }, orderBy: updatedAt) { nodes { identifier title priority state { name } } } } }"}' | jq -r '.data.team.issues.nodes[] | "\(.identifier)\t\(.state.name)\t\(.priority)\t\(.title)"'
```

Present the backlog to the user. Ask if any existing FLEs should be included in this release.

### 3. Check codebase complexity

For each work item, do a quick codebase scan:
- How many files would need to change?
- Are there existing patterns to reuse?
- Estimate scope: small (1-2 files), medium (3-5 files), large (6+ files or new package)

### 4. Propose sprints

Group work items into sprints:
- No blocking dependencies within a sprint
- Priority order (Urgent/High first)
- Each sprint produces a testable increment
- Each FLE = one feature branch = one PR

Present the sprint plan and discuss with the user. Iterate until agreed.

### 5. Create Linear items (only after user says "go")

Create the version project:

```bash
curl -s -X POST https://api.linear.app/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: $LINEAR_API_KEY" \
  -d '{"query":"mutation { projectCreate(input: { name: \"VERSION_NAME\", teamIds: [\"17a04ad2-7044-485f-b011-cf9ebeaa7eb2\"] }) { project { id name url } success } }"}' | jq '.'
```

Create each FLE:

```bash
curl -s -X POST https://api.linear.app/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: $LINEAR_API_KEY" \
  -d '{"query":"mutation { issueCreate(input: { teamId: \"17a04ad2-7044-485f-b011-cf9ebeaa7eb2\", title: \"TITLE\", description: \"DESCRIPTION\", projectId: \"PROJECT_ID\", priority: PRIORITY }) { issue { identifier url } success } }"}' | jq '.'
```

Set dependencies between FLEs where needed.

### 6. Save release plan

Write the release plan to `releases/VERSION-plan.md` with the agreed sprint structure, FLE identifiers, and decisions made.

## Rules

- NEVER create Linear items before the user explicitly approves
- Use curl + GraphQL for all Linear operations (not MCP)
- FleetDesk team ID: `17a04ad2-7044-485f-b011-cf9ebeaa7eb2`
- Linear API key is in `$LINEAR_API_KEY`
- Priority scale: Urgent (1), High (2), Medium (3), Low (4)
