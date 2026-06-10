package event

import (
	"bytes"
	"io"
	"net/http"

	"gitbot/internal/logger"
)

// EventCreate handles webhook POST requests from any git provider.
// Parses the incoming webhook payload using the given provider — which returns a
// validated domain event — and enqueues it for asynchronous processing by the
// event processor.
// Returns 401 if the webhook token is invalid, 400 if the payload cannot be parsed
// or the resulting event fails validation.
func EventCreate(queue Queue, provider Provider, webhookToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.Logger(r.Context())
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

		// ParseEvent returns an already-validated event (validation happens at the
		// creation point); an invalid event surfaces here as an error → 400.
		e, err := provider.ParseEvent(r.Header, io.NopCloser(bytes.NewReader(body)))
		if err != nil {
			log.Error("EventCreate failed to parse event", "error", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		queue.Enqueue(QueueItem{Event: e, Provider: provider})
		log.Info("EventCreate event enqueued", "type", e.Type, "repository", e.Repository)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}
}
