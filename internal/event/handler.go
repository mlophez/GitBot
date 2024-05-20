package event

import (
	"log/slog"
	"net/http"
)

func Handle(q Queue, p Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		/* Get event from webhook */
		event, err := p.ParseEvent(r.Header, r.Body)
		if err != nil {
			slog.Error("Error at parse webhook")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		/* Put event in queue */
		item := QueueItem{event: event, provider: p}
		q.Enqueue(item)

		/* Response */
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("{}"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
