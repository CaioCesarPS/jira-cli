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

func (c *Client) CreateIssue(projectKey, summary, description, issueType string) (*Issue, error) {
	fields := map[string]interface{}{
		"project":   map[string]string{"key": projectKey},
		"summary":   summary,
		"issuetype": map[string]string{"name": issueType},
	}
	if description != "" {
		fields["description"] = markdownToADF(description)
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

func (c *Client) CreateSubtask(parentKey, summary, description string) (*Issue, error) {
	fields := map[string]interface{}{
		"summary":   summary,
		"issuetype": map[string]string{"name": "Subtask"},
		"parent":    map[string]string{"key": parentKey},
	}
	if description != "" {
		fields["description"] = markdownToADF(description)
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

func (c *Client) UpdateDescription(issueKey, description string) error {
	body := map[string]interface{}{
		"fields": map[string]interface{}{
			"description": markdownToADF(description),
		},
	}
	_, _, err := c.do("PUT", "/rest/api/3/issue/"+issueKey, body)
	return err
}
