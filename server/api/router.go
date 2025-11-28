package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/a-voronkov/mattermost-bugsnag/server/bugsnag"
)

// UserMapping connects a Mattermost user to a Bugsnag user.
type UserMapping struct {
	MattermostUserID   string `json:"mattermost_user_id"`
	MattermostUsername string `json:"mattermost_username,omitempty"`
	BugsnagUserID      string `json:"bugsnag_user_id,omitempty"`
	BugsnagEmail       string `json:"bugsnag_email,omitempty"`
}

// ChannelRule describes where to send a Bugsnag event for a given project.
type ChannelRule struct {
	ID           string   `json:"id"`
	ProjectID    string   `json:"project_id"`
	ProjectName  string   `json:"project_name,omitempty"`
	ChannelID    string   `json:"channel_id"`
	ChannelName  string   `json:"channel_name,omitempty"`
	Environments []string `json:"environments,omitempty"`
	Severities   []string `json:"severities,omitempty"`
	Events       []string `json:"events,omitempty"`
}

// KVStore defines the minimal operations needed for API storage.
type KVStore interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
}

// Config holds the configuration providers for the API router.
type Config struct {
	TokenProvider func() string
	OrgIDProvider func() string
	KVStore       KVStore
}

// Router handles all /api/v1/* endpoints.
type Router struct {
	config Config
}

// NewRouter creates a new API router with the given configuration.
func NewRouter(config Config) *Router {
	return &Router{config: config}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/api/v1")

	switch {
	case path == "/test":
		r.handleTest(w, req)
	case path == "/projects":
		r.handleProjects(w, req)
	case path == "/organizations":
		r.handleOrganizations(w, req)
	case path == "/collaborators":
		r.handleCollaborators(w, req)
	case path == "/user-mappings":
		r.handleUserMappings(w, req)
	case path == "/channel-rules":
		r.handleChannelRules(w, req)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (r *Router) handleTest(w http.ResponseWriter, req *http.Request) {
	handler := NewHandlerWithOrgID(r.config.TokenProvider, r.config.OrgIDProvider)
	handler.ServeHTTP(w, req)
}

func (r *Router) handleProjects(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := strings.TrimSpace(r.config.TokenProvider())
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing Bugsnag API token")
		return
	}

	client, err := bugsnag.NewDefaultClient(token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create Bugsnag client: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(req.Context(), 15*time.Second)
	defer cancel()

	// Get organization ID from query or config
	orgID := req.URL.Query().Get("organization_id")
	if orgID == "" {
		orgID = strings.TrimSpace(r.config.OrgIDProvider())
	}

	if orgID == "" {
		// Try to get first organization
		orgs, err := client.GetOrganizations(ctx)
		if err != nil {
			writeError(w, http.StatusBadGateway, "failed to fetch organizations: "+err.Error())
			return
		}
		if len(orgs) == 0 {
			writeJSON(w, http.StatusOK, map[string]any{
				"projects": []any{},
				"message":  "No organizations found",
			})
			return
		}
		orgID = orgs[0].ID
	}

	projects, err := client.GetProjects(ctx, orgID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch projects: "+err.Error())
		return
	}

	type projectResponse struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	result := make([]projectResponse, len(projects))
	for i, p := range projects {
		result[i] = projectResponse{ID: p.ID, Name: p.Name}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"organization_id": orgID,
		"projects":        result,
	})
}

func (r *Router) handleOrganizations(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := strings.TrimSpace(r.config.TokenProvider())
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing Bugsnag API token")
		return
	}

	client, err := bugsnag.NewDefaultClient(token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create Bugsnag client: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()

	orgs, err := client.GetOrganizations(ctx)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch organizations: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"organizations": orgs,
	})
}

func (r *Router) handleCollaborators(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := r.config.TokenProvider()
	if token == "" {
		writeError(w, http.StatusUnauthorized, "Bugsnag API token not configured")
		return
	}

	client, err := bugsnag.NewDefaultClient(token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create Bugsnag client: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()

	// Get organization ID from config or fetch first org
	orgID := ""
	if r.config.OrgIDProvider != nil {
		orgID = strings.TrimSpace(r.config.OrgIDProvider())
	}

	if orgID == "" {
		orgs, err := client.GetOrganizations(ctx)
		if err != nil {
			writeError(w, http.StatusBadGateway, "failed to fetch organizations: "+err.Error())
			return
		}
		if len(orgs) == 0 {
			writeJSON(w, http.StatusOK, map[string]any{
				"collaborators": []any{},
				"message":       "No organizations found",
			})
			return
		}
		orgID = orgs[0].ID
	}

	collaborators, err := client.GetCollaborators(ctx, orgID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch collaborators: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"collaborators":   collaborators,
		"organization_id": orgID,
	})
}

const kvKeyUserMappings = "bugsnag:user-mappings"

func (r *Router) handleUserMappings(w http.ResponseWriter, req *http.Request) {
	if r.config.KVStore == nil {
		writeError(w, http.StatusInternalServerError, "storage not configured")
		return
	}

	switch req.Method {
	case http.MethodGet:
		r.getUserMappings(w)
	case http.MethodPost, http.MethodPut:
		r.saveUserMappings(w, req)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (r *Router) getUserMappings(w http.ResponseWriter) {
	data, err := r.config.KVStore.Get(kvKeyUserMappings)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load user mappings: "+err.Error())
		return
	}

	var mappings []UserMapping
	if len(data) > 0 {
		if err := json.Unmarshal(data, &mappings); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to parse user mappings: "+err.Error())
			return
		}
	}

	if mappings == nil {
		mappings = []UserMapping{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"mappings": mappings,
	})
}

func (r *Router) saveUserMappings(w http.ResponseWriter, req *http.Request) {
	var payload struct {
		Mappings []UserMapping `json:"mappings"`
	}

	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	data, err := json.Marshal(payload.Mappings)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode mappings: "+err.Error())
		return
	}

	if err := r.config.KVStore.Set(kvKeyUserMappings, data); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save mappings: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"mappings": payload.Mappings,
	})
}

const kvKeyChannelRules = "bugsnag:project-channel-mappings"

func (r *Router) handleChannelRules(w http.ResponseWriter, req *http.Request) {
	if r.config.KVStore == nil {
		writeError(w, http.StatusInternalServerError, "storage not configured")
		return
	}

	switch req.Method {
	case http.MethodGet:
		r.getChannelRules(w)
	case http.MethodPost, http.MethodPut:
		r.saveChannelRules(w, req)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (r *Router) getChannelRules(w http.ResponseWriter) {
	data, err := r.config.KVStore.Get(kvKeyChannelRules)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load channel rules: "+err.Error())
		return
	}

	var rules []ChannelRule
	if len(data) > 0 {
		if err := json.Unmarshal(data, &rules); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to parse channel rules: "+err.Error())
			return
		}
	}

	if rules == nil {
		rules = []ChannelRule{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"rules": rules,
	})
}

func (r *Router) saveChannelRules(w http.ResponseWriter, req *http.Request) {
	var payload struct {
		Rules []ChannelRule `json:"rules"`
	}

	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	data, err := json.Marshal(payload.Rules)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode channel rules: "+err.Error())
		return
	}

	if err := r.config.KVStore.Set(kvKeyChannelRules, data); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save channel rules: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"rules":  payload.Rules,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
