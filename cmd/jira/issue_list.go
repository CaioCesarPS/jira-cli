package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/caiocesarps/jira-cli/internal/api"
	"github.com/caiocesarps/jira-cli/internal/config"
	"github.com/caiocesarps/jira-cli/internal/output"
)

// ── list flag state ─────────────────────────────────────────────────────────

var (
	listType          string
	listPriority      string
	listAssignee      string
	listReporter      string
	listProject       string
	listComponent     string
	listParent        string
	listLabels        []string
	listCreated       string
	listUpdated       string
	listCreatedAfter  string
	listCreatedBefore string
	listUpdatedAfter  string
	listUpdatedBefore string
	listRawJQL        string
	listOrderBy       string
	listReverse       bool
	listLimit         int
	listOffset        int
	listFetchAll      bool
	listColumns       string
	listNoHeaders     bool
	listPlain         bool
	listMine          bool
	listTodo          bool
	listInProgress    bool
	listDone          bool
	listRaw           bool
)

var listCmd = &cobra.Command{
	Use:     "list [text query]",
	Short:   "List Jira issues with filters",
	Long: `List Jira issues matching the given filters. By default prints a formatted
table; pass --json to get a structured response, or --plain for a tab-separated
table suitable for piping to other tools (cut, awk, xargs, ...).

Use --all to fetch every matching issue (paginating past the 100-issue
API limit). Without --all, the API call returns at most 100 results.`,
	Aliases: []string{"ls", "search"},
	Args:    cobra.RangeArgs(0, 1),
	RunE:    runList,
}

func runList(cmd *cobra.Command, args []string) error {
	profile, err := config.Load(profileFlag)
	if err != nil {
		return exit(err, 2)
	}

	// Resolve shortcut flags (--mine, --todo, --in-progress) into the
	// underlying filter values before building JQL.
	if listMine && listAssignee == "" {
		listAssignee = "me"
	}
	if listTodo {
		listType = "" // not a type filter; status is handled below
	}
	statuses := []string{}
	if listTodo {
		statuses = append(statuses, "To Do")
	}
	if listInProgress {
		statuses = append(statuses, "In Progress")
	}
	if listDone {
		statuses = append(statuses, "Done")
	}
	if v, _ := cmd.Flags().GetStringSlice("status"); len(v) > 0 {
		statuses = append(statuses, v...)
	}

	// Project comes from --project flag, otherwise the profile default.
	project := listProject
	if project == "" {
		project = profile.DefaultProjectKey
	}

	// Free-text query (positional argument) becomes a `text ~ "..."` filter.
	var textQuery string
	if len(args) > 0 {
		textQuery = strings.Join(args, " ")
	}

	// Resolve "me" → account ID if needed.
	var currentUser *api.User
	needsMe := listAssignee == "me" || listReporter == "me" ||
		(listMine && listAssignee == "me")
	if needsMe {
		client := api.NewClient(profile)
		accountID, err := client.GetCurrentUserAccountID()
		if err != nil {
			return exit(fmt.Errorf("resolve current user: %w", err), 1)
		}
		currentUser = &api.User{AccountID: accountID, DisplayName: profile.Email}
	}

	// Build the JQL.
	jql := api.NewJQL(project).
		Type(listType).
		Status(statuses...).
		Priority(listPriority).
		Assignee(listAssignee, currentUser).
		Reporter(listReporter, currentUser).
		Labels(listLabels...).
		Component(listComponent).
		Parent(listParent).
		Text(textQuery).
		Created(listCreated, listCreatedAfter, listCreatedBefore).
		Updated(listUpdated, listUpdatedAfter, listUpdatedBefore).
		OrderBy(listOrderBy).
		Reverse(listReverse).
		Build()

	// Raw --jql overrides the entire composed query.
	if listRawJQL != "" {
		jql = listRawJQL
	}

	// Run the search.
	client := api.NewClient(profile)
	var result *api.SearchResult

	if listFetchAll {
		result, err = client.SearchAllIssues(api.SearchOptions{
			JQL: jql,
		}, listLimit)
	} else {
		result, err = client.SearchIssues(api.SearchOptions{
			JQL:        jql,
			StartAt:    listOffset,
			MaxResults: listLimit,
		})
	}
	if err != nil {
		return exit(err, apiExitCode(err))
	}

	// Render.
	return renderList(result, jql)
}

