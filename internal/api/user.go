package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type User struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
}

func (c *Client) GetCurrentUserAccountID() (string, error) {
	path := "/rest/api/3/user/search?query=" + url.QueryEscape(c.email)
	body, _, err := c.do("GET", path, nil)
	if err != nil {
		return "", err
	}

	var users []User
	if err := json.Unmarshal(body, &users); err != nil {
		return "", fmt.Errorf("parse user search response: %w", err)
	}
	if len(users) == 0 {
		return "", fmt.Errorf("no Jira user found for email %q", c.email)
	}
	return users[0].AccountID, nil
}
