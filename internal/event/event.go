// Package event is the vertical slice for pull request events.
// It contains the pure domain (Event, PullRequest, EventResponse, Provider, Queue,
// QueueItem, ProcessFn), the provider/queue adapters (BitbucketClient, MemoryQueue),
// and the use cases exposed via HTTP (event creation, processing, notification).
package event

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
	Type          EventType
	Repository    string
	ActorID       string
	ActorName     string
	Comment       string
	CommentId     int
	PullRequest   PullRequest
	PullRequestID int
	BotGenerated  bool // true when the event was triggered by the bot's own account
}
