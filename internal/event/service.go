package event

import "log/slog"

type Service struct {
	queue Queue
	rules []SecurityRule
}

func NewService(q Queue, rules []SecurityRule) Service {
	return Service{
		queue: q,
		rules: rules,
	}
}

func (s Service) ProcessEvent(e Event, p Provider) {
	slog.Info("Processing new event", "type", e.Type)

	// Get if action is needed
	//action :=

	// If action get all apps

	// Check security

	// Checks

	// Lock or Unlock

	slog.Info("End Processing", "type", e.Type)
}

//slog.Info("Processing new event", "type", next.event.Type)

// Logic. Run Processor
//action := ProcessEvent(next.event)
//if action == Nothing {
//	continue
//}

// // Security
// // match = security.Match(user string, action Action, rules []SecurityRule)
//
// // Get from all cluster
// apps, err := mApp.GetAllPulRequestApps(cs, next.event)
// if err != nil {
// 	slog.Error("Error at try get all apps filtered", "error", err)
// 	continue
// }
//
// validation := ValidatePullRequestApps(next.event, apps, action)
//
// switch validation {
// case ValidationNotFound:
// 	slog.Info("None app to process", "pullrequest", next.event.PullRequestId)
// case ValidationAlreadyLockedByAnother:
// 	slog.Warn("One of the apps in the pull request is blocked by another pull request.", "pullrequest", next.event.PullRequestId)
// 	response(*next, "One of the apps in the pull request is blocked by another pull request.")
// case ValidationBranchNotMatch:
// 	slog.Warn("In one of the pull request apps, the target branch does not match the app", "pullrequest", next.event.PullRequestId)
// 	response(*next, "In one of the pull request apps, the target branch does not match the app")
// case ValidationAlreadyUnLocked:
// 	response(*next, "The pull request was already unblocked")
// case ValidationOk:
// 	switch action {
// 	case LockPullRequest:
// 		err := mApp.LockApps(cs, apps, next.event.PullRequestSourceBranch, next.event.PullRequestId)
// 		if err != nil {
// 			response(*next, "Error at locked pull request")
// 			continue
// 		}
// 		response(*next, "The pull request was locked successfully")
// 	case UnlockPullRequest:
// 		err := mApp.UnLockApps(cs, apps)
// 		if err != nil {
// 			response(*next, "Error at unlocked pull request")
// 			continue
// 		}
// 		response(*next, "The pull request was unlocked successfully")
// 	}
// default:
// 	slog.Warn("Unknown", "pullrequest", next.event.PullRequestId)
// 	response(*next, "UnknownError")
// }

//func response(item QueueItem, msg string) {
//	clusterName := config.Get("CLUSTER_NAME")
//	if clusterName != "" {
//		msg = "[" + clusterName + "] " + msg
//	}
//	if err := item.provider.WriteComment(item.event.Repository, item.event.PullRequestId, item.event.CommentId, msg); err != nil {
//		slog.Error("Error at respond to provider", "error", err)
//	}
//}
