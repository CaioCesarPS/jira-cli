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

### List issues (search/filter)

```
jira issue list [optional text query] [flags]
```

Aliases: `ls`, `search`.

Lists issues matching the given filters. By default prints a formatted table;
pass `--json` for structured output, or `--plain` for a tab-separated table
ideal for piping to other tools.

**Output modes:**
- default: aligned table (KEY, SUMMARY, STATUS, TYPE, PRIORITY, ASSIGNEE, UPDATED)
- `--json`: structured response with `data.issues[]` and `data.total`
- `--plain`: tab-separated, no decoration, easy to grep/awk
- `--raw`: prints the full Jira JSON response (debug)

**Filter flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--type` | `-t` | Issue type (Bug, Story, Task, ...) |
| `--status` | `-s` | Status (repeatable: `-s "To Do" -s "In Progress"`) |
| `--priority` | `-y` | Priority (Highest, High, Medium, Low, Lowest) |
| `--assignee` | `-a` | Assignee (`"me"`, `"x"` for unassigned, or account ID) |
| `--reporter` | `-r` | Reporter (`"me"` or account ID) |
| `--project` | `-P` | Project key (defaults to active profile's project) |
| `--component` | `-C` | Component name |
| `--parent` |  | Parent issue key |
| `--label` | `-l` | Label (repeatable, AND) |
| `--created` |  | Created date: `today`, `week`, `month`, `year`, `-Nd`, `yyyy-mm-dd` |
| `--updated` |  | Updated date (same format as `--created`) |
| `--created-after` / `--created-before` |  | Absolute date range |
| `--updated-after` / `--updated-before` |  | Absolute date range |
| `--jql` | `-q` | Raw JQL (overrides all other filters) |

**Shortcut flags:**

| Flag | Equivalent |
|------|------------|
| `--mine` | `--assignee="me"` |
| `--todo` | `--status="To Do"` |
| `--in-progress` | `--status="In Progress"` |
| `--done` | `--status="Done"` |

**Pagination & sorting:**

| Flag | Description |
|------|-------------|
| `--limit N` | Max issues per page (default 50, max 100) |
| `--offset N` | Skip the first N results |
| `--all` | Fetch every matching issue (paginates past 100) |
| `--order-by <field>` | Sort field (default: `created`) |
| `--reverse` | ASC instead of DESC |

**Examples:**

```bash
# My open work
jira issue list --mine --todo

# My recently updated issues
jira issue list --mine --updated -7d

# All high-priority bugs in the project
jira issue list -t Bug -y High

# Search for text in any field
jira issue list "OAuth login"

# Multiple labels
jira issue list -l backend -l urgent

# Get keys only, for piping into other commands
jira issue list --mine --todo --plain --no-headers --columns key
# (note: --columns is reserved; --plain emits all columns by default)

# Raw JQL
jira issue list -q "project = SED AND sprint in openSprints()"

# JSON for scripts
jira --json issue list --mine --todo

# Fetch every result (paginating past the 100-issue API cap)
jira issue list --mine --todo --all
```

**Pattern: pipe issue keys to other commands**

```bash
# Move every high-priority TODO issue to "In Progress"
jira --json issue list -y High --todo | \
  jq -r '.data.issues[].key' | \
  xargs -I {} jira issue transition {} --status "In Progress"
```

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

### Attach files to an issue

```
jira issue attach <ISSUE-KEY> <file> [file...]
```

- Uploads one or more local files (images, videos, or any file type) as attachments on the issue.
- Accepts multiple file paths in one call: `jira issue attach SED-180 ./screenshot.png ./clip.mp4`.
- Fails fast (exit code 2) before any network call if any given file path doesn't exist or is a directory.
- Use `--json` to get structured output: `{ "issue_key", "attachments": [{ "id", "filename", "size" }] }`.

### Move an issue to a sprint or back to the backlog

```
jira issue sprint <ISSUE-KEY> [--sprint-id <id>] [--board-id <id>] [--backlog]
```

**Flags:**

| Flag | Description |
|---|---|
| *(none)* | Auto-discover the board from `default_project_key` and move to it |
| `--sprint-id <id>` | Move to a specific sprint by ID (scrum boards) |
| `--board-id <id>` | Specify which board to use when the project has multiple boards |
| `--backlog` | Move the issue back to the backlog |

**Behavior by board type (auto-detected):**
- **Scrum boards** — moves to the active sprint via `POST /rest/agile/1.0/sprint/{id}/issue`
- **Next-gen / team-managed / simple boards** — moves directly onto the board via `POST /rest/agile/1.0/board/{id}/issue` (these boards have no sprint endpoint)

**Examples:**

```bash
# Move to board (works for both scrum and next-gen projects)
jira issue sprint SED-180

