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
		issue, err := client.CreateIssue(project, createSummary, createDescription, issueType)
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
		issue, err := client.CreateSubtask(args[0], subtaskSummary, subtaskDescription)
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

// ── helpers ───────────────────────────────────────────────────────────────────

func init() {
	createCmd.Flags().StringVar(&createSummary, "summary", "", "Issue summary (required)")
	createCmd.Flags().StringVar(&createDescription, "description", "", "Issue description")
	createCmd.Flags().StringVar(&createProject, "project", "", "Project key (e.g. PROJ)")
	createCmd.Flags().StringVar(&createType, "type", "", "Issue type (default: Task)")
	_ = createCmd.MarkFlagRequired("summary")

	subtaskCmd.Flags().StringVar(&subtaskSummary, "summary", "", "Subtask summary (required)")
	subtaskCmd.Flags().StringVar(&subtaskDescription, "description", "", "Subtask description")
	_ = subtaskCmd.MarkFlagRequired("summary")

	describeCmd.Flags().StringVar(&describeText, "description", "", "New description text (required)")
	_ = describeCmd.MarkFlagRequired("description")

	transitionCmd.Flags().StringVar(&transitionStatus, "status", "", "Target status name (required)")
	_ = transitionCmd.MarkFlagRequired("status")

	commentCmd.Flags().StringVar(&commentBody, "body", "", "Comment text (required)")
	_ = commentCmd.MarkFlagRequired("body")

	issueCmd.AddCommand(viewCmd, createCmd, subtaskCmd, describeCmd, transitionCmd, commentsCmd, commentCmd)
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
