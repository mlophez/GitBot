package event

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

type Worker struct {
	service     Service
	queue       Queue
	quit        chan int
	clusterName string
}

func NewWorker(q Queue, s Service, clusterName string) *Worker {
	return &Worker{
		quit:        make(chan int),
		service:     s,
		queue:       q,
		clusterName: clusterName,
	}
}

func (w *Worker) Start() {
	for {
		select {
		default:
			time.Sleep(1 * time.Second)
			// Get event from queue
			next := w.queue.Dequeue()
			if next == nil {
				continue
			}

			// Process
			resp := w.service.Process(next.Event)
			if resp == nil {
				continue
			}

			w.responseProvider(*next, resp)

		case <-w.quit:
			return
		}
	}
}

func (w *Worker) Stop(ctx context.Context) {
	defer close(w.quit)
	for {
		select {
		default:
			time.Sleep(1 * time.Second)
			if w.queue.Size() <= 0 {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (w Worker) responseProvider(item QueueItem, resp *Response) {
	var msg string

	if w.clusterName != "" {
		msg = "**[" + strings.ToUpper(w.clusterName) + "]** => **" + ifTernary(resp.Success, "SUCCESS", "FAILED") + "**\n\n"
	} else {
		msg = "### Status: **" + ifTernary(resp.Success, "Success", "Failed") + "**"
	}

	for _, app := range resp.Summary {
		msg = msg + "- **" + strings.ToUpper(app.Name) + ":** " + app.Message + ".  \n"
	}

	if err := item.Provider.WriteComment(item.Event.Repository, item.Event.PullRequest.Id, item.Event.CommentId, msg); err != nil {
		slog.Error("Error at respond to provider", "error", err)
	}
}

func ifTernary[T any](condition bool, trueVal T, falseVal T) T {
	if condition {
		return trueVal
	}
	return falseVal
}
