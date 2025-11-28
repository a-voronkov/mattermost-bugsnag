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

func buildActions(mapping ErrorPostMapping, errorURL string) []*model.PostAction {
	actionURL := fmt.Sprintf("/plugins/%s/actions", kvkeys.PluginID)

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
					"error_id":   mapping.ErrorID,
					"project_id": mapping.ProjectID,
					"error_url":  errorURL,
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
					"error_id":   mapping.ErrorID,
					"project_id": mapping.ProjectID,
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
					"error_id":   mapping.ErrorID,
					"project_id": mapping.ProjectID,
				},
			},
		},
	}

	if strings.TrimSpace(errorURL) != "" {
		actions = append(actions, &model.PostAction{
			Id:    "open",
			Name:  "🔗 Open in Bugsnag",
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

	return actions
}
