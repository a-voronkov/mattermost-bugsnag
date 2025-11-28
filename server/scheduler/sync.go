package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/a-voronkov/mattermost-bugsnag/server/bugsnag"
	"github.com/a-voronkov/mattermost-bugsnag/server/kvkeys"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// ActiveError tracks a Bugsnag error that should be refreshed periodically.
type ActiveError struct {
	ProjectID string `json:"project_id"`
	ErrorID   string `json:"error_id"`
	ChannelID string `json:"channel_id"`
	PostID    string `json:"post_id"`
}

type errorSnapshot struct {
	Status     string
	Events     int
	Events24h  int
	LastSeen   time.Time
	LastSynced time.Time
}

// BugsnagClient defines the interface for Bugsnag API operations needed by the scheduler.
type BugsnagClient interface {
	GetError(ctx context.Context, projectID, errorID string) (*bugsnag.ErrorDetails, error)
}

// Runner periodically refreshes active errors and updates their posts/threads.
type Runner struct {
	api           plugin.API
	debug         bool
	client        BugsnagClient
	tokenProvider func() string
	stop          chan struct{}
	done          chan struct{}
	interval      time.Duration
}

// NewRunner builds a scheduler runner backed by the plugin API.
func NewRunner(api plugin.API, debug bool, tokenProvider func() string) *Runner {
	return &Runner{
		api:           api,
		debug:         debug,
		tokenProvider: tokenProvider,
	}
}

// SetClient allows injection of a custom Bugsnag client (useful for testing).
func (r *Runner) SetClient(client BugsnagClient) {
	r.client = client
}

// Start launches the ticker loop.
func (r *Runner) Start(interval time.Duration) {
	r.interval = interval
	r.stop = make(chan struct{})
	r.done = make(chan struct{})

	go r.run()
}

// Stop halts the ticker loop.
func (r *Runner) Stop() {
	if r.stop == nil {
		return
	}

	close(r.stop)
	<-r.done
	r.stop = nil
	r.done = nil
}

func (r *Runner) run() {
	defer close(r.done)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stop:
			return
		case <-ticker.C:
			r.tick()
		}
	}
}

func (r *Runner) tick() {
	ctx, cancel := context.WithTimeout(context.Background(), r.interval)
	defer cancel()

	// Ensure we have a Bugsnag client
	if err := r.ensureClient(); err != nil {
		r.logDebug("failed to create Bugsnag client", "err", err.Error())
		return
	}

	activeErrors, err := loadActiveErrors(r.api)
	if err != nil {
		r.logDebug("failed to load active errors", "err", err.Error())
		return
	}

	for _, active := range activeErrors {
		snapshot, fetchErr := r.fetchErrorSnapshot(ctx, active.ProjectID, active.ErrorID)
		if fetchErr != nil {
			r.logDebug("bugsnag sync fetch failed", "project_id", active.ProjectID, "error_id", active.ErrorID, "err", fetchErr.Error())
			continue
		}

		post, appErr := r.api.GetPost(active.PostID)
		if appErr != nil {
			r.logDebug("sync: failed to load post", "post_id", active.PostID, "err", appErr.Error())
			continue
		}

		post.Message = strings.TrimSpace(post.Message)
		post.Message = fmt.Sprintf("%s\n\nStatus: %s | Events (total/24h): %d/%d | Last seen: %s | Synced: %s", post.Message, snapshot.Status, snapshot.Events, snapshot.Events24h, snapshot.LastSeen.Format(time.RFC3339), snapshot.LastSynced.Format(time.RFC3339))

		if _, appErr = r.api.UpdatePost(post); appErr != nil {
			r.logDebug("sync: failed to update post", "post_id", post.Id, "err", appErr.Error())
			continue
		}

		threadMessage := fmt.Sprintf("[sync] Status: %s, events (total/24h): %d/%d, last seen: %s", snapshot.Status, snapshot.Events, snapshot.Events24h, snapshot.LastSeen.Format(time.RFC3339))
		if _, appErr = r.api.CreatePost(&model.Post{ChannelId: active.ChannelID, RootId: active.PostID, Message: threadMessage}); appErr != nil {
			r.logDebug("sync: failed to create thread note", "post_id", active.PostID, "err", appErr.Error())
			continue
		}
	}
}

func (r *Runner) ensureClient() error {
	if r.client != nil {
		return nil
	}

	token := r.tokenProvider()
	if token == "" {
		return fmt.Errorf("no Bugsnag API token configured")
	}

	client, err := bugsnag.NewDefaultClient(token)
	if err != nil {
		return err
	}

	r.client = client
	return nil
}

func (r *Runner) fetchErrorSnapshot(ctx context.Context, projectID, errorID string) (errorSnapshot, error) {
	details, err := r.client.GetError(ctx, projectID, errorID)
	if err != nil {
		return errorSnapshot{}, err
	}

	lastSeen, _ := time.Parse(time.RFC3339, details.LastSeen)

	return errorSnapshot{
		Status:     details.Status,
		Events:     details.Events,
		Events24h:  details.EventsLast24h,
		LastSeen:   lastSeen,
		LastSynced: time.Now().UTC(),
	}, nil
}

func loadActiveErrors(api plugin.API) ([]ActiveError, error) {
	data, appErr := api.KVGet(kvkeys.ActiveErrors)
	if appErr != nil {
		return nil, fmt.Errorf("load active errors: %w", appErr)
	}
	if data == nil {
		return []ActiveError{}, nil
	}

	var active []ActiveError
	if err := json.Unmarshal(data, &active); err != nil {
		return nil, fmt.Errorf("parse active errors: %w", err)
	}

	return active, nil
}

func (r *Runner) logDebug(msg string, keyValuePairs ...interface{}) {
	if r.debug {
		r.api.LogDebug(msg, keyValuePairs...)
	}
}
