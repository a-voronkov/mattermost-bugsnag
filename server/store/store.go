package store

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	projectChannelMappingsKey = "bugsnag:project-channel-mappings"
	activeErrorsKey           = "bugsnag:active-errors"
)

// ProjectChannelMapping links a Bugsnag error to the Mattermost post and channel
// where updates should be sent.
type ProjectChannelMapping struct {
	ErrorID      string    `json:"error_id"`
	ProjectID    string    `json:"project_id"`
	PostID       string    `json:"post_id"`
	ChannelID    string    `json:"channel_id"`
	LastSyncedAt time.Time `json:"last_synced_at"`
}

// ActiveError represents the latest state of a Bugsnag error that should remain
// in sync with Mattermost.
type ActiveError struct {
	ErrorID      string    `json:"error_id"`
	ProjectID    string    `json:"project_id"`
	PostID       string    `json:"post_id"`
	ChannelID    string    `json:"channel_id"`
	LastSyncedAt time.Time `json:"last_synced_at"`
}

// KVStore defines the minimal operations needed to persist data. Implementations
// can wrap pluginapi/cluster stores, mocks, or in-memory helpers for testing.
type KVStore interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
}

// Store wraps a KV backend with helpers for persisting plugin data.
type Store struct {
	kv KVStore
}

// New creates a store backed by the provided KV implementation.
func New(kv KVStore) *Store {
	return &Store{kv: kv}
}

// SaveProjectChannelMapping creates or updates the record linking a Bugsnag
// error to the Mattermost post and channel where updates are posted.
func (s *Store) SaveProjectChannelMapping(mapping ProjectChannelMapping) error {
	mappings, err := s.loadProjectChannelMappings()
	if err != nil {
		return err
	}

	updated := false
	for i, existing := range mappings {
		if existing.ProjectID == mapping.ProjectID && existing.ErrorID == mapping.ErrorID {
			mappings[i] = mapping
			updated = true
			break
		}
	}

	if !updated {
		mappings = append(mappings, mapping)
	}

	return s.saveProjectChannelMappings(mappings)
}

// GetProjectChannelMappings returns the mappings stored for a specific project.
func (s *Store) GetProjectChannelMappings(projectID string) ([]ProjectChannelMapping, error) {
	mappings, err := s.loadProjectChannelMappings()
	if err != nil {
		return nil, err
	}

	var filtered []ProjectChannelMapping
	for _, mapping := range mappings {
		if mapping.ProjectID == projectID {
			filtered = append(filtered, mapping)
		}
	}

	return filtered, nil
}

// UpsertActiveError creates or updates the active error record that tracks the
// last sync state for a Bugsnag error.
func (s *Store) UpsertActiveError(active ActiveError) error {
	activeErrors, err := s.loadActiveErrors()
	if err != nil {
		return err
	}

	updated := false
	for i, existing := range activeErrors {
		if existing.ProjectID == active.ProjectID && existing.ErrorID == active.ErrorID {
			activeErrors[i] = active
			updated = true
			break
		}
	}

	if !updated {
		activeErrors = append(activeErrors, active)
	}

	return s.saveActiveErrors(activeErrors)
}

// ListActiveErrors returns all active error records.
func (s *Store) ListActiveErrors() ([]ActiveError, error) {
	return s.loadActiveErrors()
}

func (s *Store) loadProjectChannelMappings() ([]ProjectChannelMapping, error) {
	data, err := s.kv.Get(projectChannelMappingsKey)
	if err != nil {
		return nil, fmt.Errorf("get project channel mappings: %w", err)
	}

	if len(data) == 0 {
		return []ProjectChannelMapping{}, nil
	}

	var mappings []ProjectChannelMapping
	if err := json.Unmarshal(data, &mappings); err != nil {
		return nil, fmt.Errorf("decode project channel mappings: %w", err)
	}

	return mappings, nil
}

func (s *Store) saveProjectChannelMappings(mappings []ProjectChannelMapping) error {
	data, err := json.Marshal(mappings)
	if err != nil {
		return fmt.Errorf("encode project channel mappings: %w", err)
	}

	if err := s.kv.Set(projectChannelMappingsKey, data); err != nil {
		return fmt.Errorf("set project channel mappings: %w", err)
	}

	return nil
}

func (s *Store) loadActiveErrors() ([]ActiveError, error) {
	data, err := s.kv.Get(activeErrorsKey)
	if err != nil {
		return nil, fmt.Errorf("get active errors: %w", err)
	}

	if len(data) == 0 {
		return []ActiveError{}, nil
	}

	var activeErrors []ActiveError
	if err := json.Unmarshal(data, &activeErrors); err != nil {
		return nil, fmt.Errorf("decode active errors: %w", err)
	}

	return activeErrors, nil
}

func (s *Store) saveActiveErrors(activeErrors []ActiveError) error {
	data, err := json.Marshal(activeErrors)
	if err != nil {
		return fmt.Errorf("encode active errors: %w", err)
	}

	if err := s.kv.Set(activeErrorsKey, data); err != nil {
		return fmt.Errorf("set active errors: %w", err)
	}

	return nil
}
