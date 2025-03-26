package event

import (
	"io"
	"net/http"
)

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

type QueueItem struct {
	Event    Event
	Provider Provider
}

type SecurityRule struct {
	Repository   string
	FilePatterns []string
	Actions      []string
	Users        []string
}

type Queue interface {
	Enqueue(item QueueItem)
	NextItem() *QueueItem
	Dequeue() *QueueItem
	Size() int
}

type Provider interface {
	//Name() string
	ParseEvent(headers http.Header, body io.ReadCloser) (Event, error)
	GetData(Event) (Event, error)
	WriteComment(repo string, prId int, parentId int, msg string) error
	//	RespondEvent(e GitEvent, msg string) error
}

type ProcessEventResult int

const (
	PROCESS_EVENT_RESULT_FAILED ProcessEventResult = iota
	PROCESS_EVENT_RESULT_SUCCESS
)

type AppValidationResult struct {
	Name          string
	Message       string
	PullRequestId int
}
