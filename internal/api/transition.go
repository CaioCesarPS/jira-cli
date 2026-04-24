package api

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) ListTransitions(issueKey string) ([]Transition, error) {
	body, _, err := c.do("GET", "/rest/api/3/issue/"+issueKey+"/transitions", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Transitions []Transition `json:"transitions"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return resp.Transitions, nil
}

func (c *Client) TransitionIssue(issueKey, statusName string) (string, error) {
	transitions, err := c.ListTransitions(issueKey)
	if err != nil {
		return "", err
	}

	target := strings.ToLower(statusName)
	for _, t := range transitions {
		if strings.ToLower(t.Name) == target {
			_, _, err := c.do("POST", "/rest/api/3/issue/"+issueKey+"/transitions", map[string]interface{}{
				"transition": map[string]string{"id": t.ID},
			})
			return t.Name, err
		}
	}

	// Build list of available statuses for a helpful error message
	names := make([]string, 0, len(transitions))
	for _, t := range transitions {
		names = append(names, t.Name)
	}
	return "", fmt.Errorf("status %q not found — available: %s", statusName, strings.Join(names, ", "))
}
