package event

type EventType int

const (
	// EventType
	EventTypeUnknown   EventType = -1
	EventTypeOpened    EventType = 0
	EventTypeUpdated   EventType = 1
	EventTypeDeclined  EventType = 2
	EventTypeMerged    EventType = 3
	EventTypeCommented EventType = 4
)

type Event struct {
	Type          EventType
	Repository    string
	Author        string
	Comment       string
	CommentId     int
	PullRequest   PullRequest
	PullRequestID int
}

func NewEvent() {}
