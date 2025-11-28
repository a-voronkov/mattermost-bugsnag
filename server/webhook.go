package main

import (
	"encoding/json"
	"net/http"
)

// webhookPayload is a light struct to keep the handler focused on routing.
// Full payload samples live in docs/sample-payloads.md.
type webhookPayload struct {
	Event     string `json:"event"`
	ErrorID   string `json:"error_id"`
	ProjectID string `json:"project_id"`
	// Additional fields from Bugsnag can be added as needed.
}

func (p *Plugin) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// TODO: validate webhook secret/token from query or header.
	cfg := p.getConfiguration()
	_ = cfg // placeholder until validation is implemented

	var payload webhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	// TODO: map project to channels and decide whether to create/update a post.
	p.API.LogInfo("received Bugsnag webhook", "event", payload.Event, "error_id", payload.ErrorID, "project_id", payload.ProjectID)

	// Placeholder response to keep Bugsnag happy while the full workflow is developed.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "accepted",
		"message": "webhook received; posting logic pending",
	})
}
