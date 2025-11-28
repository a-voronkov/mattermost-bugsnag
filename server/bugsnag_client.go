package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const defaultBugsnagAPI = "https://api.bugsnag.com"

// bugsnagClient performs authenticated HTTP requests to the Bugsnag REST API.
// It intentionally keeps the surface small (status updates, assignment) to
// support the interactive actions workflow.
type bugsnagClient struct {
	baseURL    *url.URL
	apiToken   string
	httpClient *http.Client
}

func newBugsnagClient(cfg Configuration) (*bugsnagClient, error) {
	rawBase := strings.TrimSpace(defaultBugsnagAPI)
	parsed, err := url.Parse(rawBase)
	if err != nil {
		return nil, fmt.Errorf("parse bugsnag base url: %w", err)
	}

	token := strings.TrimSpace(cfg.BugsnagAPIToken)
	if token == "" {
		return nil, fmt.Errorf("bugsnag API token is required")
	}

	return &bugsnagClient{
		baseURL:  parsed,
		apiToken: token,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// updateErrorStatus changes the error status (e.g., "open", "fixed", "ignored").
func (c *bugsnagClient) updateErrorStatus(ctx context.Context, projectID, errorID, status string) error {
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("status is required")
	}

	payload := map[string]any{
		"status": status,
	}

	endpoint := fmt.Sprintf("/projects/%s/errors/%s", url.PathEscape(projectID), url.PathEscape(errorID))
	return c.do(ctx, http.MethodPatch, endpoint, payload)
}

// assignError assigns a Bugsnag error to a user. The API is expected to accept
// either a user ID or email depending on the installation settings.
func (c *bugsnagClient) assignError(ctx context.Context, projectID, errorID, assignee string) error {
	if strings.TrimSpace(assignee) == "" {
		return fmt.Errorf("assignee is required")
	}

	payload := map[string]any{
		"assignee_id": assignee,
	}

	endpoint := fmt.Sprintf("/projects/%s/errors/%s/assignee", url.PathEscape(projectID), url.PathEscape(errorID))
	return c.do(ctx, http.MethodPut, endpoint, payload)
}

func (c *bugsnagClient) do(ctx context.Context, method, endpoint string, body any) error {
	relPath := path.Clean(endpoint)
	resolved := c.baseURL.ResolveReference(&url.URL{Path: relPath})

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, resolved.String(), &buf)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("bugsnag request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("bugsnag responded with status %d", resp.StatusCode)
	}

	return nil
}

// bestAssignee chooses the most precise Bugsnag identity to use when issuing
// an assignment request. Preference: explicit Bugsnag user ID, else email.
func bestAssignee(mapping UserMapping) string {
	if strings.TrimSpace(mapping.BugsnagUserID) != "" {
		return strings.TrimSpace(mapping.BugsnagUserID)
	}
	return strings.TrimSpace(mapping.BugsnagEmail)
}