// renderList dispatches to JSON, plain, or table output based on flags.
func renderList(result *api.SearchResult, jql string) error {
	switch {
	case listRaw:
		// Raw Jira response — useful for debugging
		output.PrintResult(map[string]interface{}{
			"jql":    jql,
			"total":  result.Total,
			"issues": result.Issues,
		}, fmt.Sprintf("%d issue(s) found", result.Total))
		return nil
	case output.JSONMode || listPlain:
		printListPlainOrJSON(result, jql)
	default:
		printListTable(result)
	}
	return nil
}

func printListPlainOrJSON(result *api.SearchResult, jql string) {
	if output.JSONMode {
		output.PrintResult(
			map[string]interface{}{
				"total":  result.Total,
				"issues": result.Issues,
			},
			fmt.Sprintf("%d issue(s) found", result.Total),
		)
		_ = jql // JQL goes in human mode only
		return
	}
	// plain text — TSV
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if !listNoHeaders {
		fmt.Fprintln(w, "KEY	SUMMARY	STATUS	TYPE	PRIORITY	ASSIGNEE	LABELS	UPDATED")
	}
	for _, iss := range result.Issues {
		assignee := "-"
		if iss.Fields.Assignee != nil {
			assignee = iss.Fields.Assignee.DisplayName
		}
		fmt.Fprintf(w, "%s	%s	%s	%s	%s	%s	%s	%s\n",
			iss.Key,
			iss.Fields.Summary,
			iss.Fields.Status.Name,
			iss.Fields.IssueType.Name,
			priorityOrDash(iss.Fields.Priority.Name),
			assignee,
			strings.Join(iss.Fields.Labels, ","),
			shortDate(iss.Fields.Updated),
		)
	}
	_ = w.Flush()
}

func printListTable(result *api.SearchResult) {
	if len(result.Issues) == 0 {
		fmt.Fprintf(os.Stderr, "No issues found.\n")
		return
	}

	// Always include the URL prefix
	w := tabwriter.NewWriter(io.Writer(os.Stdout), 0, 0, 2, ' ', 0)

	// Header
	if !listNoHeaders {
		fmt.Fprintln(w, "KEY\tSUMMARY\tSTATUS\tTYPE\tPRIORITY\tASSIGNEE\tUPDATED")
	}
	for _, iss := range result.Issues {
		assignee := "-"
		if iss.Fields.Assignee != nil {
			assignee = iss.Fields.Assignee.DisplayName
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			iss.Key,
			truncate(iss.Fields.Summary, 50),
			iss.Fields.Status.Name,
			iss.Fields.IssueType.Name,
			priorityOrDash(iss.Fields.Priority.Name),
			truncate(assignee, 20),
			shortDate(iss.Fields.Updated),
		)
	}
	_ = w.Flush()

	fmt.Fprintf(os.Stderr, "\n%d issue(s) found", result.Total)
	if listFetchAll {
		fmt.Fprintf(os.Stderr, " (fetched all)")
	}
	fmt.Fprintln(os.Stderr)
}

// ── small helpers ───────────────────────────────────────────────────────────

func priorityOrDash(p string) string {
	if p == "" {
		return "-"
	}
	return p
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}

func shortDate(iso string) string {
	// Jira returns "2026-06-10T12:00:00.000+0000" — show yyyy-mm-dd
	if len(iso) < 10 {
		return iso
	}
	return iso[:10]
}

