package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/a-voronkov/mattermost-bugsnag/server/bugsnag"
)

type TestHandler struct {
	tokenProvider func() string
	orgIDProvider func() string
}

// NewHandler creates a new test endpoint handler.
func NewHandler(tokenProvider func() string) http.Handler {
	return NewHandlerWithOrgID(tokenProvider, func() string { return "" })
}

// NewHandlerWithOrgID creates a new test endpoint handler with organization ID support.
func NewHandlerWithOrgID(tokenProvider, orgIDProvider func() string) http.Handler {
	return &TestHandler{tokenProvider: tokenProvider, orgIDProvider: orgIDProvider}
}

func (h *TestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/test" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := strings.TrimSpace(h.tokenProvider())
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing Bugsnag API token")
		return
	}

	client, err := bugsnag.NewDefaultClient(token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create Bugsnag client: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// If org ID is provided, fetch projects for that org directly
	orgID := strings.TrimSpace(h.orgIDProvider())
	if orgID != "" {
		projects, err := client.GetProjects(ctx, orgID)
		if err != nil {
			writeError(w, http.StatusBadGateway, "failed to fetch projects: "+err.Error())
			return
		}

		projectNames := make([]string, len(projects))
		for i, p := range projects {
			projectNames[i] = p.Name
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":        "ok",
			"organization":  orgID,
			"project_count": len(projects),
			"projects":      projectNames,
		})
		return
	}

	// Otherwise, fetch organizations first
	orgs, err := client.GetOrganizations(ctx)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch organizations: "+err.Error())
		return
	}

	if len(orgs) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"message": "No organizations found. Check API token permissions.",
		})
		return
	}

	// Fetch projects for the first organization
	projects, err := client.GetProjects(ctx, orgs[0].ID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch projects: "+err.Error())
		return
	}

	projectNames := make([]string, len(projects))
	for i, p := range projects {
		projectNames[i] = p.Name
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":             "ok",
		"organization_count": len(orgs),
		"organization":       orgs[0].Name,
		"project_count":      len(projects),
		"projects":           projectNames,
	})
}
