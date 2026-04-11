package internal

import (
	"net/http"

	"gitbot/internal/server"
	"gitbot/internal/types"
)

// EventCreate handles webhook POST requests from any git provider.
// Parses the incoming webhook payload using the given provider, creates a structured event,
// and enqueues it for asynchronous processing by the event processor.
// Returns 400 if the payload cannot be parsed.
func EventCreate(queue types.Queue, provider types.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := server.Logger(r.Context())
		log.Info("EventCreate webhook received")

		e, err := provider.ParseEvent(r.Header, r.Body)
		if err != nil {
			log.Error("EventCreate failed to parse event", "error", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		queue.Enqueue(types.QueueItem{Event: e, Provider: provider})
		log.Info("EventCreate event enqueued", "type", e.Type, "repository", e.Repository)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}
}
