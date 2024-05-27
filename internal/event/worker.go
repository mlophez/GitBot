package event

import (
	"context"
	"time"
)

type Worker struct {
	service Service
	quit    chan int
}

func NewWorker(s Service) *Worker {
	return &Worker{
		quit:    make(chan int),
		service: s,
	}
}

func (w *Worker) Start() {
	for {
		select {
		default:
			time.Sleep(1 * time.Second)
			// Get event from queue
			next := w.service.queue.Dequeue()
			if next == nil {
				continue
			}
			w.service.ProcessEvent(next.Event, next.Provider)
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
			if w.service.queue.Size() <= 0 {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
