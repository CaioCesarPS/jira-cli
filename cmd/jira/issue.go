package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/caiocesarps/jira-cli/internal/api"
	"github.com/caiocesarps/jira-cli/internal/config"
	"github.com/caiocesarps/jira-cli/internal/output"
)

// ── attach ────────────────────────────────────────────────────────────────────

var attachCmd = &cobra.Command{
	Use:   "attach <issue-key> <file>...",
	Short: "Attach one or more files (images, videos, etc.) to an issue",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		issueKey := args[0]
		filePaths := args[1:]

		for _, path := range filePaths {
			info, err := os.Stat(path)
			if err != nil {
				return exit(fmt.Errorf("file not found: %s", path), 2)
			}
			if info.IsDir() {
				return exit(fmt.Errorf("not a file: %s", path), 2)
			}
		}

		profile, err := config.Load(profileFlag)
		if err != nil {
			return exit(err, 2)
		}

		client := api.NewClient(profile)
		attachments, err := client.AttachFiles(issueKey, filePaths)
		if err != nil {
			return exit(err, apiExitCode(err))
		}

		jsonData := make([]map[string]interface{}, 0, len(attachments))
		var humanLines []string
		for _, a := range attachments {
			jsonData = append(jsonData, map[string]interface{}{
				"id":       a.ID,
				"filename": a.Filename,
				"size":     a.Size,
			})
			humanLines = append(humanLines, fmt.Sprintf("  - %s (id=%s, %d bytes)", a.Filename, a.ID, a.Size))
		}
		output.PrintResult(
			map[string]interface{}{"issue_key": issueKey, "attachments": jsonData},
			fmt.Sprintf("Attached %d file(s) to %s\n%s", len(attachments), issueKey, strings.Join(humanLines, "\n")),
		)
		return nil
	},
}

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Manage Jira issues",
}

// ── create ────────────────────────────────────────────────────────────────────

var (
	createSummary     string
	createDescription string
	createProject     string
	createType        string
	createAssignee    string
	createAutoAssign  bool
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Jira issue",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := config.Load(profileFlag)
		if err != nil {
			return exit(err, 2)
		}

		project := createProject
		if project == "" {
			project = profile.DefaultProjectKey
		}
		if project == "" {
			return exit(fmt.Errorf("project key required — use --project or set default_project_key in the config"), 2)
		}

		issueType := createType
		if issueType == "" {
			issueType = profile.DefaultIssueType
		}
		if issueType == "" {
			issueType = "Task"
		}

		client := api.NewClient(profile)

		assigneeID := createAssignee
		if assigneeID == "" && createAutoAssign {
			id, err := client.GetCurrentUserAccountID()
			if err != nil {
				return exit(fmt.Errorf("auto-assign: %w", err), 1)
			}
			assigneeID = id
		}

		issue, err := client.CreateIssue(project, createSummary, createDescription, issueType, assigneeID)
		if err != nil {
			return exit(err, apiExitCode(err))
		}

		url := profile.BaseURL + "/browse/" + issue.Key
		output.PrintResult(map[string]string{
			"issue_key": issue.Key,
			"issue_id":  issue.ID,
			"url":       url,
		}, fmt.Sprintf("Created issue %s\n→ %s", issue.Key, url))
		return nil
	},
}

// ── view ──────────────────────────────────────────────────────────────────────

var viewCmd = &cobra.Command{
	Use:   "view <issue-key>",
	Short: "Show the summary and description of an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := config.Load(profileFlag)
		if err != nil {
			return exit(err, 2)
		}

		client := api.NewClient(profile)
		issue, err := client.GetIssue(args[0])
		if err != nil {
			return exit(err, apiExitCode(err))
		}

		desc := issue.DescriptionText()
		output.PrintResult(
			map[string]string{
				"issue_key":   issue.Key,
				"summary":     issue.Fields.Summary,
				"status":      issue.Fields.Status.Name,
				"description": desc,
			},
			fmt.Sprintf("[%s] %s\nStatus: %s\n\n%s", issue.Key, issue.Fields.Summary, issue.Fields.Status.Name, desc),
		)
		return nil
	},
}

// ── subtask ───────────────────────────────────────────────────────────────────

