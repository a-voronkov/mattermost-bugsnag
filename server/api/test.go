package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

type TestHandler struct {
	tokenProvider func() string
}

func NewHandler(tokenProvider func() string) http.Handler {
	return &TestHandler{tokenProvider: tokenProvider}
}

func (h *TestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/test" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := strings.TrimSpace(h.tokenProvider())
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing Bugsnag API token")
		return
	}

	projects := []string{"stub-project"}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"projects": projects,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
