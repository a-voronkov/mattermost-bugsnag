package main

import (
	"encoding/json"
	"net/http"
)

type actionContext struct {
	Action    string `json:"action"`
	ErrorID   string `json:"error_id"`
	ProjectID string `json:"project_id"`
	ErrorURL  string `json:"error_url,omitempty"`
}

type interactiveAction struct {
	UserID  string        `json:"user_id"`
	Context actionContext `json:"context"`
}

func (p *Plugin) handleActions(w http.ResponseWriter, r *http.Request) {
	var payload interactiveAction
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid interactive action payload", http.StatusBadRequest)
		return
	}

	// TODO: map Mattermost user → Bugsnag user and execute requested action via Bugsnag API.
	p.API.LogInfo("received interactive action", "action", payload.Context.Action, "error_id", payload.Context.ErrorID, "project_id", payload.Context.ProjectID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"text": "Action queued; full implementation pending.",
	})
}
