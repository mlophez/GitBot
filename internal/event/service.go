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
	Message string
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

func (s Service) Process(e Event) (*Response, bool) {
	slog.Info("Processing new event", "type", e.Type)

	retry := false

	/* Get action, env is "all" if all environments */
	action, env := getActionFromEvent(e)
	if action == UNKNOWN_ACTION || env == nil {
		return nil, retry
	}

	switch action {
	case LOCK_ACTION:
		/* Check if action is permitted */
		if e.PullRequest.Approved == 0 && e.PullRequest.Reviewers != 0 {
			return &Response{Success: false, Message: "You need at least one approval from a reviewer"}, retry
		}

		if e.PullRequest.RequestChanged > 0 {
			return &Response{Success: false, Message: "One of the reviewers has requested changes"}, retry
		}

		if e.PullRequest.CommitsBehind > 0 {
			return &Response{Success: false, Message: fmt.Sprintf(
				"This pull request is %d commits behind '%s', sync your branch!", e.PullRequest.CommitsBehind, e.PullRequest.DestinationBranch)}, retry
		}

		/* Get apps from cluster */
		apps, err := s.appService.FindAppsByRepoAndFiles(e.Repository, e.PullRequest.FilesChanged)
		if err != nil {
			slog.Error("Error at try get apps", "error", err)
			return nil, retry
		}

		/* Filter by environment */
		apps = filterByEnv(apps, *env)
		if len(apps) == 0 {
			slog.Info("Not apps founds")
			return nil, retry
		}

		re := s.lockPullRequest(e.PullRequest, apps)
		// TODO: Double lock for apps of apps, check if already blocked, caution loop infinity
		// retry = IsAnyContainApps(apps)
		// if retry {
		// 	slog.Info("Some app has apps, retrying")
		// }
		return re, retry
	case UNLOCK_ACTION:
		/* Get apps from cluster */
		apps, err := s.appService.FindAppsByPrID(e.PullRequest.Id)
		if err != nil {
			slog.Error("Error at try get apps", "error", err)
			return nil, retry
		}

		/* Filter by environment */
		apps = filterByEnv(apps, *env)
		if len(apps) == 0 {
			slog.Info("Not apps founds")
			return nil, retry
		}

		return s.unlockPullRequest(e, e.PullRequest, apps), retry
	default:
		return nil, retry
	}
}

func getActionFromEvent(e Event) (Action, *string) {
	var env string

	switch e.Type {

	case EventTypeMerged:
		env = "all"
		return UNLOCK_ACTION, &env

	case EventTypeDeclined:
		env = "all"
		return UNLOCK_ACTION, &env

	case EventTypeCommented:
		var command string

		filter := regexp.MustCompile(`(?i)(/|#)(argo|flux|bot)\s(lock|deploy|test|unlock|undeploy|rollback)(?: (\w+))?`).FindStringSubmatch(e.Comment)
		if len(filter) > 3 {
			command = filter[3]
		}

		/* If #argo deploy environment, if environment not match ignore */
		if len(filter) > 4 {
			if filter[4] != "" {
				env = strings.ToLower(filter[4])
			}
		}

		switch strings.ToUpper(command) {
		case "LOCK", "DEPLOY", "TEST":
			return LOCK_ACTION, &env

		case "UNLOCK", "UNDEPLOY", "ROLLBACK":
			return UNLOCK_ACTION, &env
		}
	}

	return UNKNOWN_ACTION, nil
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

	// filter apps and only unlock the ones locked by this PR. Make new slice filtering by pr id
	newApps := make([]app.Application, 0)
	for _, a := range apps {
		if a.Locked && a.PullRequestId == pr.Id {
			newApps = append(newApps, a)
		}
	}
	apps = newApps

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

	/* In merge if suscesfully unlock no response */
	if resp.Success && (e.Type == EventTypeMerged || e.Type == EventTypeDeclined) {
		return nil
	}

	return &resp
}

func filterByEnv(apps []app.Application, envFilter string) []app.Application {
	var result []app.Application

	for _, app := range apps {
		if app.Environment == envFilter || envFilter == "all" {
			result = append(result, app.Sanitize())
		}
	}

	return result

}

func IsAnyContainApps(apps []app.Application) bool {
	for _, a := range apps {
		if a.ContainOther {
			return true
		}
	}
	return false
}
