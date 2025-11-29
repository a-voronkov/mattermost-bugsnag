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
	"github.com/mattermost/mattermost/server/public/model"
)

func (p *Plugin) handleActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload model.PostActionIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		p.API.LogError("failed to decode action payload", "err", err.Error())
		http.Error(w, "invalid interactive action payload", http.StatusBadRequest)
		return
	}

	// Extract values from context
	action, _ := payload.Context["action"].(string)
	errorID, _ := payload.Context["error_id"].(string)
	projectID, _ := payload.Context["project_id"].(string)
	errorURL, _ := payload.Context["error_url"].(string)

	p.API.LogInfo("received interactive action", "user_id", payload.UserId, "action", action, "error_id", errorID, "project_id", projectID)

	action = strings.TrimSpace(action)
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

	user, appErr := mm.GetUser(payload.UserId)
	if appErr != nil {
		http.Error(w, "invalid user", http.StatusBadRequest)
		return
	}

	mappings, err := loadUserMappings(mm)
	if err != nil {
		p.API.LogError("failed to load user mappings", "err", err.Error())
	}

	bugsnagUser, mapped := mapUserToBugsnag(mappings, user)

	mappingKey := errorPostKVKey(projectID, errorID)
	var postMapping ErrorPostMapping
	found, appErr := mm.LoadJSON(mappingKey, &postMapping)
	if appErr != nil {
		p.API.LogDebug("interactive action missing card mapping", "error_id", errorID, "project_id", projectID, "err", appErr.Error())
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
	if errorURL != "" {
		msgParts = append(msgParts, fmt.Sprintf("source: %s", errorURL))
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
			if err := bugsnagClient.AssignError(ctx, projectID, errorID, assignee); err != nil {
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
			if err := bugsnagClient.UpdateProjectErrorStatus(ctx, projectID, errorID, "fixed"); err != nil {
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
			if err := bugsnagClient.UpdateProjectErrorStatus(ctx, projectID, errorID, "ignored"); err != nil {
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
		if errorURL != "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":            "ok",
				"open_in_browser": errorURL,
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
				ProjectID: projectID,
				ErrorID:   errorID,
			}
			updatedPost := formatter.UpdatePost(formatter.UpdatePostParams{
				Post:             post,
				NewStatus:        newStatus,
				Mapping:          mapping,
				ErrorURL:         errorURL,
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
