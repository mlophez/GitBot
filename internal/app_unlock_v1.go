package internal

import (
	"encoding/json"
	"net/http"

	"gitbot/internal/types"
)

// UnlockApp handles POST /api/v1/apps/{id}/unlock.
// Restores the ArgoCD app to the branch it was pointing to before the lock,
// removes the lock annotations, and marks the app as unlocked.
// Returns 404 if the app does not exist, 409 if it is not currently locked.
func UnlockApp(manager types.AppManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		apps, err := manager.List()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		app, err := findByName(apps, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		if !app.Locked {
			http.Error(w, "app is not locked", http.StatusConflict)
			return
		}

		if err := manager.Unlock(app); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(toAppResponse(app.Unlock()))
	}
}
