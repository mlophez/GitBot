// Package internal contains the use cases (imperative shell) for kubeops-agent.
// Each file implements one use case and is responsible for:
//   - Receiving injected dependencies via types.AppManager
//   - Orchestrating I/O calls and pure domain logic
//   - Defining the HTTP request/response contract for its endpoint
package internal

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"gitbot/internal/types"
)

// AppResponse is the HTTP response representation of an ArgoCD application.
// It exposes only the fields relevant to API consumers, hiding internal
// domain details such as Paths, ProviderId or ContainOther.
type AppResponse struct {
	Name          string `json:"name"`
	Repository    string `json:"repository"`
	Branch        string `json:"branch"`
	Locked        bool   `json:"locked"`
	PullRequestId int    `json:"pull_request_id"`
	Environment   string `json:"environment"`
}

// toAppResponse converts a domain Application into its HTTP response form.
func toAppResponse(app types.Application) AppResponse {
	return AppResponse{
		Name:          app.Name,
		Repository:    app.Repository,
		Branch:        app.Branch,
		Locked:        app.Locked,
		PullRequestId: app.PullRequestId,
		Environment:   app.Environment,
	}
}

// ListApps handles GET /api/v1/apps.
// Returns the list of all ArgoCD applications currently tracked in the cluster.
func ListApps(manager types.AppManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("ListApps request received")

		apps, err := manager.List()
		if err != nil {
			slog.Error("ListApps failed to fetch apps", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		slog.Info("ListApps returning apps", "count", len(apps))

		resp := make([]AppResponse, len(apps))
		for i, a := range apps {
			resp[i] = toAppResponse(a)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
