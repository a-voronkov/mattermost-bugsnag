package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

// webhookPayload is a light struct to keep the handler focused on routing.
// Full payload samples live in docs/sample-payloads.md.
type webhookPayload struct {
	Event       string        `json:"event"`
	ErrorID     string        `json:"error_id"`
	ProjectID   string        `json:"project_id"`
	Summary     string        `json:"summary"`
	Environment string        `json:"environment"`
	Severity    string        `json:"severity"`
	ErrorURL    string        `json:"error_url"`
	LastSeen    string        `json:"last_seen"`
	Counts      payloadCounts `json:"counts"`
	// Additional fields from Bugsnag can be added as needed.
}

type payloadCounts struct {
	Users     int `json:"users"`
	Events1h  int `json:"events_1h"`
	Events24h int `json:"events_24h"`
}

func (p *Plugin) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := p.getConfiguration()
	if err := validateWebhookToken(cfg, r); err != nil {
		p.API.LogWarn("webhook rejected", "err", err.Error(), "remote", r.RemoteAddr)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	p.API.LogInfo("received webhook", "remote", r.RemoteAddr)

	mm := newMMClient(p.API, cfg.EnableDebugLog, p.kvNS())

	var payload webhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	mappings, err := loadProjectChannelMappings(mm)
	if err != nil {
		p.API.LogError("failed to load project/channel mappings", "err", err.Error())
		http.Error(w, "cannot load channel mappings", http.StatusInternalServerError)
		return
	}

	channelRules := mappings[payload.ProjectID]
	processed := 0

	for _, rule := range channelRules {
		if !matchesRule(rule, payload) {
			continue
		}

		if err := p.upsertErrorCard(mm, rule.ChannelID, payload, cfg); err != nil {
			p.API.LogError("failed to upsert webhook card", "channel", rule.ChannelID, "error_id", payload.ErrorID, "project_id", payload.ProjectID, "err", err.Error())
			continue
		}

		processed++
	}

	// For early testing, allow an explicit channel_id query parameter to render a
	// provisional card. This will be replaced by project→channel mappings.
	channelID := strings.TrimSpace(r.URL.Query().Get("channel_id"))
	if channelID != "" {
		if _, appErr := mm.GetChannel(channelID); appErr != nil {
			http.Error(w, "invalid channel_id", http.StatusBadRequest)
			return
		}

		if err := p.upsertErrorCard(mm, channelID, payload, cfg); err != nil {
			p.API.LogError("failed to create provisional webhook post", "err", err.Error())
			http.Error(w, "failed to create post", http.StatusInternalServerError)
			return
		}

		processed++
	}

	// Placeholder response to keep Bugsnag happy while the full workflow is developed.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":    "accepted",
		"processed": processed,
	})
}

func validateWebhookToken(cfg Configuration, r *http.Request) error {
	expected := strings.TrimSpace(cfg.WebhookToken)
	if expected == "" {
		expected = strings.TrimSpace(cfg.WebhookSecret)
	}

	if expected == "" {
		return nil
	}

	provided := strings.TrimSpace(r.URL.Query().Get("token"))
	if provided == "" {
		provided = strings.TrimSpace(r.Header.Get("X-Bugsnag-Token"))
	}

	if provided == "" {
		return fmt.Errorf("missing webhook token")
	}

	if provided != expected {
		return fmt.Errorf("invalid webhook token")
	}

	return nil
}

func buildCardTitle(payload webhookPayload) string {
	switch {
	case strings.TrimSpace(payload.Summary) != "":
		return ":rotating_light: **[BUG]** " + payload.Summary
	case strings.TrimSpace(payload.ErrorID) != "":
		return ":rotating_light: **[BUG]** " + payload.ErrorID
	default:
		return ":rotating_light: Bugsnag error"
	}
}

