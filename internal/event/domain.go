package event

import "gitbot/internal/types"

// Type aliases — the canonical definitions live in internal/types.
// These aliases keep existing code in this package compiling without changes.

type PullRequest = types.PullRequest
type QueueItem = types.QueueItem
type SecurityRule = types.SecurityRule
type Queue = types.Queue
type Provider = types.Provider

// ProcessEventResult and AppValidationResult are internal to this package.

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
