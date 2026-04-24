package api

import (
	"encoding/json"
	"fmt"
)

type Comment struct {
	ID   string `json:"id"`
	Self string `json:"self"`
}

type CommentDetail struct {
	ID      string `json:"id"`
	Created string `json:"created"`
	Author  struct {
		DisplayName string `json:"displayName"`
	} `json:"author"`
	Body json.RawMessage `json:"body"`
}

func (c *CommentDetail) BodyText() string {
	if c.Body == nil || string(c.Body) == "null" {
		return ""
	}
	var adf map[string]interface{}
	if err := json.Unmarshal(c.Body, &adf); err != nil {
		return ""
	}
	return extractADFText(adf)
}

func (c *Client) GetComments(issueKey string) ([]CommentDetail, error) {
	body, _, err := c.do("GET", "/rest/api/3/issue/"+issueKey+"/comment?orderBy=created", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Comments []CommentDetail `json:"comments"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return resp.Comments, nil
}

func (c *Client) AddComment(issueKey, text string) (*Comment, error) {
	body := map[string]interface{}{
		"body": toADF(text),
	}

	respBody, _, err := c.do("POST", "/rest/api/3/issue/"+issueKey+"/comment", body)
	if err != nil {
		return nil, err
	}

	var comment Comment
	if err := json.Unmarshal(respBody, &comment); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &comment, nil
}
