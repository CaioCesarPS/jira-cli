package api

func (c *Client) LinkIssues(inwardKey, outwardKey, linkType string) error {
	body := map[string]interface{}{
		"type":         map[string]string{"name": linkType},
		"inwardIssue":  map[string]string{"key": inwardKey},
		"outwardIssue": map[string]string{"key": outwardKey},
	}
	_, _, err := c.do("POST", "/rest/api/3/issueLink", body)
	return err
}
