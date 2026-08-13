package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caiocesarps/jira-cli/internal/config"
)

type Client struct {
	baseURL    string
	email      string
	apiToken   string
	httpClient *http.Client
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("jira API error (%d): %s", e.StatusCode, e.Body)
}

func NewClient(profile *config.Profile) *Client {
	return &Client{
		baseURL:  strings.TrimRight(profile.BaseURL, "/"),
		email:    profile.Email,
		apiToken: profile.APIToken,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) do(method, path string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, &APIError{
			StatusCode: resp.StatusCode,
			Body:       extractErrorMessage(respBody),
		}
	}

	return respBody, resp.StatusCode, nil
}

// doMultipart uploads one or more local files to path as a multipart/form-data
// request, using field name "file" per Jira's attachments API. It sets the
// X-Atlassian-Token header Jira requires to bypass XSRF checks on this endpoint.
func (c *Client) doMultipart(method, path string, filePaths []string) ([]byte, int, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for _, filePath := range filePaths {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, 0, fmt.Errorf("open file %s: %w", filePath, err)
		}

		part, err := writer.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			f.Close()
			return nil, 0, fmt.Errorf("create form field for %s: %w", filePath, err)
		}
		if _, err := io.Copy(part, f); err != nil {
			f.Close()
			return nil, 0, fmt.Errorf("read file %s: %w", filePath, err)
		}
		f.Close()
	}

	if err := writer.Close(); err != nil {
		return nil, 0, fmt.Errorf("finalize multipart body: %w", err)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, &buf)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Atlassian-Token", "no-check")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, &APIError{
			StatusCode: resp.StatusCode,
			Body:       extractErrorMessage(respBody),
		}
	}

	return respBody, resp.StatusCode, nil
}

func extractErrorMessage(body []byte) string {
	var errResp struct {
		ErrorMessages []string          `json:"errorMessages"`
		Errors        map[string]string `json:"errors"`
	}
	if err := json.Unmarshal(body, &errResp); err != nil {
		return string(body)
	}
	if len(errResp.ErrorMessages) > 0 {
		return strings.Join(errResp.ErrorMessages, "; ")
	}
	parts := make([]string, 0, len(errResp.Errors))
	for k, v := range errResp.Errors {
		parts = append(parts, k+": "+v)
	}
	if len(parts) > 0 {
		return strings.Join(parts, "; ")
	}
	return string(body)
}
