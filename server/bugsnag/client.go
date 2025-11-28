package bugsnag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
)

// Client wraps authenticated access to the Bugsnag REST API.
type Client struct {
	BaseURL    *url.URL
	Token      string
	HTTPClient *http.Client
}

type Project struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	OrganizationID string `json:"organization_id"`
}

// ErrorStatus represents the status of a Bugsnag error.
type ErrorStatus struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	AssigneeID string `json:"assignee_id,omitempty"`
}

// NewClient constructs a Client instance.
func NewClient(rawBaseURL, token string, httpClient *http.Client) (*Client, error) {
	if rawBaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	baseURL, err := url.Parse(rawBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}

	if token == "" {
		return nil, fmt.Errorf("token is required")
	}

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{BaseURL: baseURL, Token: token, HTTPClient: httpClient}, nil
}

// GetProjects retrieves all projects for the given organization.
func (c *Client) GetProjects(ctx context.Context, orgID string) ([]Project, error) {
	endpoint := fmt.Sprintf("/organizations/%s/projects", url.PathEscape(orgID))

	var projects []Project
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

// UpdateErrorStatus updates a Bugsnag error status and optional assignee.
func (c *Client) UpdateErrorStatus(ctx context.Context, errorID, status, assignee string) (*ErrorStatus, error) {
	payload := map[string]string{
		"status": status,
	}
	if assignee != "" {
		payload["assignee_id"] = assignee
	}

	endpoint := fmt.Sprintf("/errors/%s", url.PathEscape(errorID))

	var result ErrorStatus
	if err := c.do(ctx, http.MethodPatch, endpoint, payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) do(ctx context.Context, method, endpoint string, body any, out any) error {
	if c == nil {
		return fmt.Errorf("client is nil")
	}
	if c.BaseURL == nil {
		return fmt.Errorf("base URL is not configured")
	}
	if c.HTTPClient == nil {
		return fmt.Errorf("http client is not configured")
	}

	relPath := path.Clean(endpoint)
	resolved := c.BaseURL.ResolveReference(&url.URL{Path: relPath})

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, resolved.String(), &buf)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.Token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("bugsnag API returned status %d", resp.StatusCode)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
