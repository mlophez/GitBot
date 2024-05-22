package worker

import (
	"context"
	"gitbot/internal/event"
	"time"
)

type Worker struct {
	quit  chan int
	queue event.Queue
}

func NewWorker(q event.Queue) *Worker {
	return &Worker{
		quit:  make(chan int),
		queue: q,
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
			next.Handler.ProcessEvent(next.Event)
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
