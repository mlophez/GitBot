package types

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
	Author        string
	Comment       string
	CommentId     int
	PullRequest   PullRequest
	PullRequestID int
}

// PullRequest contains the state of a pull request at the time an event was received.
type PullRequest struct {
	Id                int
	SourceBranch      string
	DestinationBranch string
	Reviewers         int
	Approved          int
	RequestChanged    int
	CommitsBehind     int
	FilesChanged      []string
}

// QueueItem groups an event with the provider that generated it so the worker
// can use the same provider to enrich the event and write comments back.
type QueueItem struct {
	Event    Event
	Provider Provider
}

// SecurityRule defines which users are allowed to perform which actions
// on files belonging to a specific repository.
type SecurityRule struct {
	Repository   string
	FilePatterns []string
	Actions      []string
	Users        []string
}

// EventResponse holds the result of processing an event, including a success
// flag, an optional error message, and a per-app status summary.
type EventResponse struct {
	Success bool
	Message string
	Summary []EventAppStatus
}

// EventAppStatus describes the outcome of a lock or unlock operation on a single app.
type EventAppStatus struct {
	Name    string
	Message string
}