var (
	subtaskSummary     string
	subtaskDescription string
	subtaskAssignee    string
	subtaskAutoAssign  bool
)

var subtaskCmd = &cobra.Command{
	Use:   "subtask <parent-issue-key>",
	Short: "Create a subtask under a parent issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := config.Load(profileFlag)
		if err != nil {
			return exit(err, 2)
		}

		client := api.NewClient(profile)

		assigneeID := subtaskAssignee
		if assigneeID == "" && subtaskAutoAssign {
			id, err := client.GetCurrentUserAccountID()
			if err != nil {
				return exit(fmt.Errorf("auto-assign: %w", err), 1)
			}
			assigneeID = id
		}

		issue, err := client.CreateSubtask(args[0], subtaskSummary, subtaskDescription, assigneeID)
		if err != nil {
			return exit(err, apiExitCode(err))
		}

		url := profile.BaseURL + "/browse/" + issue.Key
		output.PrintResult(map[string]string{
			"issue_key":  issue.Key,
			"issue_id":   issue.ID,
			"parent_key": args[0],
			"url":        url,
		}, fmt.Sprintf("Created subtask %s under %s\n→ %s", issue.Key, args[0], url))
		return nil
	},
}

// ── describe ──────────────────────────────────────────────────────────────────

var describeText string

var describeCmd = &cobra.Command{
	Use:   "describe <issue-key>",
	Short: "Update the description of an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := config.Load(profileFlag)
		if err != nil {
			return exit(err, 2)
		}

		client := api.NewClient(profile)
		if err := client.UpdateDescription(args[0], describeText); err != nil {
			return exit(err, apiExitCode(err))
		}

		output.PrintResult(map[string]string{"issue_key": args[0]},
			fmt.Sprintf("Updated description for %s", args[0]))
		return nil
	},
}

// ── transition ────────────────────────────────────────────────────────────────

var transitionStatus string

var transitionCmd = &cobra.Command{
	Use:   "transition <issue-key>",
	Short: "Change the status of an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := config.Load(profileFlag)
		if err != nil {
			return exit(err, 2)
		}

		client := api.NewClient(profile)
		appliedStatus, err := client.TransitionIssue(args[0], transitionStatus)
		if err != nil {
			return exit(err, apiExitCode(err))
		}

		output.PrintResult(
			map[string]string{"issue_key": args[0], "status": appliedStatus},
			fmt.Sprintf("%s → %s", args[0], appliedStatus),
		)
		return nil
	},
}

// ── comments (list) ───────────────────────────────────────────────────────────

var commentsCmd = &cobra.Command{
	Use:   "comments <issue-key>",
	Short: "List all comments on an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := config.Load(profileFlag)
		if err != nil {
			return exit(err, 2)
		}

		client := api.NewClient(profile)
		comments, err := client.GetComments(args[0])
		if err != nil {
			return exit(err, apiExitCode(err))
		}

		if len(comments) == 0 {
			output.PrintResult(nil, fmt.Sprintf("%s has no comments.", args[0]))
			return nil
		}

		// JSON mode: emit array of comment objects
		jsonData := make([]map[string]string, 0, len(comments))
		var humanLines []string
		for i, c := range comments {
			text := c.BodyText()
			jsonData = append(jsonData, map[string]string{
				"id":      c.ID,
				"author":  c.Author.DisplayName,
				"created": c.Created,
				"body":    text,
			})
			humanLines = append(humanLines, fmt.Sprintf("[%d] %s (%s)\n%s", i+1, c.Author.DisplayName, c.Created[:10], text))
		}
		output.PrintResult(map[string]interface{}{"comments": jsonData},
			fmt.Sprintf("%s — %d comment(s)\n\n%s", args[0], len(comments), strings.Join(humanLines, "\n\n---\n\n")))
		return nil
	},
}

// ── comment (add) ─────────────────────────────────────────────────────────────

var commentBody string

var commentCmd = &cobra.Command{
	Use:   "comment <issue-key>",
	Short: "Add a comment to an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := config.Load(profileFlag)
		if err != nil {
			return exit(err, 2)
		}

		client := api.NewClient(profile)
		comment, err := client.AddComment(args[0], commentBody)
		if err != nil {
			return exit(err, apiExitCode(err))
		}

		output.PrintResult(
			map[string]string{"issue_key": args[0], "comment_id": comment.ID},
			fmt.Sprintf("Comment added to %s", args[0]),
		)
		return nil
	},
}

