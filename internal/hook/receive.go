package hook

import (
	"io"
	"log/slog"
	"net/http"
	"gitbot/internal/hook/types"
)

type IGitProvider interface {
	ParseHook(h http.Header, b io.ReadCloser) (Event, error)
	WriteMarkdownCommentInPR(rpo string, id int, msg string) error
}

type HookParser interface {
	Parse(body []byte) (*types.GitHook, error)
}

type ReceiveHookUseCase struct {
	HookParser HookParser
}

func (s *ReceiveHookUseCase) ReceiveEvent(p ParseEvent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		// 1. Get event from webhook
		e, err := s.HookParser.Parse(r.Header, r.Body)
		if err != nil {
			slog.Error("Error at parse webhook")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Route to sub-use case
		//switch e.Type {
		//case EventTypeOpened:
		//	err = s.ReceiveEventOpened(e)
		//case EventTypeUpdated:
		//	// TODO: Implementar lógica para eventos actualizados
		//	slog.Warn("EventTypeUpdated received but not implemented")
		//case EventTypeCommented:
		//	// TODO: Implementar lógica para eventos comentados
		//	slog.Warn("EventTypeCommented received but not implemented")
		//case EventTypeMerged:
		//	// TODO: Implementar lógica para eventos fusionados
		//	slog.Warn("EventTypeMerged received but not implemented")
		//case EventTypeDeclined:
		//	// TODO: Implementar lógica para eventos rechazados
		//	slog.Warn("EventTypeDeclined received but not implemented")
		//default:
		//	slog.Warn("Unknown event type")
		//	http.Error(w, "Unknown event type", http.StatusBadRequest)
		//	return
		//}

		//if err != nil {
		//	slog.Error("Error processing event:", err)
		//	http.Error(w, err.Error(), http.StatusInternalServerError)
		//	return
		//}

		//// Default response
		//w.Header().Set("Content-Type", "application/json")
		//w.WriteHeader(http.StatusOK)
		//_, err = w.Write([]byte("{}"))
		//if err != nil {
		//	http.Error(w, err.Error(), http.StatusInternalServerError)
		//}
	}
}

func (s *ReceiveHookUseCase) ReceiveEventOpened(e Event) error {
	// 1. Get all enviroments
	// 2. Write help comment
	msg := `**The lifecycle of this pull request is managed by the Git bot!**
Enviroments: dev, test, demo, demosign, prod, santander, bbva, mercedes
The following commands are available:
- argo deploy all
- argo deploy <env>
- argo rollback all
- argo rollback <env>
`
	if err := s.gitp.WriteMarkdownCommentInPR(e.Repository, e.PullRequestID, msg); err != nil {
		return err
	}

	return nil
}
