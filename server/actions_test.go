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

func TestHandleActionsMethodNotAllowed(t *testing.T) {
	api := &plugintest.API{}
	p := &Plugin{}
	p.SetAPI(api)

	req := httptest.NewRequest(http.MethodGet, "/actions", nil)
	rr := httptest.NewRecorder()

	p.handleActions(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestHandleActionsInvalidPayload(t *testing.T) {
	api := &plugintest.API{}
	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return()
	p := &Plugin{}
	p.SetAPI(api)

	req := httptest.NewRequest(http.MethodPost, "/actions", bytes.NewReader([]byte("invalid json")))
	rr := httptest.NewRecorder()

	p.handleActions(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleActionsMissingAction(t *testing.T) {
	api := &plugintest.API{}
	api.On("LogInfo", "received interactive action", "user_id", "user-123", "action", "", "error_id", "", "project_id", "").Return()

	p := &Plugin{}
	p.SetAPI(api)

	payload := model.PostActionIntegrationRequest{
		UserId:  "user-123",
		Context: map[string]any{}, // empty action
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/actions", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	p.handleActions(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleActionsInvalidUser(t *testing.T) {
	api := &plugintest.API{}
	api.On("LogInfo", "received interactive action", "user_id", "unknown-user", "action", "resolve", "error_id", "err-123", "project_id", "proj-1").Return()
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	api.On("KVGet", mock.Anything).Return(nil, nil).Maybe()
	api.On("GetUser", "unknown-user").Return(nil, model.NewAppError("GetUser", "user not found", nil, "", http.StatusNotFound))

	p := &Plugin{}
	p.SetAPI(api)
	p.kvNamespace = pluginID
	p.configuration.Store(&Configuration{})

	payload := model.PostActionIntegrationRequest{
		UserId: "unknown-user",
		Context: map[string]any{
			"action":     "resolve",
			"error_id":   "err-123",
			"project_id": "proj-1",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/actions", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	p.handleActions(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleActionsResolveNoClient(t *testing.T) {
	userID := "user-123"

	api := &plugintest.API{}
	api.On("LogInfo", "received interactive action", "user_id", userID, "action", "resolve", "error_id", "err-123", "project_id", "proj-1").Return()
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	// Return empty array for user mappings
	api.On("KVGet", pluginID+":"+KVKeyUserMappings).Return([]byte("[]"), (*model.AppError)(nil)).Maybe()
	// Return nil for error post mapping (not found)
	api.On("KVGet", mock.Anything).Return(nil, (*model.AppError)(nil)).Maybe()
	api.On("GetUser", userID).Return(&model.User{Id: userID, Username: "testuser", Email: "test@example.com"}, (*model.AppError)(nil))

	p := &Plugin{}
	p.SetAPI(api)
	p.kvNamespace = pluginID
	p.configuration.Store(&Configuration{}) // no Bugsnag token

	payload := model.PostActionIntegrationRequest{
		UserId: userID,
		Context: map[string]any{
			"action":     "resolve",
			"error_id":   "err-123",
			"project_id": "proj-1",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/actions", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	p.handleActions(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d: %s", http.StatusAccepted, rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["text"] == "" {
		t.Fatal("expected non-empty text in response")
	}
}

func TestHandleActionsUnsupportedAction(t *testing.T) {
	userID := "user-123"

	api := &plugintest.API{}
	api.On("LogInfo", "received interactive action", "user_id", userID, "action", "unknown_action", "error_id", "", "project_id", "").Return()
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	api.On("GetUser", userID).Return(&model.User{Id: userID, Username: "testuser"}, nil)
	api.On("KVGet", mock.Anything).Return(nil, nil).Maybe()

	p := &Plugin{}
	p.SetAPI(api)
	p.kvNamespace = pluginID
	p.configuration.Store(&Configuration{})

	payload := model.PostActionIntegrationRequest{
		UserId: userID,
		Context: map[string]any{
			"action": "unknown_action",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/actions", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	p.handleActions(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}
