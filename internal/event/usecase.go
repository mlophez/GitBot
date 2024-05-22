package event

import (
	"gitbot/internal/types"
	"regexp"
	"strings"
)

//import (
//	. "gitbot/types"
//	"regexp"
//	"strings"
//)
//
//type ProcessorResult int
//type Validation int
//
//const (
//	Nothing ProcessorResult = iota
//	LockPullRequest
//	UnlockPullRequest
//	ValidationUnknown Validation = iota
//	ValidationOk
//	ValidationNotFound
//	ValidationAlreadyLocked
//	ValidationAlreadyLockedByAnother
//	ValidationAlreadyUnLocked
//	ValidationBranchNotMatch
//)

func GetAction(e types.Event) types.Action {
	var command string

	// Get command from comment
	filter := regexp.MustCompile(`(?i)(/|#)(argo|flux|bot)\s(lock|deploy|test|unlock|undeploy|rollback)`).FindStringSubmatch(e.Comment)
	if len(filter) == 4 {
		command = filter[3]
	}

	switch e.Type {
	case types.EventTypeMerged:
		return types.ActionUnlock
	case types.EventTypeDeclined:
		return types.ActionUnlock
	case types.EventTypeUpdated:
		return types.ActionNothing
	case types.EventTypeCommented:
		switch strings.ToUpper(command) {
		case "LOCK", "DEPLOY", "TEST":
			return types.ActionLock
		case "UNLOCK", "UNDEPLOY", "ROLLBACK":
			return types.ActionUnlock
		default:
			return types.ActionNothing
		}
	default:
		return types.ActionNothing
	}
}

//
//func ValidatePullRequestApps(e Event, apps []Application, action ProcessorResult) Validation {
//	//alreadyLocked := false // Pull request already was locked previusly
//	alreadyUnlocked := true
//	branchNotMatch := false
//	alreadyLockedByAnother := false
//
//	for _, app := range apps {
//		//if app.Locked && app.PullRequestId == e.PullRequestId {
//		//	alreadyLocked = true
//		//}
//		if app.Locked && app.PullRequestId != e.PullRequestId {
//			alreadyLockedByAnother = true
//		}
//		if !app.Locked && e.PullRequestDestinationBranch != app.Branch {
//			branchNotMatch = true
//		}
//		if e.PullRequestId == app.PullRequestId {
//			alreadyUnlocked = false
//		}
//	}
//
//	switch {
//
//	case len(apps) == 0: // None app in pull request matching
//		return ValidationNotFound
//
//	// Locking
//	case action == LockPullRequest && alreadyLockedByAnother: // Locked by another pr
//		return ValidationAlreadyLockedByAnother
//	case action == LockPullRequest && branchNotMatch: // Already locked previusly
//		return ValidationBranchNotMatch
//	//case action == LockPullRequest && alreadyLocked: // Already locked previusly
//	//	return ValidationOk
//	//return ValidationAlreadyLocked
//	case action == LockPullRequest:
//		return ValidationOk
//
//		// Unlocking
//	case action == UnlockPullRequest && alreadyUnlocked:
//		return ValidationAlreadyUnLocked
//	case action == UnlockPullRequest:
//		return ValidationOk
//	default:
//		return ValidationUnknown
//	}
//}
