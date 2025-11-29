package formatter

import (
	"fmt"
	"strings"

	"github.com/a-voronkov/mattermost-bugsnag/server/kvkeys"
	"github.com/mattermost/mattermost/server/public/model"
)

// Counts mirrors the aggregate counts provided by Bugsnag for an error.
type Counts struct {
	Users     int
	Events1h  int
	Events24h int
}

// ErrorData captures the essential details needed to render a Bugsnag error card.
type ErrorData struct {
	ID          string
	ProjectID   string
	ProjectName string
	Summary     string
	Status      string
	Environment string
	Severity    string
	Counts      Counts
	LastSeen    string
	ErrorURL    string
}

// ErrorPostMapping identifies where the card belongs in Mattermost.
type ErrorPostMapping struct {
	ChannelID string
	ProjectID string
	ErrorID   string
}

// MMUserMapping links a Bugsnag error to a Mattermost user mention for display.
type MMUserMapping struct {
	MMUserID string
}

// BuildErrorPost creates a Mattermost post representing a Bugsnag error with
// an attachment that includes summary details and action buttons.
func BuildErrorPost(errorData ErrorData, mapping ErrorPostMapping, mmUserMapping MMUserMapping) model.Post {
	message := buildMessage(errorData)
	attachment := buildAttachment(errorData, mapping, mmUserMapping)

	return model.Post{
		ChannelId: mapping.ChannelID,
		Message:   message,
		Props: map[string]any{
			"attachments": []*model.SlackAttachment{attachment},
		},
	}
}

// UpdatePostParams contains parameters for updating a Bugsnag error post.
type UpdatePostParams struct {
	Post             *model.Post
	NewStatus        string
	Mapping          ErrorPostMapping
	ErrorURL         string
	AssignedUsername string // Mattermost username (without @)
}

// UpdatePost updates the status and/or assignment in an existing post's message and attachment.
// Returns the updated post ready to be saved.
func UpdatePost(params UpdatePostParams) *model.Post {
	post := params.Post

	// Update the message line with new status
	message := post.Message
	if idx := strings.Index(message, " · Status:"); idx > 0 {
		message = message[:idx]
	}
	if params.NewStatus != "" {
		message = fmt.Sprintf("%s · Status: %s", message, params.NewStatus)
	}
	post.Message = message

	// Update attachment if present
	att := extractFirstAttachment(post)
	if att != nil {
		// Update status in text field
		lines := strings.Split(att.Text, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "Status:") {
				parts := strings.Split(line, " | ")
				parts[0] = fmt.Sprintf("Status: %s", params.NewStatus)
				lines[i] = strings.Join(parts, " | ")
				break
			}
		}
		att.Text = strings.Join(lines, "\n")

		// Update status in Fields
		for i, field := range att.Fields {
			if field.Title == "Status" {
				att.Fields[i].Value = params.NewStatus
				break
			}
		}

		// Rebuild actions with current status for proper button states
		att.Actions = BuildActions(BuildActionsParams{
			Mapping:        params.Mapping,
			ErrorURL:       params.ErrorURL,
			CurrentStatus:  params.NewStatus,
			AssignedUserID: params.AssignedUsername,
		})
		post.Props["attachments"] = []*model.SlackAttachment{att}
	}

	return post
}

// extractFirstAttachment extracts the first SlackAttachment from post Props.
// Handles both []*model.SlackAttachment (in-memory) and []interface{} (from DB).
func extractFirstAttachment(post *model.Post) *model.SlackAttachment {
	if post.Props == nil {
		return nil
	}

	attachments := post.Props["attachments"]
	if attachments == nil {
		return nil
	}

	// Try direct type assertion first (in-memory posts)
	if slackAttachments, ok := attachments.([]*model.SlackAttachment); ok && len(slackAttachments) > 0 {
		return slackAttachments[0]
	}

	// Handle []interface{} from JSON deserialization (posts loaded from DB)
	if arr, ok := attachments.([]interface{}); ok && len(arr) > 0 {
		if m, ok := arr[0].(map[string]interface{}); ok {
			return mapToSlackAttachment(m)
		}
	}

	// Handle []map[string]interface{}
	if arr, ok := attachments.([]map[string]interface{}); ok && len(arr) > 0 {
		return mapToSlackAttachment(arr[0])
	}

	return nil
}

// mapToSlackAttachment converts a map to SlackAttachment.
func mapToSlackAttachment(m map[string]interface{}) *model.SlackAttachment {
	att := &model.SlackAttachment{}

	if v, ok := m["title"].(string); ok {
		att.Title = v
	}
	if v, ok := m["title_link"].(string); ok {
		att.TitleLink = v
	}
	if v, ok := m["text"].(string); ok {
		att.Text = v
	}
	if v, ok := m["footer"].(string); ok {
		att.Footer = v
	}
	if v, ok := m["color"].(string); ok {
		att.Color = v
	}

	// Extract fields
	if fields, ok := m["fields"].([]interface{}); ok {
		for _, f := range fields {
			if fm, ok := f.(map[string]interface{}); ok {
				field := &model.SlackAttachmentField{}
				if v, ok := fm["title"].(string); ok {
					field.Title = v
				}
				if v, ok := fm["value"]; ok {
					field.Value = v
				}
				if v, ok := fm["short"].(bool); ok {
					field.Short = model.SlackCompatibleBool(v)
				}
				att.Fields = append(att.Fields, field)
			}
		}
	}

	return att
}

// UpdatePostStatus updates the status in an existing post's message and attachment.
// Returns the updated post ready to be saved.
// Deprecated: Use UpdatePost instead for more control over the update.
func UpdatePostStatus(post *model.Post, newStatus string, mapping ErrorPostMapping, errorURL string) *model.Post {
	return UpdatePost(UpdatePostParams{
		Post:      post,
		NewStatus: newStatus,
		Mapping:   mapping,
		ErrorURL:  errorURL,
	})
}