// ── assign ────────────────────────────────────────────────────────────────────

var (
	assignAccountID string
	assignMe        bool
)

var assignCmd = &cobra.Command{
	Use:   "assign <issue-key>",
	Short: "Assign an issue to a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := config.Load(profileFlag)
		if err != nil {
			return exit(err, 2)
		}

		if assignAccountID == "" && !assignMe {
			return exit(fmt.Errorf("provide --assign-me or --assignee <account-id>"), 2)
		}

		client := api.NewClient(profile)

		accountID := assignAccountID
		if accountID == "" {
			id, err := client.GetCurrentUserAccountID()
			if err != nil {
				return exit(fmt.Errorf("auto-assign: %w", err), 1)
			}
			accountID = id
		}

		if err := client.AssignIssue(args[0], accountID); err != nil {
			return exit(err, apiExitCode(err))
		}

		output.PrintResult(
			map[string]string{"issue_key": args[0], "assignee": accountID},
			fmt.Sprintf("Assigned %s", args[0]),
		)
		return nil
	},
}

// ── link ──────────────────────────────────────────────────────────────────────

var linkType string

var linkCmd = &cobra.Command{
	Use:   "link <issue-key> <target-issue-key>",
	Short: "Link two issues together",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := config.Load(profileFlag)
		if err != nil {
			return exit(err, 2)
		}

		client := api.NewClient(profile)
		if err := client.LinkIssues(args[0], args[1], linkType); err != nil {
			return exit(err, apiExitCode(err))
		}

		output.PrintResult(
			map[string]string{"inward": args[0], "outward": args[1], "type": linkType},
			fmt.Sprintf("Linked %s → %s (%s)", args[0], args[1], linkType),
		)
		return nil
	},
}

// ── sprint ────────────────────────────────────────────────────────────────────

var (
	sprintID      int
	sprintBoardID int
	sprintBacklog bool
)

var sprintCmd = &cobra.Command{
	Use:   "sprint <issue-key>",
	Short: "Move an issue to a sprint or back to the backlog",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		issueKey := args[0]

		profile, err := config.Load(profileFlag)
		if err != nil {
			return exit(err, 2)
		}

		client := api.NewClient(profile)

		if sprintBacklog {
			if err := client.MoveIssuesToBacklog([]string{issueKey}); err != nil {
				return exit(err, apiExitCode(err))
			}
			output.PrintResult(
				map[string]string{"issue_key": issueKey, "destination": "backlog"},
				fmt.Sprintf("%s → backlog", issueKey),
			)
			return nil
		}

		// Move to sprint (--sprint-id) or auto-discover board and move to active sprint / board
		projectKey := profile.DefaultProjectKey
		if projectKey == "" && sprintID == 0 {
			return exit(fmt.Errorf("project key required — set default_project_key in config or use --sprint-id"), 2)
		}

		// Resolve board ID when needed (auto-discover or explicit --board-id)
		boardID := sprintBoardID
		var board *api.Board
		if boardID == 0 && sprintID == 0 {
			boards, err := client.GetBoardsForProject(projectKey)
			if err != nil {
				return exit(fmt.Errorf("list boards: %w", err), apiExitCode(err))
			}
			if len(boards) == 0 {
				return exit(fmt.Errorf("no boards found for project %s", projectKey), 4)
			}
			board = &boards[0]
			boardID = board.ID
			if len(boards) > 1 {
				fmt.Fprintf(os.Stderr, "warning: multiple boards found, using %q (id=%d) — use --board-id to specify\n", board.Name, board.ID)
			}
		}

		// If explicit sprint ID: move directly to that sprint
		if sprintID != 0 {
			if err := client.MoveIssuesToSprint(sprintID, []string{issueKey}); err != nil {
				return exit(err, apiExitCode(err))
			}
			output.PrintResult(
				map[string]string{"issue_key": issueKey, "sprint_id": fmt.Sprint(sprintID)},
				fmt.Sprintf("%s → sprint %d", issueKey, sprintID),
			)
			return nil
		}

		// Try active sprint first (scrum boards); fall back to board move (next-gen/kanban)
		sprints, err := client.GetSprintsForBoard(boardID, "active")
		if err == nil && len(sprints) > 0 {
			s := sprints[0]
			if err := client.MoveIssuesToSprint(s.ID, []string{issueKey}); err != nil {
				return exit(err, apiExitCode(err))
			}
			output.PrintResult(
				map[string]string{"issue_key": issueKey, "sprint_id": fmt.Sprint(s.ID), "sprint_name": s.Name},
				fmt.Sprintf("%s → %q (sprint id=%d)", issueKey, s.Name, s.ID),
			)
			return nil
		}

		// Board doesn't support sprints (next-gen/simple) — move to board directly
		if err := client.MoveIssuesToBoard(boardID, []string{issueKey}); err != nil {
			return exit(err, apiExitCode(err))
		}

		boardName := fmt.Sprintf("board %d", boardID)
		if board != nil {
			boardName = board.Name
		}
		output.PrintResult(
			map[string]string{"issue_key": issueKey, "board_id": fmt.Sprint(boardID), "board_name": boardName},
			fmt.Sprintf("%s → %q (board)", issueKey, boardName),
		)
		return nil
	},
}

