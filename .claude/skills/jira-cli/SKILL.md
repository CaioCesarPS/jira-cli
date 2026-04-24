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
jira issue create --summary "<title>" [--description "<body>"] [--project <KEY>] [--type <Bug|Story|Task|...>]
```

- `--summary` is required; all other flags fall back to the active profile's defaults.
- Extract the created issue key from JSON when you need it in follow-up commands:
  ```bash
  KEY=$(jira --json issue create --summary "..." | jq -r '.data.issue_key')
  ```

### Create a subtask under a parent issue

```
jira issue subtask <PARENT-KEY> --summary "<title>" [--description "<body>"]
```

- Creates a subtask of type `Subtask` linked to the parent issue.
- The project is inferred from the parent key — no `--project` flag needed.
- Use `--json` to extract the new issue key:
  ```bash
  KEY=$(jira --json issue subtask SED-29 --summary "..." | jq -r '.data.issue_key')
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
| Report a new bug | `jira issue create --type Bug --summary "..."` |
| Create a story or task | `jira issue create --type Story/Task --summary "..."` |
| Update what an issue is about | `jira issue describe <KEY> --description "..."` |
| Move issue through the board | `jira issue transition <KEY> --status "..."` |
| Leave a note on an issue | `jira issue comment <KEY> --body "..."` |
| Work on a different Jira instance | `jira --profile <name> <command>` |
| Script multiple operations | Use `--json` and pipe to `jq` to extract keys |
