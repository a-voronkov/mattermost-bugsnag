package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testResponse struct {
	Status   string   `json:"status"`
	Projects []string `json:"projects"`
	Error    string   `json:"error"`
}

func TestTestHandlerHappyPath(t *testing.T) {
	handler := NewHandler(func() string { return "token" })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp testResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Fatalf("expected status ok, got %s", resp.Status)
	}

	if len(resp.Projects) == 0 {
		t.Fatalf("expected projects to be returned")
	}
}

func TestTestHandlerMissingToken(t *testing.T) {
	handler := NewHandler(func() string { return "" })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	var resp testResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error == "" {
		t.Fatalf("expected error message for missing token")
	}
}
