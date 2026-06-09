package api

import (
	"encoding/json"
	"fmt"
)

type Board struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type Sprint struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

type boardsResponse struct {
	Values []Board `json:"values"`
}

type sprintsResponse struct {
	Values []Sprint `json:"values"`
}

func (c *Client) GetBoardsForProject(projectKey string) ([]Board, error) {
	body, _, err := c.do("GET", "/rest/agile/1.0/board?projectKeyOrId="+projectKey, nil)
	if err != nil {
		return nil, err
	}
	var resp boardsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse boards: %w", err)
	}
	return resp.Values, nil
}

func (c *Client) GetSprintsForBoard(boardID int, state string) ([]Sprint, error) {
	path := fmt.Sprintf("/rest/agile/1.0/board/%d/sprint", boardID)
	if state != "" {
		path += "?state=" + state
	}
	body, _, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp sprintsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse sprints: %w", err)
	}
	return resp.Values, nil
}

func (c *Client) MoveIssuesToSprint(sprintID int, issueKeys []string) error {
	path := fmt.Sprintf("/rest/agile/1.0/sprint/%d/issue", sprintID)
	_, _, err := c.do("POST", path, map[string]any{"issues": issueKeys})
	return err
}

// MoveIssuesToBoard moves issues from the backlog onto the board.
// Used for next-gen (team-managed) projects where boards don't have sprints.
func (c *Client) MoveIssuesToBoard(boardID int, issueKeys []string) error {
	path := fmt.Sprintf("/rest/agile/1.0/board/%d/issue", boardID)
	_, _, err := c.do("POST", path, map[string]any{"issues": issueKeys})
	return err
}

func (c *Client) MoveIssuesToBacklog(issueKeys []string) error {
	_, _, err := c.do("POST", "/rest/agile/1.0/backlog/issue", map[string]any{"issues": issueKeys})
	return err
}
