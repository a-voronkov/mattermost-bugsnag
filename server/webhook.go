package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/a-voronkov/mattermost-bugsnag/server/store"
	"github.com/mattermost/mattermost/server/public/model"
)

// webhookPayload represents the full Bugsnag webhook payload.
// See https://docs.bugsnag.com/product/integrations/webhook/
type webhookPayload struct {
	Trigger triggerInfo  `json:"trigger"`
	Error   *errorInfo   `json:"error,omitempty"`
	Project *projectInfo `json:"project,omitempty"`
	Account *accountInfo `json:"account,omitempty"`
	Release *releaseInfo `json:"release,omitempty"`
}

type triggerInfo struct {
	Type        string `json:"type"`
	Message     string `json:"message"`
	Rate        int    `json:"rate,omitempty"`
	StateChange string `json:"stateChange,omitempty"`
}

type projectInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type accountInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type errorInfo struct {
	ID                   string          `json:"id"`
	ErrorID              string          `json:"errorId"`
	ExceptionClass       string          `json:"exceptionClass"`
	Message              string          `json:"message"`
	Context              string          `json:"context"`
	FirstReceived        string          `json:"firstReceived"`
	ReceivedAt           string          `json:"receivedAt"`
	RequestURL           string          `json:"requestUrl,omitempty"`
	URL                  string          `json:"url"`
	Severity             string          `json:"severity"`
	Status               string          `json:"status"`
	Unhandled            bool            `json:"unhandled"`
	AssignedCollaborator *collaborator   `json:"assigned_collaborator,omitempty"`
	App                  *appInfo        `json:"app,omitempty"`
	Device               *deviceInfo     `json:"device,omitempty"`
	User                 *userInfo       `json:"user,omitempty"`
	Exceptions           []exceptionInfo `json:"exceptions,omitempty"`
	StackTrace           []stackFrame    `json:"stackTrace,omitempty"` // deprecated but still sent
}

type collaborator struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type appInfo struct {
	ID           string `json:"id,omitempty"`
	Version      string `json:"version,omitempty"`
	ReleaseStage string `json:"releaseStage,omitempty"`
	Type         string `json:"type,omitempty"`
}

type deviceInfo struct {
	Hostname       string `json:"hostname,omitempty"`
	OSName         string `json:"osName,omitempty"`
	OSVersion      string `json:"osVersion,omitempty"`
	BrowserName    string `json:"browserName,omitempty"`
	BrowserVersion string `json:"browserVersion,omitempty"`
}

type userInfo struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type exceptionInfo struct {
	ErrorClass string       `json:"errorClass"`
	Message    string       `json:"message"`
	Type       string       `json:"type,omitempty"`
	Stacktrace []stackFrame `json:"stacktrace,omitempty"`
}

type stackFrame struct {
	InProject    bool              `json:"inProject"`
	LineNumber   interface{}       `json:"lineNumber,omitempty"` // can be int or string
	ColumnNumber interface{}       `json:"columnNumber,omitempty"`
	File         string            `json:"file,omitempty"`
	Method       string            `json:"method,omitempty"`
	Code         map[string]string `json:"code,omitempty"`
}

type releaseInfo struct {
	ID           string `json:"id"`
	Version      string `json:"version"`
	ReleaseStage string `json:"releaseStage"`
	URL          string `json:"url"`
}

// Helper methods to extract common fields from the nested payload structure
func (p webhookPayload) getProjectID() string {
	if p.Project != nil {
		return p.Project.ID
	}
	return ""
}

func (p webhookPayload) getProjectName() string {
	if p.Project != nil {
		return p.Project.Name
	}
	return ""
}

func (p webhookPayload) getErrorID() string {
	if p.Error != nil {
		return p.Error.ErrorID
	}
	return ""
}

func (p webhookPayload) getErrorURL() string {
	if p.Error != nil {
		return p.Error.URL
	}
	return ""
}

