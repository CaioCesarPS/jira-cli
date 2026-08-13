package api

import (
	"encoding/json"
	"fmt"
)

type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

// AttachFiles uploads one or more local files as attachments on issueKey.
// Jira's attachments endpoint returns a JSON array of the created attachments.
func (c *Client) AttachFiles(issueKey string, filePaths []string) ([]Attachment, error) {
	body, _, err := c.doMultipart("POST", "/rest/api/3/issue/"+issueKey+"/attachments", filePaths)
	if err != nil {
		return nil, err
	}

	var attachments []Attachment
	if err := json.Unmarshal(body, &attachments); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return attachments, nil
}
