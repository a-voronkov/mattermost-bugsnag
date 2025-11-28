package bugsnag

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustReadFixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}

	return data
}

func TestGetProjectsParsesResponse(t *testing.T) {
	fixture := mustReadFixture(t, "projects.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET request, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/organizations/org-123/projects") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(server.URL, "token-value", server.Client())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	projects, err := client.GetProjects(context.Background(), "org-123")
	if err != nil {
		t.Fatalf("GetProjects error: %v", err)
	}

	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	if projects[0].ID != "project-1" || projects[0].Name != "Backend" {
		t.Fatalf("unexpected first project: %+v", projects[0])
	}
	if projects[1].OrganizationID != "org-123" {
		t.Fatalf("unexpected organization id: %s", projects[1].OrganizationID)
	}
}

func TestUpdateErrorStatusParsesResponse(t *testing.T) {
	fixture := mustReadFixture(t, "update_error_status.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("expected PATCH request, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "\"status\":") {
			t.Fatalf("request body missing status: %s", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(server.URL, "token-value", server.Client())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	status, err := client.UpdateErrorStatus(context.Background(), "error-123", "resolved", "user-456")
	if err != nil {
		t.Fatalf("UpdateErrorStatus error: %v", err)
	}

	if status.ID != "error-123" {
		t.Fatalf("unexpected error id: %s", status.ID)
	}
	if status.Status != "resolved" {
		t.Fatalf("unexpected status: %s", status.Status)
	}
	if status.AssigneeID != "user-456" {
		t.Fatalf("unexpected assignee: %s", status.AssigneeID)
	}
}
