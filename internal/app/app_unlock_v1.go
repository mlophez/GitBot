package app

import (
	"encoding/json"
	"net/http"

	
	"gitbot/internal/logger"
)

// UnlockResponse is the HTTP response body for the unlock endpoint.
type UnlockResponse struct {
	Name          string   `json:"name"`
	Cluster       string   `json:"cluster,omitempty"`
	Repository    string   `json:"repository"`
	Branch        string   `json:"branch"`
	Paths         []string `json:"paths,omitempty"`
	Locked        bool     `json:"locked"`
	PullRequestId int      `json:"pull_request_id"`
	Environment   string   `json:"environment"`
}

// toUnlockResponse converts a domain Application into its unlock response form.
func toUnlockResponse(app Application) UnlockResponse {
	return UnlockResponse{
		Name:          app.Name,
		Cluster:       app.Cluster,
		Repository:    app.Repository,
		Branch:        app.Branch,
		Paths:         app.Paths,
		Locked:        app.Locked,
		PullRequestId: app.PullRequestId,
		Environment:   app.Environment,
	}
}

// UnlockApp handles POST /api/v1/apps/{id}/unlock.
// Restores the ArgoCD app to the branch it was pointing to before the lock,
// removes the lock annotations, and marks the app as unlocked.
// Returns 404 if the app does not exist, 409 if it is not currently locked.
func UnlockApp(manager AppManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		log := logger.Logger(r.Context())

		log.Info("UnlockApp request received", "app", id)

		apps, err := manager.List()
		if err != nil {
			log.Error("UnlockApp failed to fetch apps", "app", id, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		app, err := findByName(apps, id)
		if err != nil {
			log.Warn("UnlockApp app not found", "app", id)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		if !app.Locked {
			log.Warn("UnlockApp app is not locked", "app", id)
			http.Error(w, "app is not locked", http.StatusConflict)
			return
		}

		if err := manager.Unlock(app); err != nil {
			log.Error("UnlockApp failed to unlock app", "app", id, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("UnlockApp app unlocked successfully", "app", id, "restored_branch", app.LastBranch)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(toUnlockResponse(app.Unlock()))
	}
}
