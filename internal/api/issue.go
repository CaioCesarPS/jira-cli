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
//
// The legacy /rest/api/3/search endpoint was removed by Atlassian in
// favour of /rest/api/3/search/jql, which uses opaque cursor tokens
// instead of startAt. We expose NextPageToken to callers; the cursor
// returned by the previous response should be passed back on the next
// call. An empty NextPageToken means "first page".
type SearchOptions struct {
	JQL           string
	NextPageToken string
	MaxResults    int
}

// SearchResult is the response shape we expose to callers. NextPageToken
// is empty when there are no more pages (i.e. isLast == true).
type SearchResult struct {
	Total         int            `json:"total"`
	Issues        []IssueSummary `json:"issues"`
	NextPageToken string         `json:"nextPageToken,omitempty"`
}

// SearchIssues runs a JQL query and returns a single page of results.
// The Jira API caps MaxResults at 100 per call; callers paginate by
// passing the returned NextPageToken back on the next call.
func (c *Client) SearchIssues(opts SearchOptions) (*SearchResult, error) {
	if opts.JQL == "" {
		return nil, fmt.Errorf("JQL query is required")
	}
	if opts.MaxResults <= 0 || opts.MaxResults > 100 {
		opts.MaxResults = 50
	}

	payload := map[string]interface{}{
		"jql":        opts.JQL,
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
	}
	if opts.NextPageToken != "" {
		payload["nextPageToken"] = opts.NextPageToken
	}

	body, _, err := c.do("POST", "/rest/api/3/search/jql", payload)
	if err != nil {
		return nil, err
	}

	// The new endpoint returns a flat response (no "total" — Jira
	// removed it from the cursor-based API). We use len(issues) as
	// the "total so far" for backwards compatibility with the prior
	// signature, and an external "total" only when present.
	var raw struct {
		Issues        []IssueSummary `json:"issues"`
		NextPageToken string         `json:"nextPageToken,omitempty"`
		IsLast        bool           `json:"isLast,omitempty"`
		Total         int            `json:"total,omitempty"` // present on some responses
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}

	total := raw.Total
	if total == 0 {
		total = len(raw.Issues)
	}

	return &SearchResult{
		Total:         total,
		Issues:        raw.Issues,
		NextPageToken: raw.NextPageToken,
	}, nil
}

// SearchAllIssues paginates through the entire result set using cursor
// tokens returned by the new /search/jql endpoint, accumulating all
// issues. It stops when the server reports isLast == true (no
// NextPageToken) or when maxTotal is reached (safety cap).
func (c *Client) SearchAllIssues(opts SearchOptions, maxTotal int) (*SearchResult, error) {
	const pageSize = 100

	accumulated := &SearchResult{Total: 0, Issues: nil}
	opts.MaxResults = pageSize

	for {
		page, err := c.SearchIssues(opts)
		if err != nil {
			return accumulated, err
		}

		accumulated.Issues = append(accumulated.Issues, page.Issues...)

		if page.NextPageToken == "" {
			// No more pages
			break
		}
		if maxTotal > 0 && len(accumulated.Issues) >= maxTotal {
			break
		}

		opts.NextPageToken = page.NextPageToken
	}

	accumulated.Total = len(accumulated.Issues)
	return accumulated, nil
}
