package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/mock"
)

func TestHandleWebhookMethodNotAllowed(t *testing.T) {
	api := &plugintest.API{}
	p := &Plugin{}
	p.SetAPI(api)

	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	rr := httptest.NewRecorder()

	p.handleWebhook(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestHandleWebhookMissingToken(t *testing.T) {
	api := &plugintest.API{}
	api.On("LogWarn", "webhook rejected", "err", "missing webhook token", "remote", mock.Anything).Return()

	p := &Plugin{}
	p.SetAPI(api)
	p.configuration.Store(&Configuration{WebhookToken: "secret-token"})

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	rr := httptest.NewRecorder()

	p.handleWebhook(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandleWebhookInvalidToken(t *testing.T) {
	api := &plugintest.API{}
	api.On("LogWarn", "webhook rejected", "err", "invalid webhook token", "remote", mock.Anything).Return()

	p := &Plugin{}
	p.SetAPI(api)
	p.configuration.Store(&Configuration{WebhookToken: "secret-token"})

	req := httptest.NewRequest(http.MethodPost, "/webhook?token=wrong-token", nil)
	rr := httptest.NewRecorder()

	p.handleWebhook(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandleWebhookValidToken(t *testing.T) {
	api := &plugintest.API{}
	api.On("LogInfo", "received webhook", "remote", mock.Anything).Return()
	api.On("LogDebug", "kv namespace initialized", "namespace", pluginID).Return().Maybe()
	api.On("KVGet", pluginID+":"+KVKeyProjectChannelMappings).Return(nil, nil)

	p := &Plugin{}
	p.SetAPI(api)
	p.kvNamespace = pluginID
	p.configuration.Store(&Configuration{WebhookToken: "secret-token"})

	payload := webhookPayload{
		Trigger: triggerInfo{Type: "error"},
		Error:   &errorInfo{ID: "err-123", ExceptionClass: "Test error"},
		Project: &projectInfo{ID: "proj-1"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook?token=secret-token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	p.handleWebhook(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d: %s", http.StatusAccepted, rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "accepted" {
		t.Fatalf("expected status accepted, got %v", resp["status"])
	}
}

func TestHandleWebhookWithChannelID(t *testing.T) {
	channelID := "channel-123"
	postID := "post-456"

	api := &plugintest.API{}
	api.On("LogInfo", "received webhook", "remote", mock.Anything).Return()
	api.On("KVGet", pluginID+":"+KVKeyProjectChannelMappings).Return(nil, nil)
	api.On("GetChannel", channelID).Return(&model.Channel{Id: channelID}, nil)
	api.On("KVGet", pluginID+":"+KVKeyErrorPostPrefix+"proj-1:err-123").Return(nil, nil)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: postID, ChannelId: channelID}, nil)
	api.On("KVSet", pluginID+":"+KVKeyErrorPostPrefix+"proj-1:err-123", mock.Anything).Return(nil)
	// ActiveErrors for sync scheduler
	api.On("KVGet", pluginID+":"+KVKeyActiveErrors).Return(nil, nil)
	api.On("KVSet", pluginID+":"+KVKeyActiveErrors, mock.Anything).Return(nil)
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

	p := &Plugin{}
	p.SetAPI(api)
	p.kvNamespace = pluginID
	p.configuration.Store(&Configuration{})

	payload := webhookPayload{
		Trigger: triggerInfo{Type: "error"},
		Error:   &errorInfo{ErrorID: "err-123", ExceptionClass: "Test error"},
		Project: &projectInfo{ID: "proj-1"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook?channel_id="+channelID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	p.handleWebhook(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d: %s", http.StatusAccepted, rr.Code, rr.Body.String())
	}
}

func TestValidateWebhookToken(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Configuration
		queryToken  string
		headerToken string
		wantErr     bool
	}{
		{
			name:    "no token configured, no token provided",
			cfg:     Configuration{},
			wantErr: false,
		},
		{
			name:       "token configured, correct query token",
			cfg:        Configuration{WebhookToken: "secret"},
			queryToken: "secret",
			wantErr:    false,
		},
		{
			name:        "token configured, correct header token",
			cfg:         Configuration{WebhookToken: "secret"},
			headerToken: "secret",
			wantErr:     false,
		},
		{
			name:       "token configured, wrong token",
			cfg:        Configuration{WebhookToken: "secret"},
			queryToken: "wrong",
			wantErr:    true,
		},
		{
			name:    "token configured, no token provided",
			cfg:     Configuration{WebhookToken: "secret"},
			wantErr: true,
		},
		{
			name:       "secret fallback when token empty",
			cfg:        Configuration{WebhookSecret: "fallback-secret"},
			queryToken: "fallback-secret",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/webhook"
			if tt.queryToken != "" {
				url += "?token=" + tt.queryToken
			}
			req := httptest.NewRequest(http.MethodPost, url, nil)
			if tt.headerToken != "" {
				req.Header.Set("X-Bugsnag-Token", tt.headerToken)
			}

			err := validateWebhookToken(tt.cfg, req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateWebhookToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildCardTitle(t *testing.T) {
	tests := []struct {
		name    string
		payload webhookPayload
		want    string
	}{
		{
			name:    "with exception class only",
			payload: webhookPayload{Error: &errorInfo{ExceptionClass: "NullReferenceException"}},
			want:    ":rotating_light: **NullReferenceException**",
		},
		{
			name:    "with exception class and message",
			payload: webhookPayload{Error: &errorInfo{ExceptionClass: "NullReferenceException", Message: "Object reference not set"}},
			want:    ":rotating_light: **NullReferenceException**: Object reference not set",
		},
		{
			name:    "message only",
			payload: webhookPayload{Error: &errorInfo{Message: "Something went wrong"}},
			want:    ":rotating_light: Something went wrong",
		},
		{
			name:    "trigger message fallback",
			payload: webhookPayload{Trigger: triggerInfo{Message: "New error"}},
			want:    ":rotating_light: New error",
		},
		{
			name:    "empty payload",
			payload: webhookPayload{},
			want:    ":rotating_light: Bugsnag error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCardTitle(tt.payload)
			if got != tt.want {
				t.Errorf("buildCardTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}
