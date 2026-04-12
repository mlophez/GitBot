package internal

import (
	"bytes"
	"io"
	"net/http"

	"gitbot/internal/adapters"
	"gitbot/internal/types"
)

// EventCreate handles webhook POST requests from any git provider.
// Parses the incoming webhook payload using the given provider, creates a structured event,
// and enqueues it for asynchronous processing by the event processor.
// Returns 401 if the webhook token is invalid, 400 if the payload cannot be parsed.
func EventCreate(queue types.Queue, provider types.Provider, webhookToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := adapters.Logger(r.Context())
		log.Info("EventCreate webhook received")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error("EventCreate failed to read body", "error", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if err := provider.ValidateWebhookToken(webhookToken, r.Header, body); err != nil {
			log.Error("EventCreate webhook authentication failed", "error", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		e, err := provider.ParseEvent(r.Header, io.NopCloser(bytes.NewReader(body)))
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
