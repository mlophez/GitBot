package event

import (
	mApp "gitbot/internal/app"
	"regexp"
	"strings"
)

type ProcessorResult int

const (
	Nothing ProcessorResult = iota
	LockPullRequest
	UnlockPullRequest
)

type iProcessor struct {
	event Event
}

func (p iProcessor) ProcessEvent() ProcessorResult {
	var command string

	// Get command from comment
	filter := regexp.MustCompile(`(?i)(/|#)(argo|flux|bot)\s(lock|deploy|test|unlock|undeploy|rollback)`).FindStringSubmatch(p.event.Comment)
	if len(filter) == 4 {
		command = filter[3]
	}

	switch p.event.Type {
	case EventTypeMerged:
		return UnlockPullRequest
	case EventTypeDeclined:
		return UnlockPullRequest
	case EventTypeUpdated:
		// Depends if already locked or not, Review
		return Nothing
	case EventTypeCommented:
		switch strings.ToUpper(command) {
		case "LOCK", "DEPLOY", "TEST":
			return LockPullRequest
		case "UNLOCK", "UNDEPLOY", "ROLLBACK":
			return UnlockPullRequest
		default:
			return Nothing
		}
	default:
		return Nothing
	}
}

func (p iProcessor) isAlreadyLocked(apps []mApp.Application) bool {
	for _, app := range apps {
		if !(app.Locked && app.PullRequestId == p.event.PullRequestId) {
			return false
		}
	}
	return true
}

func (p iProcessor) isMatchDestinationBranch(apps []mApp.Application) bool {
	for _, app := range apps {
		if p.event.PullRequestDestinationBranch != app.Branch {
			return false
		}
	}
	return true
}

func (p iProcessor) isLockedByAnotherPR(apps []mApp.Application) bool {
	for _, app := range apps {
		if app.Locked && app.PullRequestId != p.event.PullRequestId {
			return true
		}
	}
	return false
}

func (p iProcessor) isAlreadyUnlocked(apps []mApp.Application) bool {
	for _, app := range apps {
		if app.PullRequestId == p.event.PullRequestId {
			return false
		}
	}
	return true
}
