package event

import (
	"context"
	mApp "gitbot/internal/app"
	"log/slog"
	"time"
)

type Worker struct {
	quit  chan int
	queue Queue
}

func NewWorker(q Queue) *Worker {
	return &Worker{
		quit:  make(chan int),
		queue: q,
	}
}

func (w *Worker) Start() {
	for {
		select {
		default:
			time.Sleep(1 * time.Second)

			// I/O. Get event from queue
			next := w.queue.Dequeue()
			if next == nil {
				continue
			}

			slog.Info("Processing new event", "type", next.event.Type)

			// Logic. Run Processor
			p := iProcessor{event: next.event}
			result := p.ProcessEvent()
			if result == Nothing {
				continue
			}

			apps, err := mApp.FindAppsByRepoAndFiles(next.event.Repository, next.event.PullRequestFilesChanged)
			if err != nil {
				slog.Error(err.Error())
			}

			isPrAlreadyLocked := p.isAlreadyLocked(apps)                   // All apps locked by me
			isPrMatchDestinationBranch := p.isMatchDestinationBranch(apps) // PrTargetBranch == AppCurrentBranch
			isPrAlreadyUnlocked := p.isAlreadyUnlocked(apps)               // All apps of pr is unlocked by me
			isAnyAppLockedByAnotherPr := p.isLockedByAnotherPR(apps)       // Some app of pr is locked by another pr

			switch result {
			case LockPullRequest:
				if isPrAlreadyLocked {
					slog.Warn("The pull request has already blocked all the apps", "pullrequest", next.event.PullRequestId)
					w.response(next.event, next.provider, "The pull request has already blocked all the apps")
				} else if isAnyAppLockedByAnotherPr {
					slog.Warn("One of the apps in the pull request is blocked by another pull request.", "pullrequest", next.event.PullRequestId)
					w.response(next.event, next.provider, "One of the apps in the pull request is blocked by another pull request.")
				} else if !isPrMatchDestinationBranch {
					slog.Warn("In one of the pull request apps, the target branch does not match the app", "pullrequest", next.event.PullRequestId)
					w.response(next.event, next.provider, "In one of the pull request apps, the target branch does not match the app")
				} else {
					slog.Info("Locking pull request", "pullrequest", next.event.PullRequestId)
					if err := w.lock(next.event, apps); err != nil {
						slog.Error(err.Error())
						w.response(next.event, next.provider, "Error at lock pull request")
					} else {
						w.response(next.event, next.provider, "The pull request was locked successfully")
					}
				}
			case UnlockPullRequest:
				if isPrAlreadyUnlocked {
					slog.Warn("The pull request was already unlocked", "pullrequest", next.event.PullRequestId)
					w.response(next.event, next.provider, "The pull request was already unblocked")
				} else {
					slog.Info("Unlocking pull request", "pullrequest", next.event.PullRequestId)
					if err := w.unlock(next.event, apps); err != nil {
						slog.Error(err.Error())
						w.response(next.event, next.provider, "Error at unlock pull request")
					} else {
						w.response(next.event, next.provider, "The pull request was unlocked successfully")
					}
				}
			}

		case <-w.quit:
			return
		}
	}
}

func (w *Worker) Stop(ctx context.Context) {
	defer close(w.quit)
	for {
		select {
		default:
			time.Sleep(1 * time.Second)
			if w.queue.Size() <= 0 {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (w Worker) lock(e Event, apps []mApp.Application) error {
	for _, app := range apps {
		if _, err := mApp.LockApp(app, e.PullRequestSourceBranch, e.PullRequestId); err != nil {
			return err
		}
	}
	return nil
}

func (w Worker) unlock(e Event, apps []mApp.Application) error {
	for _, app := range apps {
		if app.Locked && app.PullRequestId == e.PullRequestId {
			_, err := mApp.UnlockApp(app)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (w Worker) response(e Event, p Provider, msg string) {
	if err := p.WriteComment(e.Repository, e.PullRequestId, e.CommentId, msg); err != nil {
		slog.Error(err.Error())
	}
}
