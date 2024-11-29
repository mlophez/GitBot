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

//type Parser func(headers http.Header, body io.ReadCloser) (Event, error)
//
//func ParseWebhook(parse Parser, eq Queue) http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		/* Get event from webhook */
//		e, err := parse(r.Header, r.Body)
//		if err != nil {
//			slog.Error("Error at parse webhook")
//			http.Error(w, "Bad Request", http.StatusBadRequest)
//			return
//		}
//
//		/* Put event in queue */
//		item := QueueItem{Event: e, Provider: provider}
//		eq.Enqueue(item)
//
//		/* Response */
//		w.Header().Set("Content-Type", "application/json")
//		w.WriteHeader(http.StatusOK)
//		_, err = w.Write([]byte("{}"))
//		if err != nil {
//			http.Error(w, err.Error(), http.StatusInternalServerError)
//		}
//	}
//}
