package internal

import (
	"encoding/json"
	"net/http"

	"gitbot/internal/types"
)

func UnlockApp(
	getApp    func(name string) (types.Application, error),
	updateApp func(types.Application) error,
	cleanApp  func(name string) error,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		app, err := getApp(id)
		if err != nil {
			http.Error(w, "app not found", http.StatusNotFound)
			return
		}

		if !app.Locked {
			http.Error(w, "app is not locked", http.StatusConflict)
			return
		}

		// pure domain logic
		unlocked := app.Unlock()

		if err := updateApp(unlocked); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := cleanApp(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(toAppResponse(unlocked))
	}
}
