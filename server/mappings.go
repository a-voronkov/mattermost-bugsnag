package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
)

const (
	kvProjectChannelMapKey = "bugsnag:project-channel-mappings"
	kvUserMappingsKey      = "bugsnag:user-mappings"
)

// ChannelRule describes where to send a Bugsnag event for a given project, and
// what filters must match before posting.
type ChannelRule struct {
	ChannelID    string   `json:"channel_id"`
	Environments []string `json:"environments,omitempty"`
	Severities   []string `json:"severities,omitempty"`
	Events       []string `json:"events,omitempty"`
}

// ProjectChannelMappings is a map keyed by Bugsnag project ID, with one or more
// channel rules per project.
type ProjectChannelMappings map[string][]ChannelRule

// ErrorPostMapping stores where a specific Bugsnag error was posted in
// Mattermost so subsequent webhook deliveries can update the same card.
type ErrorPostMapping struct {
	ProjectID string `json:"project_id"`
	ErrorID   string `json:"error_id"`
	ChannelID string `json:"channel_id"`
	PostID    string `json:"post_id"`
}

// UserMapping connects a Mattermost user to a Bugsnag user record (by explicit
// ID or by email). Either BugsnagUserID or BugsnagEmail can be set; Mattermost
// lookups first match MMUserID, then fallback to email matching.
type UserMapping struct {
	BugsnagUserID string `json:"bugsnag_user_id,omitempty"`
	BugsnagEmail  string `json:"bugsnag_email,omitempty"`
	MMUserID      string `json:"mm_user_id,omitempty"`
}

// loadUserMappings reads the Bugsnag↔Mattermost user associations from KV. The
// admin UI will be responsible for writing this structure.
func loadUserMappings(mm *MMClient) ([]UserMapping, error) {
	var mappings []UserMapping
	found, appErr := mm.LoadJSON(kvUserMappingsKey, &mappings)
	if appErr != nil {
		return nil, fmt.Errorf("load user mappings: %w", appErr)
	}
	if !found {
		return []UserMapping{}, nil
	}

	return mappings, nil
}

func loadProjectChannelMappings(mm *MMClient) (ProjectChannelMappings, error) {
	var mappings ProjectChannelMappings
	found, appErr := mm.LoadJSON(kvProjectChannelMapKey, &mappings)
	if appErr != nil {
		return nil, fmt.Errorf("load project/channel mappings: %w", appErr)
	}
	if !found {
		return ProjectChannelMappings{}, nil
	}
	return mappings, nil
}

func matchesRule(rule ChannelRule, payload webhookPayload) bool {
	if len(rule.Environments) > 0 && !containsValue(rule.Environments, payload.Environment) {
		return false
	}

	if len(rule.Severities) > 0 && !containsValue(rule.Severities, payload.Severity) {
		return false
	}

	if len(rule.Events) > 0 && !containsValue(rule.Events, payload.Event) {
		return false
	}

	return true
}

func containsValue(values []string, candidate string) bool {
	candidate = strings.TrimSpace(strings.ToLower(candidate))
	for _, v := range values {
		if strings.TrimSpace(strings.ToLower(v)) == candidate {
			return true
		}
	}
	return false
}

func errorPostKVKey(projectID, errorID string) string {
	return fmt.Sprintf("bugsnag:error-post:%s:%s", projectID, errorID)
}

// mapUserToBugsnag resolves a Mattermost user to a Bugsnag user based on the
// configured mappings. First it matches by MMUserID; if absent, it falls back
// to case-insensitive email matching.
func mapUserToBugsnag(mappings []UserMapping, user *model.User) (UserMapping, bool) {
	if user == nil {
		return UserMapping{}, false
	}

	for _, m := range mappings {
		if strings.TrimSpace(m.MMUserID) != "" && m.MMUserID == user.Id {
			return m, true
		}
	}

	userEmail := strings.TrimSpace(strings.ToLower(user.Email))
	if userEmail == "" {
		return UserMapping{}, false
	}

	for _, m := range mappings {
		if strings.TrimSpace(strings.ToLower(m.BugsnagEmail)) == userEmail {
			return m, true
		}
	}

	return UserMapping{}, false
}