// (sorting is currently server-side; reserved for future client-side sort)
var _ = sort.Strings

func init() {
	listCmd.Flags().StringVarP(&listType, "type", "t", "", "Filter by issue type (Bug, Story, Task, ...)")
	listCmd.Flags().StringVarP(&listPriority, "priority", "y", "", "Filter by priority (Highest, High, Medium, Low, Lowest)")
	listCmd.Flags().StringVarP(&listAssignee, "assignee", "a", "", `Filter by assignee ("me", "x" for unassigned, or account ID)`)
	listCmd.Flags().StringVarP(&listReporter, "reporter", "r", "", `Filter by reporter ("me" or account ID)`)
	listCmd.Flags().StringVarP(&listProject, "project", "P", "", "Filter by project (defaults to the active profile's project)")
	listCmd.Flags().StringVarP(&listComponent, "component", "C", "", "Filter by component")
	listCmd.Flags().StringVar(&listParent, "parent", "", "Filter by parent issue key")
	listCmd.Flags().StringArrayVarP(&listLabels, "label", "l", []string{}, "Filter by label (repeatable, AND)")
	listCmd.Flags().StringVar(&listCreated, "created", "", "Filter by created date (today, week, month, year, -Nd, yyyy-mm-dd)")
	listCmd.Flags().StringVar(&listUpdated, "updated", "", "Filter by updated date (today, week, month, year, -Nd, yyyy-mm-dd)")
	listCmd.Flags().StringVar(&listCreatedAfter, "created-after", "", "Issues created on or after this date (yyyy-mm-dd)")
	listCmd.Flags().StringVar(&listCreatedBefore, "created-before", "", "Issues created on or before this date (yyyy-mm-dd)")
	listCmd.Flags().StringVar(&listUpdatedAfter, "updated-after", "", "Issues updated on or after this date (yyyy-mm-dd)")
	listCmd.Flags().StringVar(&listUpdatedBefore, "updated-before", "", "Issues updated on or before this date (yyyy-mm-dd)")
	listCmd.Flags().StringVarP(&listRawJQL, "jql", "q", "", "Raw JQL query (overrides all other filters)")
	listCmd.Flags().StringVar(&listOrderBy, "order-by", "created", "Field to order by (created, updated, priority, duedate)")
	listCmd.Flags().BoolVar(&listReverse, "reverse", false, "Reverse the display order (default is DESC)")
	listCmd.Flags().IntVar(&listLimit, "limit", 50, "Max issues to return per page (max 100)")
	listCmd.Flags().IntVar(&listOffset, "offset", 0, "Skip the first N results")
	listCmd.Flags().BoolVar(&listFetchAll, "all", false, "Fetch every matching issue (paginates past the 100-issue limit)")

	// Output formatting
	listCmd.Flags().BoolVar(&listPlain, "plain", false, "Output as a tab-separated table (good for piping)")
	listCmd.Flags().BoolVar(&listNoHeaders, "no-headers", false, "Hide the table header (works with --plain)")
	listCmd.Flags().BoolVar(&listRaw, "raw", false, "Print raw Jira response as JSON (debug)")

	// Shortcut filters
	listCmd.Flags().BoolVar(&listMine, "mine", false, `Shortcut for --assignee="me"`)
	listCmd.Flags().BoolVar(&listTodo, "todo", false, `Shortcut for --status="To Do"`)
	listCmd.Flags().BoolVar(&listInProgress, "in-progress", false, `Shortcut for --status="In Progress"`)
	listCmd.Flags().BoolVar(&listDone, "done", false, `Shortcut for --status="Done"`)

	// Hidden flag that shows up only in --help when explicitly requested
	listCmd.Flags().StringVar(&listColumns, "columns", "", "Comma-separated list of columns (advanced)")
	_ = listCmd.Flags().MarkHidden("columns")

	issueCmd.AddCommand(listCmd)
}
