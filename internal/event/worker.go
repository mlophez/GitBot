package event

import (
	"context"
	mApp "gitbot/internal/app"
	"gitbot/internal/config"
	"log/slog"
	"time"

	"k8s.io/client-go/kubernetes"
)

func StartWorker(q Queue, cs *kubernetes.Clientset) func(context.Context) {
	quit := make(chan int)
	go func() {
		for {
			select {
			default:
				time.Sleep(1 * time.Second)

				// I/O. Get event from queue
				next := q.Dequeue()
				if next == nil {
					continue
				}

				slog.Info("Processing new event", "type", next.event.Type)

				// Logic. Run Processor
				action := ProcessEvent(next.event)
				if action == Nothing {
					continue
				}

				apps, err := mApp.GetAllPulRequestApps(cs, next.event)
				if err != nil {
					slog.Error("Error at try get all apps filtered", "error", err)
					continue
				}

				validation := ValidatePullRequestApps(next.event, apps, action)

				switch validation {
				case ValidationNotFound:
					slog.Info("None app to process", "pullrequest", next.event.PullRequestId)
				case ValidationAlreadyLockedByAnother:
					slog.Warn("One of the apps in the pull request is blocked by another pull request.", "pullrequest", next.event.PullRequestId)
					response(*next, "One of the apps in the pull request is blocked by another pull request.")
				case ValidationBranchNotMatch:
					slog.Warn("In one of the pull request apps, the target branch does not match the app", "pullrequest", next.event.PullRequestId)
					response(*next, "In one of the pull request apps, the target branch does not match the app")
				case ValidationAlreadyUnLocked:
					response(*next, "The pull request was already unblocked")
				case ValidationOk:
					switch action {
					case LockPullRequest:
						mApp.LockApps(cs, apps, next.event.PullRequestSourceBranch, next.event.PullRequestId)
						if err != nil {
							response(*next, "Error at locked pull request")
							continue
						}
						response(*next, "The pull request was locked successfully")
					case UnlockPullRequest:
						err := mApp.UnLockApps(cs, apps)
						if err != nil {
							response(*next, "Error at unlocked pull request")
							continue
						}
						response(*next, "The pull request was unlocked successfully")
					}
				default:
					slog.Warn("Unknown", "pullrequest", next.event.PullRequestId)
					response(*next, "UnknownError")
				}

			case <-quit:
				return
			}
		}
	}()

	return func(ctx context.Context) {
		defer close(quit)
		for {
			select {
			default:
				time.Sleep(1 * time.Second)
				if q.Size() <= 0 {
					return
				}
			case <-ctx.Done():
				return
			}
		}

	}
}

func response(item QueueItem, msg string) {
	clusterName := config.Get("CLUSTER_NAME")
	if clusterName != "" {
		msg = "[" + clusterName + "] " + msg
	}
	if err := item.provider.WriteComment(item.event.Repository, item.event.PullRequestId, item.event.CommentId, msg); err != nil {
		slog.Error("Error at respond to provider", "error", err)
	}
}
