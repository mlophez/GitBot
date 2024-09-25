package notification

import (
	"context"
	"encoding/json"
	"gitbot/internal/providers/argocd"
	"log/slog"
	"net/http"
	"strings"

	"k8s.io/client-go/kubernetes"
)

type Notification struct {
	AppName string `json:"app_name"`
	Message string `json:"message"`
}

type NotificationProvider interface {
	WriteComment(repo string, prId int, parentId int, msg string) error
}

func HandleNotification(cs *kubernetes.Clientset, np NotificationProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		/* Get event from webhook */
		var n Notification
		err := json.NewDecoder(r.Body).Decode(&n)
		if err != nil {
			slog.Error("[notify] Notification bad request", "Body", r.Body)
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		slog.Info("[notify] New notification incoming", "AppName", n.AppName, "Message", n.Message)

		// Get App From cluster
		ctx := context.TODO()
		a, err := argocd.GetApplication(ctx, cs, n.AppName)
		if err != nil {
			slog.Error("[notify] Application not found", "AppName", n.AppName)
			http.Error(w, err.Error(), http.StatusNotFound)
		}

		if a.Locked {
			slog.Info("[notify] Sending Notification", "Repository", a.Repository, "PullRequestId", a.PullRequestId)
			err := np.WriteComment(a.Repository, a.PullRequestId, 0, "**["+strings.ToUpper(a.Environment)+"]** => "+n.Message)
			if err != nil {
				slog.Error("[notify] The notification could not be sent", "AppName", n.AppName, "PullRequestId", a.PullRequestId)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}
}
