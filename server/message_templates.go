package main

// cardAttachment approximates the Mattermost attachment payload we plan to render
// for Bugsnag errors. It is intentionally small; fields can grow alongside the
// server implementation.
type cardAttachment struct {
	Title       string            `json:"title"`
	TitleLink   string            `json:"title_link"`
	Text        string            `json:"text"`
	Footer      string            `json:"footer"`
	Actions     []cardAction      `json:"actions"`
	Fields      map[string]string `json:"fields,omitempty"`
	SeverityTag string            `json:"severity_tag,omitempty"`
}

type cardAction struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Style       string                 `json:"style"`
	Type        string                 `json:"type"`
	Integration map[string]interface{} `json:"integration"`
}

// messageTemplate is a helper structure to keep example payloads close to the
// server code until the webapp components are available.
type messageTemplate struct {
	ChannelID string         `json:"channel_id"`
	Message   string         `json:"message"`
	Props     map[string]any `json:"props"`
	Attach    cardAttachment `json:"attachment"`
	ErrorID   string         `json:"error_id"`
	ProjectID string         `json:"project_id"`
}
