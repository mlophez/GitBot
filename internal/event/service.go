package event

import (
	"fmt"
	"gitbot/internal/app"
	"log/slog"
	"regexp"
	"strings"
)

/*** Event Response ***/
type Response struct {
	Success bool
	Summary []AppStatus
}

type AppStatus struct {
	Name    string
	Message string
}

/*** Action Interface ***/
type Action int

const (
	LOCK_ACTION Action = iota
	UNLOCK_ACTION
	UNKNOWN_ACTION
)

/*** Service ***/
type Service struct {
	rules      []SecurityRule
	appService app.Service
}

func NewService(rules []SecurityRule, appService app.Service) Service {
	return Service{
		rules:      rules,
		appService: appService,
	}
}

func (s Service) Process(e Event) *Response {
	slog.Info("Processing new event", "type", e.Type)

	/* Get action */
	action := getActionFromEvent(e)
	if action == UNKNOWN_ACTION {
		return nil
	}

	/* Get apps from cluster */
	apps, err := s.appService.FindAppsByRepoAndFiles(e.Repository, e.PullRequest.FilesChanged)
	if err != nil {
		slog.Error("Error at try get apps", "error", err)
		return nil
	} else if len(apps) == 0 {
		slog.Info("Not apps founds")
		return nil
	}

	switch action {
	case LOCK_ACTION:
		return s.lockPullRequest(e.PullRequest, apps)
	case UNLOCK_ACTION:
		return s.unlockPullRequest(e, e.PullRequest, apps)
	default:
		return nil
	}
}

func getActionFromEvent(e Event) Action {
	switch e.Type {

	case EventTypeMerged:
		return UNLOCK_ACTION

	case EventTypeDeclined:
		return UNLOCK_ACTION

	case EventTypeCommented:
		var command string

		filter := regexp.MustCompile(`(?i)(/|#)(argo|flux|bot)\s(lock|deploy|test|unlock|undeploy|rollback)`).FindStringSubmatch(e.Comment)
		if len(filter) == 4 {
			command = filter[3]
		}

		switch strings.ToUpper(command) {
		case "LOCK", "DEPLOY", "TEST":
			return LOCK_ACTION

		case "UNLOCK", "UNDEPLOY", "ROLLBACK":
			return UNLOCK_ACTION
		}
	}

	return UNKNOWN_ACTION
}

func (s Service) lockPullRequest(pr PullRequest, apps []app.Application) *Response {
	resp := Response{Success: true}

	isAnyLockedByAnother := false
	isAnyTargetRevisionDontMatch := false
	// hasPermissionForAll := false

	for _, a := range apps {
		switch {
		// LOCKED_BY_ANOTHER
		case a.Locked && a.PullRequestId != pr.Id:
			isAnyLockedByAnother = true
			resp.Summary = append(resp.Summary, AppStatus{
				Name:    a.Name,
				Message: fmt.Sprintf("This app is blocked by another pr (%d)", a.PullRequestId),
			})
		// LOCKED_BY_ME
		case a.Locked:
			resp.Summary = append(resp.Summary, AppStatus{
				Name:    a.Name,
				Message: "Locked",
			})
		// TARGET_BRANCH_NOT_MATCH
		case a.Branch != pr.DestinationBranch:
			isAnyTargetRevisionDontMatch = true
			resp.Summary = append(resp.Summary, AppStatus{
				Name:    a.Name,
				Message: fmt.Sprintf("App with branch '%s' dont match with pull request target branch '%s')", a.Branch, pr.DestinationBranch),
			})
		// UNLOCKED
		default:
			resp.Summary = append(resp.Summary, AppStatus{
				Name:    a.Name,
				Message: "Unlocked",
			})
		}
	}

	//if isAnyTargetRevisionDontMatch || isAnyLockedByAnother || !hasPermissionForAll {
	if isAnyTargetRevisionDontMatch || isAnyLockedByAnother {
		resp.Success = false
		return &resp
	}

	for i, a := range apps {
		if !a.Locked {
			slog.Info("Locking application", "app", a.Name)
			err := s.appService.LockApp(a, pr.SourceBranch, pr.Id)
			if err != nil {
				slog.Error("Error at locking app", "error", err)
				resp.Success = false
				resp.Summary[i].Message = "Error at lock application"
				return &resp
			}

			resp.Summary[i].Message = "Locked"
		}
	}

	return &resp
}

func (s Service) unlockPullRequest(e Event, pr PullRequest, apps []app.Application) *Response {
	resp := Response{Success: true}

	isAnyLockedByMe := false
	// hasPermissionForAll := false

	for _, a := range apps {
		switch {
		// LOCKED_BY_ANOTHER
		case a.Locked && a.PullRequestId != pr.Id:
			resp.Summary = append(resp.Summary, AppStatus{
				Name:    a.Name,
				Message: fmt.Sprintf("This app is blocked by another pr (%d)", a.PullRequestId),
			})
		// LOCKED_BY_ME
		case a.Locked:
			isAnyLockedByMe = true
			resp.Summary = append(resp.Summary, AppStatus{
				Name:    a.Name,
				Message: "Locked",
			})
			// UNLOCKED
		default:
			resp.Summary = append(resp.Summary, AppStatus{
				Name:    a.Name,
				Message: "Unlocked",
			})
		}
	}

	if !isAnyLockedByMe {
		if e.Type == EventTypeMerged || e.Type == EventTypeDeclined {
			return nil
		}
		return &resp
	}

	//if !hasPermissionForAll {
	//	resp.Success = false
	//	return &resp
	//}

	for i, a := range apps {
		if a.Locked {
			err := s.appService.UnlockApp(a)
			if err != nil {
				slog.Error("Error at unlocking app", "error", err)
				resp.Success = false
				resp.Summary[i].Message = "Error at unlock application"
				return &resp
			}

			resp.Summary[i].Message = "Unlocked"
		}
	}

	return &resp
}
