---
name: jira-cli
description: Use the jira CLI to manage Jira issues create task, bug, story, or comment on an issue.
---

You are operating the `jira` CLI to interact with Jira Cloud. Use the reference below to execute the task: $ARGUMENTS

---

## Global Flags

Prepend these to any command:

- `--profile <name>` — override the active profile for this invocation
- `--json` — return structured JSON (use when you need to extract data with `jq`)

Priority order for configuration: CLI flag > `JIRA_*` env vars > `~/.jira-cli/config.yaml` > command defaults.

---

## Commands

### Read an issue (summary + description)

```
jira issue view <ISSUE-KEY>
```

- Prints the issue key, summary, status, and full description.
- Use `--json` to get structured output with `description`, `summary`, `status`, and `issue_key`.

### Create an issue

```
jira issue create --summary "<title>" [--description "<body>"] [--project <KEY>] [--type <Bug|Story|Task|...>] [--assign-me] [--assignee <account-id>]
```

- `--summary` is required; all other flags fall back to the active profile's defaults.
- `--assign-me` automatically assigns the issue to you (resolves your account ID from the profile email via the Jira API).
- `--assignee <account-id>` assigns to a specific account ID (use when you know the exact ID).
- Extract the created issue key from JSON when you need it in follow-up commands:
  ```bash
  KEY=$(jira --json issue create --summary "..." --assign-me | jq -r '.data.issue_key')
  ```

### Create a subtask under a parent issue

```
jira issue subtask <PARENT-KEY> --summary "<title>" [--description "<body>"] [--assign-me] [--assignee <account-id>]
```

- Creates a subtask of type `Subtask` linked to the parent issue.
- The project is inferred from the parent key — no `--project` flag needed.
- `--assign-me` and `--assignee` work the same as in `create`.
- Use `--json` to extract the new issue key:
  ```bash
  KEY=$(jira --json issue subtask SED-29 --summary "..." --assign-me | jq -r '.data.issue_key')
  ```

### Assign an issue

```
jira issue assign <ISSUE-KEY> --assign-me
jira issue assign <ISSUE-KEY> --assignee <account-id>
```

- `--assign-me` resolves your account ID from the profile email via the Jira API.
- One of `--assign-me` or `--assignee` is required.

### Link two issues

```
jira issue link <ISSUE-KEY> <TARGET-KEY> [--type <link-type>]
```

- `--type` defaults to `"Relates"`. Common values: `"Blocks"`, `"Clones"`, `"Relates"`.
- The direction matters: `<ISSUE-KEY>` is the inward issue, `<TARGET-KEY>` is the outward issue.
- Example: `jira issue link SED-10 SED-5 --type Blocks` means SED-10 blocks SED-5.

### Update an issue's description

```
jira issue describe <ISSUE-KEY> --description "<new description>"
```

### Transition an issue to a new status

```
jira issue transition <ISSUE-KEY> --status "<status name>"
```

Common statuses: `"To Do"`, `"In Progress"`, `"In Review"`, `"Done"`.

### List comments on an issue

```
jira issue comments <ISSUE-KEY>
```

- Prints each comment with author name, date, and body text (ADF rendered as plain text).
- Use `--json` to get a structured `{ "comments": [...] }` array with `id`, `author`, `created`, and `body` fields.

### Add a comment

```
jira issue comment <ISSUE-KEY> --body "<comment text>"
```

---

## IMPORTANT: Always Write Descriptions and Comments in Markdown

Whenever you create or update an issue description (`--description`) or add a comment (`--body`), the text **MUST** be written in Markdown. Never send plain prose — always use headings, lists, bold/italic, and code blocks to structure the content.

---

## Markdown Formatting in Body / Description

The CLI converts Markdown to Atlassian Document Format (ADF) automatically. Supported syntax:

| Markdown | Result in Jira |
|---|---|
| `## Heading` | Section heading (levels 1–6) |
| `**bold**` | Bold text |
| `*italic*` | Italic text |
| `` `inline code` `` | Inline code mark |
| ` ```lang ... ``` ` | Fenced code block |
| `- item` | Bullet list |
| `1. item` | Ordered list |
| `> quote` | Blockquote panel |
| `[text](url)` | Hyperlink |
| `---` | Horizontal rule |

### ADF Limitation: No inline code inside bold (or italic)

Jira's ADF rejects text nodes that carry both `strong` (or `em`) and `code` marks simultaneously. This means **never nest backticks inside bold or italic**:

| Avoid | Use instead |
|---|---|
| `` **Text `symbol`** `` | `**Text** symbol` or `**Text symbol**` |
| `` *Text `symbol`* `` | `*Text* symbol` or `*Text symbol*` |

If you write `` **Endpoints no `TimesheetController`** ``, the API returns `400 INVALID_INPUT`. Split the styles so no single word carries both marks at once.

---

### IMPORTANT: Always use a temp file for multiline or formatted text

Passing multiline markdown directly in the shell argument breaks when the text contains backticks (`` ` `` or ```` ``` ````), quotes, or special characters. The safe pattern is:

```bash
cat > /tmp/jira_body.txt << 'EOF'
## My Heading

**Bold** and *italic* text.

- item 1
- item 2

```go
fmt.Println("hello")
```

> A blockquote

---

Plain `inline code` here.
EOF

go run ./cmd/jira issue comment <ISSUE-KEY> --body "$(cat /tmp/jira_body.txt)"
```

Apply the same pattern for `--description` in `create`, `subtask`, and `describe` commands.

---

## Configuration

### Initialize or update a profile

```
jira config init [--profile <name>]
```

Interactive prompts ask for: base URL, email, API token, default project key, default issue type.
Generate tokens at: https://id.atlassian.com/manage-profile/security/api-tokens

### List profiles

```
jira config list           # human-readable, active profile marked with *
jira --json config list    # JSON output
```

---

## Environment Variables

| Variable | What it overrides |
|---|---|
| `JIRA_PROFILE` | active profile |
| `JIRA_BASE_URL` | `base_url` |
| `JIRA_EMAIL` | `email` |
| `JIRA_API_TOKEN` | `api_token` |
| `JIRA_PROJECT` | `default_project_key` |

---

## Exit Codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | General / API error |
| `2` | Invalid input (missing required flag or argument) |
| `3` | Auth failed — check email and API token |
| `4` | Not found — issue key or project does not exist |

---

## Decision Guide

| Task | Command |
|---|---|
| Read an issue's description | `jira issue view <KEY>` |
| Create a subtask | `jira issue subtask <PARENT-KEY> --summary "..."` |
| List comments on an issue | `jira issue comments <KEY>` |
| Report a new bug | `jira issue create --type Bug --summary "..." --assign-me` |
| Create a story or task | `jira issue create --type Story/Task --summary "..." --assign-me` |
| Update what an issue is about | `jira issue describe <KEY> --description "..."` |
| Move issue through the board | `jira issue transition <KEY> --status "..."` |
| Leave a note on an issue | `jira issue comment <KEY> --body "..."` |
| Assign an existing issue to yourself | `jira issue assign <KEY> --assign-me` |
| Link two issues | `jira issue link <KEY> <TARGET> --type Blocks` |
| Work on a different Jira instance | `jira --profile <name> <command>` |
| Script multiple operations | Use `--json` and pipe to `jq` to extract keys |