# Move to a specific sprint (scrum boards)
jira issue sprint SED-180 --sprint-id 42

# Disambiguate when the project has multiple boards
jira issue sprint SED-180 --board-id 7

# Send back to the backlog
jira issue sprint SED-180 --backlog
```

Use `--json` to get structured output: `{ "issue_key", "sprint_id", "sprint_name" }` for scrum, `{ "issue_key", "board_id", "board_name" }` for next-gen, or `{ "issue_key", "destination": "backlog" }`.

---

## IMPORTANT: Always Write Descriptions and Comments in Markdown

Whenever you create or update an issue description (`--description`) or add a comment (`--body`), the text **MUST** be written in Markdown. Never send plain prose — always use headings, lists, bold/italic, and code blocks to structure the content.

---

## Markdown Formatting in Body / Description

The CLI converts Markdown to Atlassian Document Format (ADF) automatically. Supported syntax:

| Markdown            | Result in Jira               |
| ------------------- | ---------------------------- |
| `## Heading`        | Section heading (levels 1–6) |
| `**bold**`          | Bold text                    |
| `*italic*`          | Italic text                  |
| `` `inline code` `` | Inline code mark             |
| ` ```lang ... ``` ` | Fenced code block            |
| `- item`            | Bullet list                  |
| `1. item`           | Ordered list                 |
| `> quote`           | Blockquote panel             |
| `[text](url)`       | Hyperlink                    |
| `---`               | Horizontal rule              |

### IMPORTANT: Always use a temp file for multiline or formatted text

Passing multiline markdown directly in the shell argument breaks when the text contains backticks (`` ` `` or ` ``` `), quotes, or special characters. The safe pattern is:

````bash
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
````

Apply the same pattern for `--description` in `create`, `subtask`, and `describe` commands.

**Assignar:** `jira issue assign <KEY> --assign-me` (não aceita email como argumento posicional)

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

| Variable         | What it overrides     |
| ---------------- | --------------------- |
| `JIRA_PROFILE`   | active profile        |
| `JIRA_BASE_URL`  | `base_url`            |
| `JIRA_EMAIL`     | `email`               |
| `JIRA_API_TOKEN` | `api_token`           |
| `JIRA_PROJECT`   | `default_project_key` |

---

## Exit Codes

| Code | Meaning                                           |
| ---- | ------------------------------------------------- |
| `0`  | Success                                           |
| `1`  | General / API error                               |
| `2`  | Invalid input (missing required flag or argument) |
| `3`  | Auth failed — check email and API token           |
| `4`  | Not found — issue key or project does not exist   |

---

## Decision Guide

|| Task                                        | Command                                                               |
|| ------------------------------------------- | --------------------------------------------------------------------- |
|| List issues matching filters                | `jira issue list [flags]` (`-t`, `-s`, `-y`, `-a`, `-l`, etc.)         |
|| List my open work                           | `jira issue list --mine --todo`                                       |
|| Free-text search                            | `jira issue list "search text"`                                       |
|| Read an issue's description                 | `jira issue view <KEY>`                                               |
|| Create a subtask                            | `jira issue subtask <PARENT-KEY> --summary "..."`                     |
|| List comments on an issue                   | `jira issue comments <KEY>`                                           |
|| Report a new bug                            | `jira issue create --type Bug --summary "..."`                        |
|| Create a story or task                      | `jira issue create --summary "..."` --assign-me`                      |
|| Update what an issue is about               | `jira issue describe <KEY> --description "$(cat /tmp/body.txt)"`      |
|| Move issue through the board                | `jira issue transition <KEY> --status "..."`                          |
|| Move issue from backlog to active sprint    | `jira issue sprint <KEY>`                                             |
|| Move issue to a specific sprint             | `jira issue sprint <KEY> --sprint-id <id>`                            |
|| Send issue back to the backlog              | `jira issue sprint <KEY> --backlog`                                   |
|| Leave a note on an issue                    | `jira issue comment <KEY> --body "$(cat /tmp/body.txt)"`              |
|| Attach a screenshot, video, or other file    | `jira issue attach <KEY> ./file1.png ./file2.mp4`                     |
|| Assign to yourself                          | `jira issue assign <KEY> --assign-me`                                 |
|| Work on a different Jira instance           | `jira --profile <name> <command>`                                     |
|| Script multiple operations                  | Use `--json` and pipe to `jq` to extract keys                         |