func buildMessage(errorData ErrorData) string {
	base := fmt.Sprintf(":rotating_light: **[BUG]** %s", strings.TrimSpace(errorData.Summary))
	status := strings.TrimSpace(errorData.Status)
	if status == "" {
		return base
	}

	return fmt.Sprintf("%s · Status: %s", base, status)
}

func buildAttachment(errorData ErrorData, mapping ErrorPostMapping, mmUserMapping MMUserMapping) *model.SlackAttachment {
	fields := []string{}

	if strings.TrimSpace(errorData.Status) != "" {
		fields = append(fields, fmt.Sprintf("Status: %s", strings.TrimSpace(errorData.Status)))
	}
	if strings.TrimSpace(errorData.Environment) != "" {
		fields = append(fields, fmt.Sprintf("Env: %s", strings.TrimSpace(errorData.Environment)))
	}
	if strings.TrimSpace(errorData.Severity) != "" {
		fields = append(fields, fmt.Sprintf("Severity: %s", strings.TrimSpace(errorData.Severity)))
	}
	if errorData.Counts.Users > 0 {
		fields = append(fields, fmt.Sprintf("Users: %d", errorData.Counts.Users))
	}
	if errorData.Counts.Events1h > 0 || errorData.Counts.Events24h > 0 {
		fields = append(fields, fmt.Sprintf("Events (1h/24h): %d / %d", errorData.Counts.Events1h, errorData.Counts.Events24h))
	}
	if strings.TrimSpace(mmUserMapping.MMUserID) != "" {
		fields = append(fields, fmt.Sprintf("Assigned to <@%s>", strings.TrimSpace(mmUserMapping.MMUserID)))
	}

	text := strings.Join(fields, " | ")
	if strings.TrimSpace(errorData.LastSeen) != "" {
		if text != "" {
			text += "\n"
		}
		text += fmt.Sprintf("Last seen: %s", strings.TrimSpace(errorData.LastSeen))
	}

	footer := "Bugsnag"
	if strings.TrimSpace(errorData.ProjectName) != "" {
		footer = fmt.Sprintf("Bugsnag • %s", strings.TrimSpace(errorData.ProjectName))
	}

	return &model.SlackAttachment{
		Title:     errorData.Summary,
		TitleLink: errorData.ErrorURL,
		Text:      text,
		Footer:    footer,
		Actions:   buildActions(mapping, errorData.ErrorURL),
	}
}

// BuildActionsParams contains the parameters for building action buttons.
type BuildActionsParams struct {
	Mapping        ErrorPostMapping
	ErrorURL       string
	CurrentStatus  string
	AssignedUserID string
}

func buildActions(mapping ErrorPostMapping, errorURL string) []*model.PostAction {
	return BuildActions(BuildActionsParams{
		Mapping:  mapping,
		ErrorURL: errorURL,
	})
}

// BuildActions creates action buttons for a Bugsnag error post with optional
// state-based modifications (disabled buttons, assigned user display).
func BuildActions(params BuildActionsParams) []*model.PostAction {
	actionURL := fmt.Sprintf("/plugins/%s/actions", kvkeys.PluginID)

	var actions []*model.PostAction

	// Assign button - show "Assigned to @user" if already assigned
	if params.AssignedUserID != "" {
		actions = append(actions, &model.PostAction{
			Id:       "assigned",
			Name:     fmt.Sprintf("Assigned to @%s", params.AssignedUserID),
			Style:    "default",
			Type:     model.PostActionTypeButton,
			Disabled: true,
		})
	} else {
		actions = append(actions, &model.PostAction{
			Id:    "assign_me",
			Name:  "Assign to me",
			Style: "primary",
			Type:  model.PostActionTypeButton,
			Integration: &model.PostActionIntegration{
				URL: actionURL,
				Context: map[string]any{
					"action":     "assign_me",
					"error_id":   params.Mapping.ErrorID,
					"project_id": params.Mapping.ProjectID,
					"error_url":  params.ErrorURL,
				},
			},
		})
	}

	// Resolve button - disable if status is already "fixed"
	resolveDisabled := params.CurrentStatus == "fixed"
	actions = append(actions, &model.PostAction{
		Id:       "resolve",
		Name:     "✓ Resolve",
		Style:    "primary",
		Type:     model.PostActionTypeButton,
		Disabled: resolveDisabled,
		Integration: &model.PostActionIntegration{
			URL: actionURL,
			Context: map[string]any{
				"action":     "resolve",
				"error_id":   params.Mapping.ErrorID,
				"project_id": params.Mapping.ProjectID,
			},
		},
	})

	// Ignore button - disable if status is already "ignored"
	ignoreDisabled := params.CurrentStatus == "ignored"
	actions = append(actions, &model.PostAction{
		Id:       "ignore",
		Name:     "✕ Ignore",
		Style:    "default",
		Type:     model.PostActionTypeButton,
		Disabled: ignoreDisabled,
		Integration: &model.PostActionIntegration{
			URL: actionURL,
			Context: map[string]any{
				"action":     "ignore",
				"error_id":   params.Mapping.ErrorID,
				"project_id": params.Mapping.ProjectID,
			},
		},
	})

	if strings.TrimSpace(params.ErrorURL) != "" {
		actions = append(actions, &model.PostAction{
			Id:    "open",
			Name:  "Open in Bugsnag",
			Style: "link",
			Type:  model.PostActionTypeButton,
			Integration: &model.PostActionIntegration{
				URL: actionURL,
				Context: map[string]any{
					"action":    "open_in_browser",
					"error_url": params.ErrorURL,
				},
			},
		})
	}

	return actions
}
