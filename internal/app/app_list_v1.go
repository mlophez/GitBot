// Package app is the vertical slice for ArgoCD application management.
// Each *_v1.go file implements one HTTP use case and is responsible for:
//   - Receiving injected dependencies via AppManager
//   - Orchestrating I/O calls and pure domain logic
//   - Defining the HTTP request/response contract for its endpoint
package app

import (
	"encoding/json"
	"net/http"

	"gitbot/internal/logger"
)

// ListResponse is the HTTP response item for the list endpoint.
type ListResponse struct {
	Name          string   `json:"name"`
	Cluster       string   `json:"cluster,omitempty"`
	Repository    string   `json:"repository"`
	Branch        string   `json:"branch"`
	Paths         []string `json:"paths,omitempty"`
	Locked        bool     `json:"locked"`
	PullRequestId int      `json:"pull_request_id"`
	Environment   string   `json:"environment"`
	Status        string   `json:"status,omitempty"`         // Aggregated sync/health status (e.g. "Healthy", "OutOfSync")
	StatusMessage string   `json:"status_message,omitempty"` // Last sync/health error message, empty when healthy
}

// toListResponse converts a domain Application into its list response form.
func toListResponse(app Application) ListResponse {
	return ListResponse{
		Name:          app.Name,
		Cluster:       app.Cluster,
		Repository:    app.Repository,
		Branch:        app.Branch,
		Paths:         app.Paths,
		Locked:        app.Locked,
		PullRequestId: app.PullRequestId,
		Environment:   app.Environment,
		Status:        app.Status,
		StatusMessage: app.StatusMessage,
	}
}

// ListApps handles GET /api/v1/apps.
// Returns the list of all ArgoCD applications currently tracked in the cluster.
func ListApps(manager AppManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.Logger(r.Context())
		log.Info("ListApps request received")

		apps, err := manager.List()
		if err != nil {
			log.Error("ListApps failed to fetch apps", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("ListApps returning apps", "count", len(apps))

		resp := make([]ListResponse, len(apps))
		for i, a := range apps {
			resp[i] = toListResponse(a)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
