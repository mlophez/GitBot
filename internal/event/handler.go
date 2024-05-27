package event

import (
	"log/slog"
	"net/http"
)

type Handler struct {
	provider Provider
	service  Service
}

func NewHandler(s Service, p Provider) *Handler {
	return &Handler{
		service:  s,
		provider: p,
	}
}

func (h Handler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		/* Get event from webhook */
		e, err := h.provider.ParseEvent(r.Header, r.Body)
		if err != nil {
			slog.Error("Error at parse webhook")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		/* Put event in queue */
		item := QueueItem{Event: e, Provider: h.provider}
		h.service.queue.Enqueue(item)

		/* Response */
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("{}"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
