package main

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestMatchesRule(t *testing.T) {
	tests := []struct {
		name    string
		rule    ChannelRule
		payload webhookPayload
		want    bool
	}{
		{
			name:    "empty rule matches any payload",
			rule:    ChannelRule{},
			payload: webhookPayload{Environment: "production", Severity: "error", Event: "error"},
			want:    true,
		},
		{
			name:    "environment filter matches",
			rule:    ChannelRule{Environments: []string{"production"}},
			payload: webhookPayload{Environment: "production"},
			want:    true,
		},
		{
			name:    "environment filter does not match",
			rule:    ChannelRule{Environments: []string{"production"}},
			payload: webhookPayload{Environment: "staging"},
			want:    false,
		},
		{
			name:    "severity filter matches",
			rule:    ChannelRule{Severities: []string{"error", "warning"}},
			payload: webhookPayload{Severity: "error"},
			want:    true,
		},
		{
			name:    "severity filter does not match",
			rule:    ChannelRule{Severities: []string{"error"}},
			payload: webhookPayload{Severity: "info"},
			want:    false,
		},
		{
			name:    "event filter matches",
			rule:    ChannelRule{Events: []string{"error", "spike"}},
			payload: webhookPayload{Event: "spike"},
			want:    true,
		},
		{
			name:    "event filter does not match",
			rule:    ChannelRule{Events: []string{"error"}},
			payload: webhookPayload{Event: "spike"},
			want:    false,
		},
		{
			name:    "multiple filters all match",
			rule:    ChannelRule{Environments: []string{"production"}, Severities: []string{"error"}, Events: []string{"error"}},
			payload: webhookPayload{Environment: "production", Severity: "error", Event: "error"},
			want:    true,
		},
		{
			name:    "multiple filters one fails",
			rule:    ChannelRule{Environments: []string{"production"}, Severities: []string{"error"}},
			payload: webhookPayload{Environment: "staging", Severity: "error"},
			want:    false,
		},
		{
			name:    "case insensitive matching",
			rule:    ChannelRule{Environments: []string{"Production"}},
			payload: webhookPayload{Environment: "production"},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesRule(tt.rule, tt.payload)
			if got != tt.want {
				t.Errorf("matchesRule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapUserToBugsnag(t *testing.T) {
	tests := []struct {
		name     string
		mappings []UserMapping
		user     *model.User
		wantMap  UserMapping
		wantOk   bool
	}{
		{
			name:     "nil user returns not found",
			mappings: []UserMapping{{MMUserID: "user-1", BugsnagUserID: "bs-user-1"}},
			user:     nil,
			wantMap:  UserMapping{},
			wantOk:   false,
		},
		{
			name:     "match by MM user ID",
			mappings: []UserMapping{{MMUserID: "user-1", BugsnagUserID: "bs-user-1"}},
			user:     &model.User{Id: "user-1", Email: "other@example.com"},
			wantMap:  UserMapping{MMUserID: "user-1", BugsnagUserID: "bs-user-1"},
			wantOk:   true,
		},
		{
			name:     "match by email fallback",
			mappings: []UserMapping{{BugsnagEmail: "test@example.com", BugsnagUserID: "bs-user-2"}},
			user:     &model.User{Id: "user-2", Email: "test@example.com"},
			wantMap:  UserMapping{BugsnagEmail: "test@example.com", BugsnagUserID: "bs-user-2"},
			wantOk:   true,
		},
		{
			name:     "email match is case insensitive",
			mappings: []UserMapping{{BugsnagEmail: "Test@Example.com", BugsnagUserID: "bs-user-3"}},
			user:     &model.User{Id: "user-3", Email: "test@example.com"},
			wantMap:  UserMapping{BugsnagEmail: "Test@Example.com", BugsnagUserID: "bs-user-3"},
			wantOk:   true,
		},
		{
			name:     "no matching mapping",
			mappings: []UserMapping{{MMUserID: "user-1", BugsnagUserID: "bs-user-1"}},
			user:     &model.User{Id: "user-99", Email: "other@example.com"},
			wantMap:  UserMapping{},
			wantOk:   false,
		},
		{
			name:     "empty mappings returns not found",
			mappings: []UserMapping{},
			user:     &model.User{Id: "user-1", Email: "test@example.com"},
			wantMap:  UserMapping{},
			wantOk:   false,
		},
		{
			name:     "user with empty email, no ID match",
			mappings: []UserMapping{{BugsnagEmail: "test@example.com"}},
			user:     &model.User{Id: "user-1", Email: ""},
			wantMap:  UserMapping{},
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMap, gotOk := mapUserToBugsnag(tt.mappings, tt.user)
			if gotOk != tt.wantOk {
				t.Errorf("mapUserToBugsnag() ok = %v, want %v", gotOk, tt.wantOk)
			}
			if gotMap != tt.wantMap {
				t.Errorf("mapUserToBugsnag() map = %+v, want %+v", gotMap, tt.wantMap)
			}
		})
	}
}

func TestContainsValue(t *testing.T) {
	tests := []struct {
		values    []string
		candidate string
		want      bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{"Production"}, "production", true},
		{[]string{" error "}, "error", true},
		{[]string{}, "a", false},
	}

	for _, tt := range tests {
		got := containsValue(tt.values, tt.candidate)
		if got != tt.want {
			t.Errorf("containsValue(%v, %q) = %v, want %v", tt.values, tt.candidate, got, tt.want)
		}
	}
}

