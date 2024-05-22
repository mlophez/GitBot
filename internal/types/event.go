package types

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

type PullRequest struct {
	Id                int
	SourceBranch      string
	DestinationBranch string
	Approved          bool
	FilesChanged      []string
}

type Event struct {
	Type        EventType
	Repository  string
	Author      string
	Comment     string
	CommentId   int
	PullRequest PullRequest
}
