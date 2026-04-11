package internal

import (
	"encoding/json"
	"net/http"
	"strings"

	
	"gitbot/internal/adapters"
	"gitbot/internal/types"
)

// notificationRequest is the expected body for POST /api/v1/notification.
type notificationRequest struct {
	AppName string `json:"app_name"`
	Message string `json:"message"`
}

// NotificationHandle handles POST /api/v1/notification.
// Looks up the ArgoCD app by name; if it is locked, posts the message to the
// pull request that holds the lock. Silently succeeds when the app is unlocked.
// Returns 400 if the body cannot be parsed, 404 if the app is not found.
func NotificationHandle(manager types.AppManager, provider types.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := adapters.Logger(r.Context())

		var body notificationRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			log.Error("NotificationHandle: bad request", "error", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		log.Info("NotificationHandle: received", "app", body.AppName, "message", body.Message)

		apps, err := manager.List()
		if err != nil {
			log.Error("NotificationHandle: failed to list apps", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var found *types.Application
		for i := range apps {
			if apps[i].Name == body.AppName {
				found = &apps[i]
				break
			}
		}

		if found == nil {
			log.Error("NotificationHandle: app not found", "app", body.AppName)
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		if !found.Locked {
			w.WriteHeader(http.StatusOK)
			return
		}

		msg := "**[" + strings.ToUpper(found.Environment) + "]** => " + body.Message
		log.Info("NotificationHandle: sending comment", "repo", found.Repository, "pr", found.PullRequestId)

		if err := provider.WriteComment(found.Repository, found.PullRequestId, 0, msg); err != nil {
			log.Error("NotificationHandle: failed to write comment", "app", body.AppName, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
