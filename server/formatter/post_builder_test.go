package formatter

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestBuildErrorPost(t *testing.T) {
	errorData := ErrorData{
		ID:          "abcd1234efgh5678",
		ProjectID:   "proj-1",
		ProjectName: "backend-api",
		Summary:     "NullReferenceException in CheckoutController",
		Status:      "open",
		Environment: "production",
		Severity:    "error",
		Counts: Counts{
			Users:     12,
			Events1h:  3,
			Events24h: 42,
		},
		LastSeen: "2025-11-28T10:23:00Z",
		ErrorURL: "https://app.bugsnag.com/org/project/errors/abcd1234efgh5678",
	}

	mapping := ErrorPostMapping{
		ChannelID: "channel-123",
		ProjectID: "proj-1",
		ErrorID:   "abcd1234efgh5678",
	}

	mmMapping := MMUserMapping{MMUserID: "mm-user-42"}

	post := BuildErrorPost(errorData, mapping, mmMapping)

	if post.ChannelId != mapping.ChannelID {
		t.Fatalf("unexpected channel id: %s", post.ChannelId)
	}

	expectedMessage := ":rotating_light: **[BUG]** NullReferenceException in CheckoutController · Status: open"
	if post.Message != expectedMessage {
		t.Fatalf("unexpected message: %s", post.Message)
	}

	attachments, ok := post.Props["attachments"].([]*model.SlackAttachment)
	if !ok {
		t.Fatalf("expected attachments slice, got %T", post.Props["attachments"])
	}

	if len(attachments) != 1 {
		t.Fatalf("expected one attachment, got %d", len(attachments))
	}

	attachment := attachments[0]

	if attachment.Title != errorData.Summary {
		t.Errorf("unexpected title: %s", attachment.Title)
	}

	expectedTitleLink := errorData.ErrorURL
	if attachment.TitleLink != expectedTitleLink {
		t.Errorf("unexpected title link: %s", attachment.TitleLink)
	}

	expectedText := "Status: open | Env: production | Severity: error | Users: 12 | Events (1h/24h): 3 / 42 | Assigned to <@mm-user-42>\nLast seen: 2025-11-28T10:23:00Z"
	if attachment.Text != expectedText {
		t.Fatalf("unexpected text: %s", attachment.Text)
	}

	expectedFooter := "Bugsnag • backend-api"
	if attachment.Footer != expectedFooter {
		t.Errorf("unexpected footer: %s", attachment.Footer)
	}

	if len(attachment.Actions) != 4 {
		t.Fatalf("expected 4 actions, got %d", len(attachment.Actions))
	}

	firstAction := attachment.Actions[0]
	if firstAction.Id != "assign_me" || firstAction.Name != "Assign to me" {
		t.Errorf("unexpected first action: %+v", firstAction)
	}

	context := firstAction.Integration.Context
	if context["error_id"] != mapping.ErrorID || context["project_id"] != mapping.ProjectID {
		t.Errorf("unexpected context: %+v", context)
	}

	lastAction := attachment.Actions[3]
	if lastAction.Id != "open" || lastAction.Integration.Context["error_url"] != errorData.ErrorURL {
		t.Errorf("unexpected open action: %+v", lastAction)
	}
}