func buildCardAttachment(payload webhookPayload, cfg Configuration) *model.SlackAttachment {
	fields := []string{}

	if payload.Environment != "" {
		fields = append(fields, fmt.Sprintf("Env: %s", payload.Environment))
	}
	if payload.Severity != "" {
		fields = append(fields, fmt.Sprintf("Severity: %s", payload.Severity))
	}
	if payload.Counts.Users > 0 {
		fields = append(fields, fmt.Sprintf("Users: %d", payload.Counts.Users))
	}
	if payload.Counts.Events1h > 0 || payload.Counts.Events24h > 0 {
		fields = append(fields, fmt.Sprintf("Events (1h/24h): %d / %d", payload.Counts.Events1h, payload.Counts.Events24h))
	}
	if payload.LastSeen != "" {
		fields = append(fields, fmt.Sprintf("Last seen: %s", payload.LastSeen))
	}

	text := strings.Join(fields, " | ")
	if payload.ProjectID != "" {
		text += fmt.Sprintf("\nProject: %s", payload.ProjectID)
	}

	footer := "Bugsnag"
	if cfg.OrganizationID != "" {
		footer = fmt.Sprintf("Bugsnag • org %s", cfg.OrganizationID)
	}

	actionURL := fmt.Sprintf("/plugins/%s/actions", pluginID)
	actions := []*model.PostAction{
		{
			Id:    "assign_me",
			Name:  "🙋 Assign to me",
			Style: "primary",
			Type:  model.PostActionTypeButton,
			Integration: &model.PostActionIntegration{
				URL: actionURL,
				Context: map[string]any{
					"action":     "assign_me",
					"error_id":   payload.ErrorID,
					"project_id": payload.ProjectID,
					"error_url":  payload.ErrorURL,
				},
			},
		},
		{
			Id:    "resolve",
			Name:  "✅ Resolve",
			Style: "primary",
			Type:  model.PostActionTypeButton,
			Integration: &model.PostActionIntegration{
				URL: actionURL,
				Context: map[string]any{
					"action":     "resolve",
					"error_id":   payload.ErrorID,
					"project_id": payload.ProjectID,
				},
			},
		},
		{
			Id:    "ignore",
			Name:  "🙈 Ignore",
			Style: "default",
			Type:  model.PostActionTypeButton,
			Integration: &model.PostActionIntegration{
				URL: actionURL,
				Context: map[string]any{
					"action":     "ignore",
					"error_id":   payload.ErrorID,
					"project_id": payload.ProjectID,
				},
			},
		},
	}

	if payload.ErrorURL != "" {
		actions = append(actions, &model.PostAction{
			Id:    "open",
			Name:  "🔗 Open in Bugsnag",
			Style: "link",
			Type:  model.PostActionTypeButton,
			Integration: &model.PostActionIntegration{
				URL: actionURL,
				Context: map[string]any{
					"action":    "open_in_browser",
					"error_url": payload.ErrorURL,
				},
			},
		})
	}

	return &model.SlackAttachment{
		Title:     payload.Summary,
		TitleLink: payload.ErrorURL,
		Text:      text,
		Footer:    footer,
		Actions:   actions,
	}
}

func (p *Plugin) upsertErrorCard(mm *MMClient, channelID string, payload webhookPayload, cfg Configuration) error {
	if strings.TrimSpace(channelID) == "" {
		return fmt.Errorf("channelID is required")
	}

	key := errorPostKVKey(payload.ProjectID, payload.ErrorID)
	var mapping ErrorPostMapping
	found, appErr := mm.LoadJSON(key, &mapping)
	if appErr != nil {
		return fmt.Errorf("load existing mapping: %w", appErr)
	}

	attachments := []*model.SlackAttachment{buildCardAttachment(payload, cfg)}
	title := buildCardTitle(payload)

	if found {
		post, appErr := mm.GetPost(mapping.PostID)
		if appErr != nil {
			return fmt.Errorf("load post: %w", appErr)
		}

		post.Message = title
		if post.Props == nil {
			post.Props = map[string]any{}
		}
		post.Props["attachments"] = attachments

		if _, appErr := mm.UpdatePost(post); appErr != nil {
			return fmt.Errorf("update post: %w", appErr)
		}

		if payload.Event != "" {
			if _, appErr := mm.CreateReply(mapping.ChannelID, mapping.PostID, fmt.Sprintf("Webhook event: %s", payload.Event)); appErr != nil {
				mm.LogDebug("failed to append webhook reply", "err", appErr.Error())
			}
		}

		return nil
	}

	post, appErr := mm.CreatePost(channelID, title, attachments)
	if appErr != nil {
		return fmt.Errorf("create post: %w", appErr)
	}

	mapping = ErrorPostMapping{
		ProjectID: payload.ProjectID,
		ErrorID:   payload.ErrorID,
		ChannelID: channelID,
		PostID:    post.Id,
	}

	if err := mm.StoreJSON(key, mapping); err != nil {
		mm.LogDebug("failed to store error→post mapping", "err", err.Error())
	}

	return nil
}
