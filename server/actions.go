package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/a-voronkov/mattermost-bugsnag/server/bugsnag"
	"github.com/a-voronkov/mattermost-bugsnag/server/formatter"
)

type actionContext struct {
	Action    string `json:"action"`
	ErrorID   string `json:"error_id"`
	ProjectID string `json:"project_id"`
	ErrorURL  string `json:"error_url,omitempty"`
}

type interactiveAction struct {
	UserID  string        `json:"user_id"`
	Context actionContext `json:"context"`
}

func (p *Plugin) handleActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload interactiveAction
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid interactive action payload", http.StatusBadRequest)
		return
	}

	p.API.LogInfo("received interactive action", "user_id", payload.UserID, "action", payload.Context.Action)

	action := strings.TrimSpace(payload.Context.Action)
	if action == "" {
		http.Error(w, "missing action", http.StatusBadRequest)
		return
	}

	cfg := p.getConfiguration()
	mm := newMMClient(p.API, cfg.EnableDebugLog, p.kvNS(), p.botUserID)

	bugsnagClient, err := bugsnag.NewDefaultClient(cfg.BugsnagAPIToken)
	if err != nil {
		mm.LogDebug("bugsnag client init failed", "err", err.Error())
	}

	user, appErr := mm.GetUser(payload.UserID)
	if appErr != nil {
		http.Error(w, "invalid user", http.StatusBadRequest)
		return
	}

	mappings, err := loadUserMappings(mm)
	if err != nil {
		p.API.LogError("failed to load user mappings", "err", err.Error())
	}

	bugsnagUser, mapped := mapUserToBugsnag(mappings, user)

	mappingKey := errorPostKVKey(payload.Context.ProjectID, payload.Context.ErrorID)
	var postMapping ErrorPostMapping
	found, appErr := mm.LoadJSON(mappingKey, &postMapping)
	if appErr != nil {
		p.API.LogDebug("interactive action missing card mapping", "error_id", payload.Context.ErrorID, "project_id", payload.Context.ProjectID, "err", appErr.Error())
	}

	mention := fmt.Sprintf("@%s", user.Username)
	msgParts := []string{fmt.Sprintf("%s requested action \"%s\"", mention, action)}
	if mapped {
		bugsnagIdentity := bugsnagUser.BugsnagUserID
		if bugsnagIdentity == "" {
			bugsnagIdentity = bugsnagUser.BugsnagEmail
		}
		if bugsnagIdentity != "" {
			msgParts = append(msgParts, fmt.Sprintf("mapped to Bugsnag user %s", bugsnagIdentity))
		}
	}
	if payload.Context.ErrorURL != "" {
		msgParts = append(msgParts, fmt.Sprintf("source: %s", payload.Context.ErrorURL))
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	var newStatus string
	var assignedUsername string
	var actionSuccess bool

	switch action {
	case "assign_me":
		assignee := bugsnag.BestAssignee(bugsnag.UserMapping{
			BugsnagUserID: bugsnagUser.BugsnagUserID,
			BugsnagEmail:  bugsnagUser.BugsnagEmail,
		})
		if assignee == "" {
			msgParts = append(msgParts, "no Bugsnag mapping available for assignment")
			break
		}

		if bugsnagClient != nil {
			if err := bugsnagClient.AssignError(ctx, payload.Context.ProjectID, payload.Context.ErrorID, assignee); err != nil {
				msgParts = append(msgParts, fmt.Sprintf("Bugsnag assign failed: %v", err))
			} else {
				msgParts = append(msgParts, fmt.Sprintf("assigned to %s in Bugsnag", assignee))
				assignedUsername = user.Username
				actionSuccess = true
			}
		} else {
			msgParts = append(msgParts, "Bugsnag client unavailable, assignment skipped")
		}
	case "resolve":
		if bugsnagClient != nil {
			if err := bugsnagClient.UpdateProjectErrorStatus(ctx, payload.Context.ProjectID, payload.Context.ErrorID, "fixed"); err != nil {
				msgParts = append(msgParts, fmt.Sprintf("Bugsnag resolve failed: %v", err))
			} else {
				msgParts = append(msgParts, "status set to fixed in Bugsnag")
				newStatus = "fixed"
				actionSuccess = true
			}
		} else {
			msgParts = append(msgParts, "Bugsnag client unavailable, resolve skipped")
		}
	case "ignore":
		if bugsnagClient != nil {
			if err := bugsnagClient.UpdateProjectErrorStatus(ctx, payload.Context.ProjectID, payload.Context.ErrorID, "ignored"); err != nil {
				msgParts = append(msgParts, fmt.Sprintf("Bugsnag ignore failed: %v", err))
			} else {
				msgParts = append(msgParts, "status set to ignored in Bugsnag")
				newStatus = "ignored"
				actionSuccess = true
			}
		} else {
			msgParts = append(msgParts, "Bugsnag client unavailable, ignore skipped")
		}
	case "open_in_browser":
		// Return response that tells the client to open the URL
		if payload.Context.ErrorURL != "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":            "ok",
				"open_in_browser": payload.Context.ErrorURL,
			})
			return
		}
		http.Error(w, "no URL available", http.StatusBadRequest)
		return
	default:
		http.Error(w, "unsupported action", http.StatusBadRequest)
		return
	}

	// Update the card if action was successful
	if actionSuccess && found {
		if post, appErr := mm.GetPost(postMapping.PostID); appErr == nil {
			mapping := formatter.ErrorPostMapping{
				ChannelID: postMapping.ChannelID,
				ProjectID: payload.Context.ProjectID,
				ErrorID:   payload.Context.ErrorID,
			}
			updatedPost := formatter.UpdatePost(formatter.UpdatePostParams{
				Post:             post,
				NewStatus:        newStatus,
				Mapping:          mapping,
				ErrorURL:         payload.Context.ErrorURL,
				AssignedUsername: assignedUsername,
			})
			if _, appErr := mm.UpdatePost(updatedPost); appErr != nil {
				mm.LogDebug("failed to update card", "err", appErr.Error())
			}
		}
	}

	note := strings.Join(msgParts, " · ")

	if found {
		if _, appErr := mm.CreateReply(postMapping.ChannelID, postMapping.PostID, note); appErr != nil {
			mm.LogDebug("failed to record interactive action", "err", appErr.Error())
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"text": note,
	})
}
