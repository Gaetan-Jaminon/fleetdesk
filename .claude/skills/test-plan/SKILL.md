---
name: test-plan
description: Create a test plan and test run in Linear for manual validation. Use after implementing a feature, before user testing.
disable-model-invocation: true
---

# Test Plan Creation

Create a structured test plan and test run in Linear for manual validation.

## Input

The user provides an FLE identifier or description: `$ARGUMENTS`

## Process

### 1. Understand the feature

- Read the FLE description from Linear (if identifier provided)
- Read the git diff on the current branch to understand what changed
- Identify what needs manual testing: actions, views, state transitions, edge cases

### 2. Design test cases

Act as a professional QA architect. For each test case provide:

| # | Test | Precondition | Expected |
|---|------|-------------|----------|

Follow these rules:
- Group tests by sequence (what order to run)
- Add preconditions (what state before each test)
- Include negative tests (state guards, invalid operations)
- Include parallel scenarios (multiple resources at once)
- Include edge cases (navigate away during action, cancel confirm)
- Remove tests that can't be manually validated
- Each sequence should flow naturally (e.g. start → restart → stop)

### 3. Present for review

Show the test plan to the user. Iterate until approved.

### 4. Create in Linear (only after user approves)

Create the Test Plan document (reusable template):

```bash
cat <<'PAYLOAD' > /tmp/linear-doc.json
{"query":"mutation($content: String!, $projectId: String!) { documentCreate(input: { title: \"TITLE Test Plan\", content: $content, projectId: $projectId }) { document { id url title } success } }","variables":{"projectId":"PROJECT_ID","content":"CONTENT"}}
PAYLOAD
curl -s -X POST https://api.linear.app/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: $LINEAR_API_KEY" \
  -d @/tmp/linear-doc.json | jq '.'
```

Create the Test Run document (instance with empty Result/Notes columns):

Same pattern, with title "TITLE Test Run — YYYY-MM-DD" and Result/Notes columns added to each table.

### 5. Report

Show the user the Linear document URLs.

## Rules

- NEVER create documents before the user explicitly approves the test plan
- Use curl + GraphQL for all Linear operations (not MCP)
- Use variables in the GraphQL mutation to handle newlines properly
- FleetDesk team ID: `17a04ad2-7044-485f-b011-cf9ebeaa7eb2`
- Linear API key is in `$LINEAR_API_KEY`
