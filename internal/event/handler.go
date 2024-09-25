package event

import (
	"log/slog"
	"net/http"
)

type Handler struct {
	provider Provider
	queue    Queue
}

func NewHandler(q Queue, p Provider) *Handler {
	return &Handler{
		queue:    q,
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
		h.queue.Enqueue(item)

		/* Response */
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("{}"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// type EventParser func(headers http.Header, body io.ReadCloser) (Event, error)
//
// func ParseWebhook(parser EventParser, provider Provider, queue Queue) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		/* Get event from webhook */
// 		e, err := parser(r.Header, r.Body)
// 		if err != nil {
// 			slog.Error("Error at parse webhook")
// 			http.Error(w, "Bad Request", http.StatusBadRequest)
// 			return
// 		}
//
// 		/* Put event in queue */
// 		item := QueueItem{Event: e, Provider: provider}
// 		queue.Enqueue(item)
//
// 		/* Response */
// 		w.Header().Set("Content-Type", "application/json")
// 		w.WriteHeader(http.StatusOK)
// 		_, err = w.Write([]byte("{}"))
// 		if err != nil {
// 			http.Error(w, err.Error(), http.StatusInternalServerError)
// 		}
// 	}
// }
