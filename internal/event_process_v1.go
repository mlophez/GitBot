package internal

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gitbot/internal/types"
)

// ProcessFn is the function signature for processing a single event.
// Returns the response to send back to the PR, and a bool indicating whether
// the event should be retried (e.g. for app-of-apps double-lock).
type ProcessFn func(types.Event) (*types.EventResponse, bool)

// EventProcessor dequeues events and processes them asynchronously.
// For each event it: enriches via the provider, runs the process function,
// and writes the result as a comment on the pull request.
type EventProcessor struct {
	queue       types.Queue
	process     ProcessFn
	clusterName string
	quit        chan struct{}
}

// NewEventProcessor creates a processor ready to be started.
func NewEventProcessor(queue types.Queue, process ProcessFn, clusterName string) *EventProcessor {
	return &EventProcessor{
		queue:       queue,
		process:     process,
		clusterName: clusterName,
		quit:        make(chan struct{}),
	}
}

// Start begins the processing loop. Intended to be run in a goroutine.
// Polls the queue every second and processes each item.
func (p *EventProcessor) Start() {
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
			var e types.Event
			for i := 1; i <= 3; i++ {
				var err error
				e, err = next.Provider.GetData(next.Event)
				if err == nil {
					break
				}
				slog.Warn("EventProcessor GetData failed, retrying", "attempt", i, "error", err)
				time.Sleep(1 * time.Second)
			}

			resp, retry := p.process(e)

			// Re-enqueue for app-of-apps double-lock scenario.
			if resp != nil && retry {
				slog.Info("EventProcessor re-enqueueing event for retry")
				time.Sleep(30 * time.Second)
				p.queue.Enqueue(*next)
			}

			if resp == nil {
				continue
			}

			msg := p.formatResponse(resp)
			if err := next.Provider.WriteComment(
				next.Event.Repository,
				next.Event.PullRequest.Id,
				next.Event.CommentId,
				msg,
			); err != nil {
				slog.Error("EventProcessor failed to write comment", "error", err)
			}
		}
	}
}

// Stop drains the queue gracefully and shuts down the processor.
// Blocks until the queue is empty or the context is cancelled.
func (p *EventProcessor) Stop(ctx context.Context) {
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

// formatResponse builds the comment body from an EventResponse.
func (p *EventProcessor) formatResponse(resp *types.EventResponse) string {
	var msg string

	if p.clusterName != "" {
		status := ternary(resp.Success, "SUCCESS", "FAILED")
		msg = fmt.Sprintf("**[%s]** => **%s**\n\n", strings.ToUpper(p.clusterName), status)
	} else {
		status := ternary(resp.Success, "Success", "Failed")
		msg = fmt.Sprintf("### Status: **%s**", status)
	}

	if resp.Message != "" {
		msg += resp.Message + ".  \n"
	} else {
		for _, app := range resp.Summary {
			msg += fmt.Sprintf("- **%s:** %s.  \n", strings.ToUpper(app.Name), app.Message)
		}
	}

	return msg
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
