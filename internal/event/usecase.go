package event

import (
	"regexp"
	"strings"
)

func GetAction(e Event) Action {
	var command string

	// Get command from comment
	filter := regexp.MustCompile(`(?i)(/|#)(argo|flux|bot)\s(lock|deploy|test|unlock|undeploy|rollback)`).FindStringSubmatch(e.Comment)
	if len(filter) == 4 {
		command = filter[3]
	}

	switch e.Type {
	case EventTypeMerged:
		return ActionUnlock
	case EventTypeDeclined:
		return ActionUnlock
	case EventTypeUpdated:
		return ActionNothing
	case EventTypeCommented:
		switch strings.ToUpper(command) {
		case "LOCK", "DEPLOY", "TEST":
			return ActionLock
		case "UNLOCK", "UNDEPLOY", "ROLLBACK":
			return ActionUnlock
		default:
			return ActionNothing
		}
	default:
		return ActionNothing
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