// ── helpers ───────────────────────────────────────────────────────────────────

func init() {
	createCmd.Flags().StringVar(&createSummary, "summary", "", "Issue summary (required)")
	createCmd.Flags().StringVar(&createDescription, "description", "", "Issue description")
	createCmd.Flags().StringVar(&createProject, "project", "", "Project key (e.g. PROJ)")
	createCmd.Flags().StringVar(&createType, "type", "", "Issue type (default: Task)")
	createCmd.Flags().StringVar(&createAssignee, "assignee", "", "Assignee account ID")
	createCmd.Flags().BoolVar(&createAutoAssign, "assign-me", false, "Assign issue to yourself (resolved via API)")
	_ = createCmd.MarkFlagRequired("summary")

	subtaskCmd.Flags().StringVar(&subtaskSummary, "summary", "", "Subtask summary (required)")
	subtaskCmd.Flags().StringVar(&subtaskDescription, "description", "", "Subtask description")
	subtaskCmd.Flags().StringVar(&subtaskAssignee, "assignee", "", "Assignee account ID")
	subtaskCmd.Flags().BoolVar(&subtaskAutoAssign, "assign-me", false, "Assign subtask to yourself (resolved via API)")
	_ = subtaskCmd.MarkFlagRequired("summary")

	describeCmd.Flags().StringVar(&describeText, "description", "", "New description text (required)")
	_ = describeCmd.MarkFlagRequired("description")

	transitionCmd.Flags().StringVar(&transitionStatus, "status", "", "Target status name (required)")
	_ = transitionCmd.MarkFlagRequired("status")

	commentCmd.Flags().StringVar(&commentBody, "body", "", "Comment text (required)")
	_ = commentCmd.MarkFlagRequired("body")

	linkCmd.Flags().StringVar(&linkType, "type", "Relates", "Link type (e.g. Blocks, Clones, Relates)")

	assignCmd.Flags().StringVar(&assignAccountID, "assignee", "", "Assignee account ID")
	assignCmd.Flags().BoolVar(&assignMe, "assign-me", false, "Assign to yourself (resolved via API)")

	sprintCmd.Flags().IntVar(&sprintID, "sprint-id", 0, "Target sprint ID (skips board/sprint discovery)")
	sprintCmd.Flags().IntVar(&sprintBoardID, "board-id", 0, "Board ID to use when discovering the active sprint")
	sprintCmd.Flags().BoolVar(&sprintBacklog, "backlog", false, "Move issue back to the backlog")

	issueCmd.AddCommand(viewCmd, createCmd, subtaskCmd, describeCmd, transitionCmd, commentsCmd, commentCmd, linkCmd, assignCmd, sprintCmd, attachCmd)
}

func exit(err error, code int) error {
	output.PrintError(err, code)
	os.Exit(code)
	return nil
}

func apiExitCode(err error) int {
	if apiErr, ok := err.(*api.APIError); ok {
		switch apiErr.StatusCode {
		case 401, 403:
			return 3
		case 404:
			return 4
		}
	}
	return 1
}
