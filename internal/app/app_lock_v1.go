package app

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gitbot/internal/logger"
)

// LockRequest is the request body for the lock endpoint.
type LockRequest struct {
	Branch        string `json:"branch"`          // PR source branch to point the app at
	PullRequestId int    `json:"pull_request_id"` // PR that is acquiring the lock
}

// LockResponse is the HTTP response body for the lock endpoint.
type LockResponse struct {
	Name          string   `json:"name"`
	Cluster       string   `json:"cluster,omitempty"`
	Repository    string   `json:"repository"`
	Branch        string   `json:"branch"`
	Paths         []string `json:"paths,omitempty"`
	Locked        bool     `json:"locked"`
	PullRequestId int      `json:"pull_request_id"`
	Environment   string   `json:"environment"`
}

// toLockResponse converts a domain Application into its lock response form.
func toLockResponse(app Application) LockResponse {
	return LockResponse{
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

// LockApp handles POST /api/v1/apps/{id}/lock.
// Points the ArgoCD app to the given branch and marks it as locked by the PR.
// Returns 404 if the app does not exist.
// Returns 409 if the app is already locked, unless the query parameter ?force=true
// is provided — in that case the lock is overwritten directly (branch-to-branch transition).
func LockApp(manager AppManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		force := r.URL.Query().Get("force") == "true"
		log := logger.Logger(r.Context())

		log.Info("LockApp request received", "app", id, "force", force)

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

		if app.Locked && !force {
			log.Warn("LockApp app already locked", "app", id, "pull_request_id", app.PullRequestId)
			http.Error(w, "app is already locked", http.StatusConflict)
			return
		}

		if err := manager.Lock(app, req.Branch, req.PullRequestId, force); err != nil {
			log.Error("LockApp failed to lock app", "app", id, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("LockApp app locked successfully", "app", id, "branch", req.Branch, "pull_request_id", req.PullRequestId, "force", force)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(toLockResponse(app.Lock(req.Branch, req.PullRequestId)))
	}
}

// findByName returns the first application whose Name matches, or an error if none is found.
func findByName(apps []Application, name string) (Application, error) {
	for _, a := range apps {
		if a.Name == name {
			return a, nil
		}
	}
	return Application{}, fmt.Errorf("app %q not found", name)
}
