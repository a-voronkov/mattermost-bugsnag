package store

import (
	"bytes"
	"testing"
	"time"

	"github.com/a-voronkov/mattermost-bugsnag/server/kvkeys"
)

type memoryKVStore struct {
	data map[string][]byte
}

func newMemoryKVStore() *memoryKVStore {
	return &memoryKVStore{data: map[string][]byte{}}
}

func (kv *memoryKVStore) Get(key string) ([]byte, error) {
	value, ok := kv.data[key]
	if !ok {
		return nil, nil
	}

	// Return a copy to avoid accidental mutations in tests.
	return append([]byte(nil), value...), nil
}

func (kv *memoryKVStore) Set(key string, value []byte) error {
	kv.data[key] = append([]byte(nil), value...)
	return nil
}

func TestProjectChannelMappings(t *testing.T) {
	kv := newMemoryKVStore()
	s := New(kv)
	timestamp := time.Now().UTC()

	mapping := ProjectChannelMapping{
		ErrorID:      "err1",
		ProjectID:    "proj1",
		PostID:       "post1",
		ChannelID:    "chan1",
		LastSyncedAt: timestamp,
	}

	if err := s.SaveProjectChannelMapping(mapping); err != nil {
		t.Fatalf("save mapping: %v", err)
	}

	stored, err := s.GetProjectChannelMappings("proj1")
	if err != nil {
		t.Fatalf("get mapping: %v", err)
	}

	if len(stored) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(stored))
	}

	if stored[0] != mapping {
		t.Fatalf("unexpected mapping: %+v", stored[0])
	}

	updated := mapping
	updated.PostID = "post2"
	updated.LastSyncedAt = timestamp.Add(time.Minute)

	if err := s.SaveProjectChannelMapping(updated); err != nil {
		t.Fatalf("update mapping: %v", err)
	}

	stored, err = s.GetProjectChannelMappings("proj1")
	if err != nil {
		t.Fatalf("get mapping after update: %v", err)
	}

	if len(stored) != 1 {
		t.Fatalf("expected 1 mapping after update, got %d", len(stored))
	}

	if stored[0].PostID != updated.PostID || !stored[0].LastSyncedAt.Equal(updated.LastSyncedAt) {
		t.Fatalf("mapping was not updated: %+v", stored[0])
	}
}

func TestActiveErrors(t *testing.T) {
	kv := newMemoryKVStore()
	s := New(kv)

	first := ActiveError{
		ErrorID:      "err1",
		ProjectID:    "proj1",
		PostID:       "post1",
		ChannelID:    "chan1",
		LastSyncedAt: time.Now().UTC(),
	}

	second := ActiveError{
		ErrorID:      "err2",
		ProjectID:    "proj2",
		PostID:       "post2",
		ChannelID:    "chan2",
		LastSyncedAt: time.Now().UTC().Add(time.Minute),
	}

	if err := s.UpsertActiveError(first); err != nil {
		t.Fatalf("insert first active error: %v", err)
	}

	if err := s.UpsertActiveError(second); err != nil {
		t.Fatalf("insert second active error: %v", err)
	}

	activeErrors, err := s.ListActiveErrors()
	if err != nil {
		t.Fatalf("list active errors: %v", err)
	}

	if len(activeErrors) != 2 {
		t.Fatalf("expected 2 active errors, got %d", len(activeErrors))
	}

	firstUpdated := first
	firstUpdated.PostID = "post1b"

	if err := s.UpsertActiveError(firstUpdated); err != nil {
		t.Fatalf("update active error: %v", err)
	}

	activeErrors, err = s.ListActiveErrors()
	if err != nil {
		t.Fatalf("list active errors after update: %v", err)
	}

	if len(activeErrors) != 2 {
		t.Fatalf("expected 2 active errors after update, got %d", len(activeErrors))
	}

	found := false
	for _, ae := range activeErrors {
		if ae.ProjectID == firstUpdated.ProjectID && ae.ErrorID == firstUpdated.ErrorID {
			if ae.PostID != firstUpdated.PostID {
				t.Fatalf("active error was not updated: %+v", ae)
			}
			found = true
		}
	}

	if !found {
		t.Fatalf("updated active error was not found")
	}
}

func TestStoreSerializationIsolation(t *testing.T) {
	kv := newMemoryKVStore()
	s := New(kv)

	mapping := ProjectChannelMapping{ProjectID: "proj", ErrorID: "err"}
	if err := s.SaveProjectChannelMapping(mapping); err != nil {
		t.Fatalf("save mapping: %v", err)
	}

	raw := kv.data[kvkeys.ProjectChannelMappings]
	if len(raw) == 0 {
		t.Fatalf("expected data to be stored")
	}

	if bytes.Contains(raw, []byte("\n")) {
		t.Fatalf("unexpected formatting in stored data")
	}
}
