package internal

import (
	"encoding/json"
	"log/slog"
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

		slog.Info("UnlockApp request received", "app", id)

		apps, err := manager.List()
		if err != nil {
			slog.Error("UnlockApp failed to fetch apps", "app", id, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		app, err := findByName(apps, id)
		if err != nil {
			slog.Warn("UnlockApp app not found", "app", id)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		if !app.Locked {
			slog.Warn("UnlockApp app is not locked", "app", id)
			http.Error(w, "app is not locked", http.StatusConflict)
			return
		}

		if err := manager.Unlock(app); err != nil {
			slog.Error("UnlockApp failed to unlock app", "app", id, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		slog.Info("UnlockApp app unlocked successfully", "app", id, "restored_branch", app.LastBranch)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(toAppResponse(app.Unlock()))
	}
}
