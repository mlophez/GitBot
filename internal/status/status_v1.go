package status

import (
	"encoding/json"
	"net/http"
)

// Status handles GET /status.
// Returns a simple health check response to confirm the server is running.
func Status(w http.ResponseWriter, _ *http.Request) {
	response := map[string]string{"status": "OK"}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(jsonResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
