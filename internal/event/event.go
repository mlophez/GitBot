// Package event is the vertical slice for pull request events.
// It contains the pure domain (Event, PullRequest, EventResponse, Provider, Queue,
// QueueItem, ProcessFn), the provider/queue adapters (BitbucketClient, MemoryQueue),
// and the use cases exposed via HTTP (event creation, processing, notification).
package event

import "fmt"

// EventType represents the category of a pull request event received from a Git provider.
type EventType int

const (
	EventTypeUnknown   EventType = -1
	EventTypeOpened    EventType = 0
	EventTypeUpdated   EventType = 1
	EventTypeDeclined  EventType = 2
	EventTypeMerged    EventType = 3
	EventTypeCommented EventType = 4
)

// Event represents a pull request event received from a Git provider webhook.
type Event struct {
	Type         EventType
	Repository   string
	ActorID      string
	ActorName    string
	Comment      string
	CommentId    int
	PullRequest  PullRequest
	BotGenerated bool // true when the event was triggered by the bot's own account
}

// Validate reports whether the event satisfies the minimal domain invariants: it
// must carry a repository and a recognized type. A recognized pull request event
// must also carry a valid pull request. EventTypeUnknown is accepted without a
// pull request because such webhooks are irrelevant and discarded downstream, not
// rejected at parse time. It is called right after an Event is parsed from a
// provider webhook (see Provider.ParseEvent implementations).
func (e Event) Validate() error {
	if e.Repository == "" {
		return fmt.Errorf("event repository is required")
	}
	// Exhaustive check (not a numeric range) so adding a new EventType forces an
	// explicit decision here instead of silently passing.
	switch e.Type {
	case EventTypeUnknown, EventTypeOpened, EventTypeUpdated,
		EventTypeDeclined, EventTypeMerged, EventTypeCommented:
	default:
		return fmt.Errorf("unrecognized event type %d", e.Type)
	}
	if e.Type != EventTypeUnknown {
		if err := e.PullRequest.Validate(); err != nil {
			return fmt.Errorf("event pull request: %w", err)
		}
	}
	return nil
}
