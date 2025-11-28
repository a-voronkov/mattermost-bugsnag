package bugsnag

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

// DefaultBaseURL is the standard Bugsnag API endpoint.
const DefaultBaseURL = "https://api.bugsnag.com"

// DefaultTimeout is the default HTTP timeout for Bugsnag API requests.
const DefaultTimeout = 10 * time.Second

// Client wraps authenticated access to the Bugsnag REST API.
type Client struct {
	BaseURL    *url.URL
	Token      string
	HTTPClient *http.Client
}

// Organization represents a Bugsnag organization.
type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Project represents a Bugsnag project.
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

// ErrorDetails represents detailed information about a Bugsnag error.
type ErrorDetails struct {
	ID            string `json:"id"`
	ProjectID     string `json:"project_id"`
	ErrorClass    string `json:"error_class"`
	Message       string `json:"message"`
	Context       string `json:"context"`
	Status        string `json:"status"`
	Severity      string `json:"severity"`
	Events        int    `json:"events"`
	EventsLast24h int    `json:"events_last_24h,omitempty"`
	FirstSeen     string `json:"first_seen"`
	LastSeen      string `json:"last_seen"`
	AssigneeID    string `json:"assignee_id,omitempty"`
	URL           string `json:"url,omitempty"`
}

// Collaborator represents a user with access to a Bugsnag organization.
type Collaborator struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
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

// GetOrganizations retrieves all organizations accessible by the current user.
func (c *Client) GetOrganizations(ctx context.Context) ([]Organization, error) {
	var orgs []Organization
	if err := c.do(ctx, http.MethodGet, "/user/organizations", nil, &orgs); err != nil {
		return nil, err
	}

	return orgs, nil
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

// GetCollaborators retrieves all users (collaborators) for the given organization.
func (c *Client) GetCollaborators(ctx context.Context, orgID string) ([]Collaborator, error) {
	endpoint := fmt.Sprintf("/organizations/%s/collaborators", url.PathEscape(orgID))

	var collaborators []Collaborator
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &collaborators); err != nil {
		return nil, err
	}

	return collaborators, nil
}

// GetError retrieves detailed information about a specific error.
func (c *Client) GetError(ctx context.Context, projectID, errorID string) (*ErrorDetails, error) {
	endpoint := fmt.Sprintf("/projects/%s/errors/%s", url.PathEscape(projectID), url.PathEscape(errorID))

	var details ErrorDetails
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &details); err != nil {
		return nil, err
	}

	return &details, nil
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

// UpdateProjectErrorStatus changes the error status (e.g., "open", "fixed", "ignored")
// using the project-scoped endpoint.
func (c *Client) UpdateProjectErrorStatus(ctx context.Context, projectID, errorID, status string) error {
	if status == "" {
		return fmt.Errorf("status is required")
	}

	payload := map[string]string{
		"status": status,
	}

	endpoint := fmt.Sprintf("/projects/%s/errors/%s", url.PathEscape(projectID), url.PathEscape(errorID))
	return c.do(ctx, http.MethodPatch, endpoint, payload, nil)
}

// AssignError assigns a Bugsnag error to a user. The assignee can be either
// a user ID or email depending on the installation settings.
func (c *Client) AssignError(ctx context.Context, projectID, errorID, assignee string) error {
	if assignee == "" {
		return fmt.Errorf("assignee is required")
	}

	payload := map[string]string{
		"assignee_id": assignee,
	}

	endpoint := fmt.Sprintf("/projects/%s/errors/%s/assignee", url.PathEscape(projectID), url.PathEscape(errorID))
	return c.do(ctx, http.MethodPut, endpoint, payload, nil)
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

// NewDefaultClient creates a client with the default Bugsnag API URL and timeout.
func NewDefaultClient(token string) (*Client, error) {
	return NewClient(DefaultBaseURL, token, &http.Client{Timeout: DefaultTimeout})
}

// UserMapping connects a Mattermost user to a Bugsnag user record.
type UserMapping struct {
	BugsnagUserID string
	BugsnagEmail  string
}

// BestAssignee returns the most precise Bugsnag identity to use for assignment.
// It prefers explicit Bugsnag user ID, falling back to email.
func BestAssignee(mapping UserMapping) string {
	if id := strings.TrimSpace(mapping.BugsnagUserID); id != "" {
		return id
	}
	return strings.TrimSpace(mapping.BugsnagEmail)
}
