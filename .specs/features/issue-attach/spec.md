# Feature: `jira issue attach`

## Scope: Medium

Adds a CLI command to upload one or more files (images, videos, or any file type) as
attachments on a Jira issue, via the Jira REST API attachments endpoint.

## Requirements

- **AT-1**: `jira issue attach <issue-key> <file>...` uploads one or more local files as
  attachments to the given issue.
- **AT-2**: Accepts multiple file paths in a single invocation (variadic args, `cobra.MinimumNArgs(2)`
  — issue key + at least one file).
- **AT-3**: No file-type restriction — any file the user passes is uploaded, matching native
  Jira attachment behavior. "Videos and images" is the use case, not an enforced constraint.
- **AT-4**: Each file must exist and be readable on disk before any upload is attempted; if any
  file is missing/unreadable, fail fast with a clear error before making API calls (exit code 2).
- **AT-5**: Uses Jira's multipart attachment endpoint:
  `POST /rest/api/3/issue/{issueIdOrKey}/attachments` with header `X-Atlassian-Token: no-check`
  and `multipart/form-data` body (field name `file`). This requires new multipart support in
  `internal/api/client.go` (existing `do()` only marshals JSON bodies).
- **AT-6**: On success, prints the uploaded attachment(s) — id, filename, size — via
  `output.PrintResult`, consistent with other issue subcommands (JSON mode vs human mode).
- **AT-7**: On partial/total API failure, surfaces the Jira API error message and exits with the
  existing `apiExitCode(err)` convention (401/403 → 3, 404 → 4, else 1).

## Out of scope

- Downloading/listing existing attachments.
- Deleting attachments.
- Progress bars / streaming upload progress.
- MIME-type or file-size client-side validation.

## Acceptance criteria

- Running `jira issue attach PROJ-123 ./screenshot.png` uploads the file and prints a success
  message with the attachment id and filename.
- Running `jira issue attach PROJ-123 ./a.png ./b.mp4` uploads both files in one command.
- Running `jira issue attach PROJ-123 ./missing.png` fails before any HTTP call, exit code 2,
  clear "file not found" style error.
- `--json` mode emits structured data (issue_key, attachments: [{id, filename, size}]).
