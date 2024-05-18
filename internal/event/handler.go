package event

import (
	"log/slog"
	"net/http"
)

type Handler struct {
	queue    Queue
	provider Provider
}

func NewHandler(q Queue, p Provider) *Handler {
	return &Handler{
		queue:    q,
		provider: p,
	}
}

func (h *Handler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		/* Get event from webhook */
		event, err := h.provider.ParseEvent(r.Header, r.Body)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			slog.Error("Error at parse webhook")
			return
		}

		/* Parse command here? or in process event? */
		// Better in process event

		/* Put event in queue */
		item := QueueItem{event: event, provider: h.provider}
		h.queue.Enqueue(item)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("{}"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
