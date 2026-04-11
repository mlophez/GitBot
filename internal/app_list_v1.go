// Package internal contains the use cases (imperative shell) for kubeops-agent.
// Each file implements one use case and is responsible for:
//   - Receiving injected dependencies via types.AppManager
//   - Orchestrating I/O calls and pure domain logic
//   - Defining the HTTP request/response contract for its endpoint
package internal

import (
	"encoding/json"
	"net/http"

	
	"gitbot/internal/adapters"
	"gitbot/internal/types"
)

// AppResponse is the HTTP response representation of an ArgoCD application.
// It exposes the fields relevant to API consumers and to remote agent
// communication (Paths is needed by the central instance for file-matching).
type AppResponse struct {
	Name          string   `json:"name"`
	Cluster       string   `json:"cluster,omitempty"`
	Repository    string   `json:"repository"`
	Branch        string   `json:"branch"`
	Paths         []string `json:"paths,omitempty"`
	Locked        bool     `json:"locked"`
	PullRequestId int      `json:"pull_request_id"`
	Environment   string   `json:"environment"`
}

// toAppResponse converts a domain Application into its HTTP response form.
func toAppResponse(app types.Application) AppResponse {
	return AppResponse{
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

// ListApps handles GET /api/v1/apps.
// Returns the list of all ArgoCD applications currently tracked in the cluster.
func ListApps(manager types.AppManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := adapters.Logger(r.Context())
		log.Info("ListApps request received")

		apps, err := manager.List()
		if err != nil {
			log.Error("ListApps failed to fetch apps", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("ListApps returning apps", "count", len(apps))

		resp := make([]AppResponse, len(apps))
		for i, a := range apps {
			resp[i] = toAppResponse(a)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
