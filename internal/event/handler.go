package event

import (
	"gitbot/internal/types"
	"io"
	"log/slog"
	"net/http"
)

type Provider interface {
	//Name() string
	ParseEvent(headers http.Header, body io.ReadCloser) (types.Event, error)
	WriteComment(repo string, prId int, parentId int, msg string) error
	//	RespondEvent(e GitEvent, msg string) error
}

type Queue interface {
	Enqueue(item QueueItem)
	NextItem() *QueueItem
	Dequeue() *QueueItem
	Size() int
}

type QueueItem struct {
	Event   types.Event
	Handler Handler
}

type Handler struct {
	provider Provider
	queue    Queue
}

func NewHandler(q Queue, p Provider) *Handler {
	return &Handler{
		provider: p,
		queue:    q,
	}
}

func (h Handler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		/* Get event from webhook */
		e, err := h.provider.ParseEvent(r.Header, r.Body)
		if err != nil {
			slog.Error("Error at parse webhook")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		/* Put event in queue */
		item := QueueItem{Event: e, Handler: h}
		h.queue.Enqueue(item)

		/* Response */
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("{}"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (h Handler) ProcessEvent(e types.Event) {
	slog.Info("Processing new event", "type", e.Type)

	// Get if action is needed
	action :=

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
