package internal

import (
	"encoding/json"
	"net/http"

	"gitbot/internal/types"
)

type LockRequest struct {
	Branch        string `json:"branch"`
	PullRequestId int    `json:"pull_request_id"`
}

func LockApp(
	getApp    func(name string) (types.Application, error),
	updateApp func(types.Application) error,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var req LockRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}

		app, err := getApp(id)
		if err != nil {
			http.Error(w, "app not found", http.StatusNotFound)
			return
		}

		if app.Locked {
			http.Error(w, "app is already locked", http.StatusConflict)
			return
		}

		// pure domain logic
		locked := app.Lock(req.Branch, req.PullRequestId)

		if err := updateApp(locked); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(toAppResponse(locked))
	}
}
