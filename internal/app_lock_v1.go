package internal

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gitbot/internal/server"
	"gitbot/internal/types"
)

// LockRequest is the request body for the lock endpoint.
type LockRequest struct {
	Branch        string `json:"branch"`          // PR source branch to point the app at
	PullRequestId int    `json:"pull_request_id"` // PR that is acquiring the lock
}

// LockApp handles POST /api/v1/apps/{id}/lock.
// Points the ArgoCD app to the given branch and marks it as locked by the PR.
// Returns 404 if the app does not exist, 409 if it is already locked by another PR.
func LockApp(manager types.AppManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		log := server.Logger(r.Context())

		log.Info("LockApp request received", "app", id)

		var req LockRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("LockApp invalid request body", "app", id, "error", err)
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}

		apps, err := manager.List()
		if err != nil {
			log.Error("LockApp failed to fetch apps", "app", id, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		app, err := findByName(apps, id)
		if err != nil {
			log.Warn("LockApp app not found", "app", id)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		if app.Locked {
			log.Warn("LockApp app already locked", "app", id, "pull_request_id", app.PullRequestId)
			http.Error(w, "app is already locked", http.StatusConflict)
			return
		}

		if err := manager.Lock(app, req.Branch, req.PullRequestId); err != nil {
			log.Error("LockApp failed to lock app", "app", id, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("LockApp app locked successfully", "app", id, "branch", req.Branch, "pull_request_id", req.PullRequestId)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(toAppResponse(app.Lock(req.Branch, req.PullRequestId)))
	}
}

// findByName returns the first application whose Name matches, or an error if none is found.
func findByName(apps []types.Application, name string) (types.Application, error) {
	for _, a := range apps {
		if a.Name == name {
			return a, nil
		}
	}
	return types.Application{}, fmt.Errorf("app %q not found", name)
}