func (p webhookPayload) getSeverity() string {
	if p.Error != nil {
		return p.Error.Severity
	}
	return ""
}

func (p webhookPayload) getEnvironment() string {
	if p.Error != nil && p.Error.App != nil {
		return p.Error.App.ReleaseStage
	}
	return ""
}

func (p webhookPayload) getExceptionClass() string {
	if p.Error != nil {
		return p.Error.ExceptionClass
	}
	return ""
}

func (p webhookPayload) getMessage() string {
	if p.Error != nil {
		return p.Error.Message
	}
	return ""
}

func (p webhookPayload) getContext() string {
	if p.Error != nil {
		return p.Error.Context
	}
	return ""
}

func (p webhookPayload) getStatus() string {
	if p.Error != nil {
		return p.Error.Status
	}
	return ""
}

func (p webhookPayload) isUnhandled() bool {
	if p.Error != nil {
		return p.Error.Unhandled
	}
	return false
}

func (p webhookPayload) getAssignedCollaborator() *collaborator {
	if p.Error != nil {
		return p.Error.AssignedCollaborator
	}
	return nil
}

func (p webhookPayload) getAppVersion() string {
	if p.Error != nil && p.Error.App != nil {
		return p.Error.App.Version
	}
	return ""
}

func (p webhookPayload) getStacktrace() []stackFrame {
	if p.Error != nil {
		// Try exceptions first (newer format)
		if len(p.Error.Exceptions) > 0 && len(p.Error.Exceptions[0].Stacktrace) > 0 {
			return p.Error.Exceptions[0].Stacktrace
		}
		// Fall back to deprecated stackTrace field
		if len(p.Error.StackTrace) > 0 {
			return p.Error.StackTrace
		}
	}
	return nil
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

	mm := newMMClient(p.API, cfg.EnableDebugLog, p.kvNS(), p.botUserID)

	var payload webhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	allRules, err := loadChannelRules(mm)
	if err != nil {
		p.API.LogError("failed to load channel rules", "err", err.Error())
		http.Error(w, "cannot load channel mappings", http.StatusInternalServerError)
		return
	}

	projectID := payload.getProjectID()
	errorID := payload.getErrorID()

	channelRules := getRulesForProject(allRules, projectID)
	processed := 0

	for _, rule := range channelRules {
		if !matchesRule(rule, payload) {
			continue
		}

		if err := p.upsertErrorCard(mm, rule.ChannelID, payload, cfg); err != nil {
			p.API.LogError("failed to upsert webhook card", "channel", rule.ChannelID, "error_id", errorID, "project_id", projectID, "err", err.Error())
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
	exceptionClass := payload.getExceptionClass()
	message := payload.getMessage()

	if exceptionClass != "" && message != "" {
		return fmt.Sprintf(":rotating_light: **%s**: %s", exceptionClass, message)
	}
	if exceptionClass != "" {
		return ":rotating_light: **" + exceptionClass + "**"
	}
	if message != "" {
		return ":rotating_light: " + message
	}
	if payload.Trigger.Message != "" {
		return ":rotating_light: " + payload.Trigger.Message
	}
	return ":rotating_light: Bugsnag error"
}

func buildCardAttachment(payload webhookPayload, cfg Configuration, userMappings []UserMapping, mm *MMClient) *model.SlackAttachment {
	errorID := payload.getErrorID()
	projectID := payload.getProjectID()
	projectName := payload.getProjectName()
	errorURL := payload.getErrorURL()
	severity := payload.getSeverity()
	environment := payload.getEnvironment()
	context := payload.getContext()
	status := payload.getStatus()
	appVersion := payload.getAppVersion()

	// Build fields with short=true for compact display
	var attachmentFields []*model.SlackAttachmentField

	if severity != "" {
		attachmentFields = append(attachmentFields, &model.SlackAttachmentField{
			Title: "Severity",
			Value: severityEmoji(severity) + " " + severity,
			Short: true,
		})
	}

	if environment != "" {
		attachmentFields = append(attachmentFields, &model.SlackAttachmentField{
			Title: "Environment",
			Value: environment,
			Short: true,
		})
	}

	if status != "" {
		attachmentFields = append(attachmentFields, &model.SlackAttachmentField{
			Title: "Status",
			Value: status,
			Short: true,
		})
	}

	// Show assigned user instead of "Handled" field
	if assignee := payload.getAssignedCollaborator(); assignee != nil {
		assignedValue := assignee.Email // Default to email
		// Try to find Mattermost user
		if mmUserID := mapBugsnagToMattermost(userMappings, assignee.ID, assignee.Email); mmUserID != "" {
			if mmUser, appErr := mm.GetUser(mmUserID); appErr == nil {
				assignedValue = "@" + mmUser.Username
			}
		}
		attachmentFields = append(attachmentFields, &model.SlackAttachmentField{
			Title: "Assigned",
			Value: assignedValue,
			Short: true,
		})
	}

	if context != "" {
		attachmentFields = append(attachmentFields, &model.SlackAttachmentField{
			Title: "Context",
			Value: context,
			Short: true,
		})
	}

	if appVersion != "" {
		attachmentFields = append(attachmentFields, &model.SlackAttachmentField{
			Title: "App Version",
			Value: appVersion,
			Short: true,
		})
	}

	if projectName != "" {
		attachmentFields = append(attachmentFields, &model.SlackAttachmentField{
			Title: "Project",
			Value: projectName,
			Short: true,
		})
	}

	// Trigger info is shown in update comments, not in the card fields

	footer := "Bugsnag"
	if cfg.OrganizationID != "" {
		footer = fmt.Sprintf("Bugsnag • org %s", cfg.OrganizationID)
	}

	actionURL := fmt.Sprintf("/plugins/%s/actions", pluginID)
	actions := []*model.PostAction{
		{
			Id:    "assign_me",
			Name:  "Assign to me",
			Style: "primary",
			Type:  model.PostActionTypeButton,
			Integration: &model.PostActionIntegration{
				URL: actionURL,
				Context: map[string]any{
					"action":     "assign_me",
					"error_id":   errorID,
					"project_id": projectID,
					"error_url":  errorURL,
				},
			},
		},
		{
			Id:    "resolve",
			Name:  "Resolve",
			Style: "primary",
			Type:  model.PostActionTypeButton,
			Integration: &model.PostActionIntegration{
				URL: actionURL,
				Context: map[string]any{
					"action":     "resolve",
					"error_id":   errorID,
					"project_id": projectID,
				},
			},
		},
		{
			Id:    "ignore",
			Name:  "Ignore",
			Style: "default",
			Type:  model.PostActionTypeButton,
			Integration: &model.PostActionIntegration{
				URL: actionURL,
				Context: map[string]any{
					"action":     "ignore",
					"error_id":   errorID,
					"project_id": projectID,
				},
			},
		},
	}

	if errorURL != "" {
		actions = append(actions, &model.PostAction{
			Id:    "open",
			Name:  "Open in Bugsnag",
			Style: "link",
			Type:  model.PostActionTypeButton,
			Integration: &model.PostActionIntegration{
				URL: actionURL,
				Context: map[string]any{
					"action":    "open_in_browser",
					"error_url": errorURL,
				},
			},
		})
	}

	// Set color based on severity
	color := "#4949E4" // Bugsnag purple default
	switch severity {
	case "error":
		color = "#D9534F" // red
	case "warning":
		color = "#F0AD4E" // yellow/orange
	case "info":
		color = "#5BC0DE" // blue
	}

	return &model.SlackAttachment{
		Color:     color,
		Title:     payload.getExceptionClass(),
		TitleLink: errorURL,
		Text:      payload.getMessage(),
		Fields:    attachmentFields,
		Footer:    footer,
		Actions:   actions,
	}
}

func severityEmoji(severity string) string {
	switch severity {
	case "error":
		return "🔴"
	case "warning":
		return "🟡"
	case "info":
		return "🔵"
	default:
		return "⚪"
	}
}

// formatStacktrace formats the stacktrace for display in a comment
func formatStacktrace(frames []stackFrame, maxFrames int) string {
	if len(frames) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("**Stacktrace:**\n```\n")

	limit := len(frames)
	if maxFrames > 0 && limit > maxFrames {
		limit = maxFrames
	}

	for i := 0; i < limit; i++ {
		frame := frames[i]
		prefix := "  "
		if frame.InProject {
			prefix = "→ " // highlight in-project frames
		}

		file := frame.File
		if file == "" {
			file = "<unknown>"
		}

		method := frame.Method
		if method == "" {
			method = "<anonymous>"
		}

		line := ""
		if frame.LineNumber != nil {
			line = fmt.Sprintf(":%v", frame.LineNumber)
		}

		sb.WriteString(fmt.Sprintf("%s%s%s in %s\n", prefix, file, line, method))
	}

	if len(frames) > limit {
		sb.WriteString(fmt.Sprintf("  ... and %d more frames\n", len(frames)-limit))
	}

	sb.WriteString("```")
	return sb.String()
}

func (p *Plugin) upsertErrorCard(mm *MMClient, channelID string, payload webhookPayload, cfg Configuration) error {
	if strings.TrimSpace(channelID) == "" {
		return fmt.Errorf("channelID is required")
	}

	projectID := payload.getProjectID()
	errorID := payload.getErrorID()

	key := errorPostKVKey(projectID, errorID)
	var mapping ErrorPostMapping
	found, appErr := mm.LoadJSON(key, &mapping)
	if appErr != nil {
		return fmt.Errorf("load existing mapping: %w", appErr)
	}

	// Load user mappings for Assigned field
	userMappings, _ := loadUserMappings(mm)

	attachments := []*model.SlackAttachment{buildCardAttachment(payload, cfg, userMappings, mm)}
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

		// Add update comment with trigger info
		if payload.Trigger.Type != "" {
			replyMsg := fmt.Sprintf("🔄 **Update**: %s", payload.Trigger.Message)
			if _, appErr := mm.CreateReply(mapping.ChannelID, mapping.PostID, replyMsg); appErr != nil {
				mm.LogDebug("failed to append webhook reply", "err", appErr.Error())
			}
		}

		return nil
	}

	// Create new post
	post, appErr := mm.CreatePost(channelID, title, attachments)
	if appErr != nil {
		return fmt.Errorf("create post: %w", appErr)
	}

	// Store mapping for future updates
	mapping = ErrorPostMapping{
		ProjectID: projectID,
		ErrorID:   errorID,
		ChannelID: channelID,
		PostID:    post.Id,
	}

	if err := mm.StoreJSON(key, mapping); err != nil {
		mm.LogDebug("failed to store error→post mapping", "err", err.Error())
	}

	// Add stacktrace as first reply if available
	stacktrace := payload.getStacktrace()
	if len(stacktrace) > 0 {
		traceComment := formatStacktrace(stacktrace, 15) // limit to 15 frames
		if traceComment != "" {
			if _, appErr := mm.CreateReply(channelID, post.Id, traceComment); appErr != nil {
				mm.LogDebug("failed to add stacktrace reply", "err", appErr.Error())
			}
		}
	}

	// Register error for periodic sync
	kvStore := &pluginKVAdapter{api: p.API, namespace: p.kvNS()}
	s := store.New(kvStore)
	activeErr := store.ActiveError{
		ErrorID:      errorID,
		ProjectID:    projectID,
		PostID:       post.Id,
		ChannelID:    channelID,
		LastSyncedAt: time.Now().UTC(),
	}
	if err := s.UpsertActiveError(activeErr); err != nil {
		mm.LogDebug("failed to register active error for sync", "err", err.Error())
	}

	return nil
}
