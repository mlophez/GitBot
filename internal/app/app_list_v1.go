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

		resp := make([]AppResponse, len(apps))
		for i, a := range apps {
			resp[i] = toAppResponse(a)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
