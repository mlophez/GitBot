package main

import (
	"context"
	"log/slog"
	"time"

	"gitbot/internal/event"
)

// eventProcessor dequeues events and processes them asynchronously.
// For each item it: enriches via the provider, runs the ProcessFn,
// and writes the result as a comment on the pull request.
// Intended to run as a long-lived background goroutine (see start).
type eventProcessor struct {
	queue       event.Queue
	process     event.ProcessFn
	clusterName string
	quit        chan struct{}
}

// newEventProcessor creates a processor ready to be started.
func newEventProcessor(queue event.Queue, process event.ProcessFn, clusterName string) *eventProcessor {
	return &eventProcessor{
		queue:       queue,
		process:     process,
		clusterName: clusterName,
		quit:        make(chan struct{}),
	}
}

// start begins the processing loop. Must be run in a goroutine.
// Polls the queue every second; for each item it enriches the event via the
// provider (up to 3 retries), calls the process function, and posts the result
// as a PR comment.
func (p *eventProcessor) start() {
	for {
		select {
		case <-p.quit:
			return
		default:
			time.Sleep(1 * time.Second)

			next := p.queue.Dequeue()
			if next == nil {
				continue
			}

			// Enrich event with files changed and commits behind — retry up to 3 times.
			var e event.Event
			for i := 1; i <= 3; i++ {
				var err error
				e, err = next.Provider.GetData(next.Event)
				if err == nil {
					break
				}
				slog.Warn("eventProcessor: GetData failed, retrying", "attempt", i, "error", err)
				time.Sleep(1 * time.Second)
			}

			resp, retry := p.process(e)

			// Re-enqueue for app-of-apps double-lock scenario.
			if resp != nil && retry {
				slog.Info("eventProcessor: re-enqueueing event for retry")
				time.Sleep(30 * time.Second)
				p.queue.Enqueue(*next)
			}

			if resp == nil {
				continue
			}

			if err := next.Provider.WriteEventResponse(
				next.Event.Repository,
				next.Event.PullRequest.Id,
				next.Event.CommentId,
				resp,
				p.clusterName,
			); err != nil {
				slog.Error("eventProcessor: failed to write comment", "error", err)
			}
		}
	}
}

// stop drains the queue gracefully and shuts down the processor.
// Blocks until the queue is empty or the context is cancelled.
func (p *eventProcessor) stop(ctx context.Context) {
	defer close(p.quit)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(1 * time.Second)
			if p.queue.Size() <= 0 {
				return
			}
		}
	}
}
