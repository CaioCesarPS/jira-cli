package api

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Issue struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

type IssueDetail struct {
	Key    string `json:"key"`
	Fields struct {
		Summary     string          `json:"summary"`
		Status      struct {
			Name string `json:"name"`
		} `json:"status"`
		Description json.RawMessage `json:"description"`
	} `json:"fields"`
}

func (d *IssueDetail) DescriptionText() string {
	if d.Fields.Description == nil || string(d.Fields.Description) == "null" {
		return ""
	}
	var adf map[string]interface{}
	if err := json.Unmarshal(d.Fields.Description, &adf); err != nil {
		return ""
	}
	return strings.TrimRight(extractADFText(adf), "\n")
}

func extractADFText(node map[string]interface{}) string {
	nodeType, _ := node["type"].(string)

	if nodeType == "text" {
		text, _ := node["text"].(string)
		return text
	}
	if nodeType == "hardBreak" {
		return "\n"
	}

	content, _ := node["content"].([]interface{})
	var sb strings.Builder

	for _, item := range content {
		child, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		childType, _ := child["type"].(string)
		childText := extractADFText(child)
		if childType == "listItem" {
			sb.WriteString("• ")
		}
		sb.WriteString(childText)
		switch childType {
		case "paragraph", "heading", "codeBlock", "blockquote", "listItem":
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (c *Client) CreateIssue(projectKey, summary, description, issueType, assigneeAccountID string) (*Issue, error) {
	fields := map[string]interface{}{
		"project":   map[string]string{"key": projectKey},
		"summary":   summary,
		"issuetype": map[string]string{"name": issueType},
	}
	if description != "" {
		fields["description"] = markdownToADF(description)
	}
	if assigneeAccountID != "" {
		fields["assignee"] = map[string]string{"accountId": assigneeAccountID}
	}

	body, _, err := c.do("POST", "/rest/api/3/issue", map[string]interface{}{"fields": fields})
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &issue, nil
}

func (c *Client) GetIssue(issueKey string) (*IssueDetail, error) {
	body, _, err := c.do("GET", "/rest/api/3/issue/"+issueKey+"?fields=summary,description,status", nil)
	if err != nil {
		return nil, err
	}

	var issue IssueDetail
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &issue, nil
}

func (c *Client) CreateSubtask(parentKey, summary, description, assigneeAccountID string) (*Issue, error) {
	// Team-managed (next-gen) projects require an explicit project on the create
	// payload; the parent alone is not enough and the API returns
	// "project: Specify a valid project ID or key". Derive the project key from
	// the parent issue key (the segment before the first "-", e.g. SED-563 -> SED).
	projectKey := parentKey
	if i := strings.Index(parentKey, "-"); i > 0 {
		projectKey = parentKey[:i]
	}
	fields := map[string]interface{}{
		"project":   map[string]string{"key": projectKey},
		"summary":   summary,
		"issuetype": map[string]string{"name": "Subtask"},
		"parent":    map[string]string{"key": parentKey},
	}
	if description != "" {
		fields["description"] = markdownToADF(description)
	}
	if assigneeAccountID != "" {
		fields["assignee"] = map[string]string{"accountId": assigneeAccountID}
	}

	body, _, err := c.do("POST", "/rest/api/3/issue", map[string]interface{}{"fields": fields})
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &issue, nil
}

func (c *Client) AssignIssue(issueKey, accountID string) error {
	_, _, err := c.do("PUT", "/rest/api/3/issue/"+issueKey+"/assignee", map[string]interface{}{"accountId": accountID})
	return err
}

func (c *Client) UpdateDescription(issueKey, description string) error {
	body := map[string]interface{}{
		"fields": map[string]interface{}{
			"description": markdownToADF(description),
		},
	}
	_, _, err := c.do("PUT", "/rest/api/3/issue/"+issueKey, body)
	return err
}

// ── list / search ──────────────────────────────────────────────────────────

// IssueSummary is a lightweight issue representation returned by search.
// Unlike IssueDetail, it only carries the fields needed to render a list view.
type IssueSummary struct {
	Key      string `json:"key"`
	Self     string `json:"self,omitempty"`
	Fields   struct {
		Summary  string `json:"summary"`
		Created  string `json:"created"`
		Updated  string `json:"updated"`
		Status   struct {
			Name string `json:"name"`
		} `json:"status"`
		Priority struct {
			Name string `json:"name"`
		} `json:"priority"`
		IssueType struct {
			Name string `json:"name"`
		} `json:"issuetype"`
		Assignee *struct {
			AccountID   string `json:"accountId"`
			DisplayName string `json:"displayName"`
		} `json:"assignee"`
		Reporter *struct {
			AccountID   string `json:"accountId"`
			DisplayName string `json:"displayName"`
		} `json:"reporter"`
		Labels []string `json:"labels"`
	} `json:"fields"`
}

// SearchOptions controls a single call to the Jira search endpoint.
type SearchOptions struct {
	JQL        string
	StartAt    int
	MaxResults int
}

// SearchResult is the response shape we expose to callers (Jira returns
// the same fields but we wrap them in our own type for stability).
type SearchResult struct {
	Total  int            `json:"total"`
	Issues []IssueSummary `json:"issues"`
}

// SearchIssues runs a JQL query and returns the first page of results.
// The Jira API caps MaxResults at 100 per call; callers paginate with
// StartAt to fetch more.
func (c *Client) SearchIssues(opts SearchOptions) (*SearchResult, error) {
	if opts.JQL == "" {
		return nil, fmt.Errorf("JQL query is required")
	}
	if opts.MaxResults <= 0 || opts.MaxResults > 100 {
		opts.MaxResults = 50
	}
	if opts.StartAt < 0 {
		opts.StartAt = 0
	}

	body, _, err := c.do("POST", "/rest/api/3/search", map[string]interface{}{
		"jql":        opts.JQL,
		"startAt":    opts.StartAt,
		"maxResults": opts.MaxResults,
		"fields": []string{
			"summary",
			"status",
			"priority",
			"issuetype",
			"assignee",
			"reporter",
			"labels",
			"created",
			"updated",
		},
	})
	if err != nil {
		return nil, err
	}

	var raw struct {
		Total  int            `json:"total"`
		StartAt int           `json:"startAt"`
		MaxResults int        `json:"maxResults"`
		Issues []IssueSummary `json:"issues"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}

	return &SearchResult{
		Total:  raw.Total,
		Issues: raw.Issues,
	}, nil
}

// SearchAllIssues paginates through the entire result set using the
// `SearchIssues` endpoint, accumulating all issues. It stops when Jira
// reports no more results or when maxTotal is reached (safety cap).
func (c *Client) SearchAllIssues(opts SearchOptions, maxTotal int) (*SearchResult, error) {
	const pageSize = 100

	accumulated := &SearchResult{Total: 0, Issues: nil}
	startAt := 0

	for {
		pageOpts := opts
		pageOpts.StartAt = startAt
		pageOpts.MaxResults = pageSize

		page, err := c.SearchIssues(pageOpts)
		if err != nil {
			return accumulated, err
		}

		accumulated.Issues = append(accumulated.Issues, page.Issues...)
		accumulated.Total = page.Total // Jira reports the global total on every page

		if len(page.Issues) == 0 {
			break
		}
		if maxTotal > 0 && len(accumulated.Issues) >= maxTotal {
			break
		}
		if startAt+len(page.Issues) >= page.Total {
			break
		}

		startAt += len(page.Issues)
	}

	return accumulated, nil
}
