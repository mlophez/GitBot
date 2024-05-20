package types

const (
	// EventType
	EventTypeUnknown   EventType = -1
	EventTypeOpened    EventType = 0
	EventTypeUpdated   EventType = 1
	EventTypeDeclined  EventType = 2
	EventTypeMerged    EventType = 3
	EventTypeCommented EventType = 4
	// Command
	//NoneCommand       GitEventCommand = 0
	//LockCommand       GitEventCommand = 1
	//UnLockCommand     GitEventCommand = 2
)

type EventType int

type Event struct {
	Type                         EventType
	Repository                   string
	Author                       string
	Comment                      string
	CommentId                    int
	PullRequestId                int
	PullRequestSourceBranch      string
	PullRequestDestinationBranch string
	PullRequestRepoSlug          string
	PullRequestFilesChanged      []string
	//Command                 GitEventCommand
}
