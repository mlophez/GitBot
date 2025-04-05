package hook

import (
	"encoding/json"
	"gitbot/internal/hook/types"
	"io"
	"net/http"
)

type BitbucketHookRequest struct {
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	PullRequest struct {
		Id     int    `json:"id"`
		Title  string `json:"title"`
		State  string `json:"state"`
		Source struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
		} `json:"source"`
		Destination struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
		} `json:"destination"`
		Participants []struct {
			Role     string `json:"role"`
			Approved bool   `json:"approved"`
			State    string `json:"state"`
		} `json:"participants"`
	} `json:"pullrequest"`
	Comment struct {
		Id      int    `json:"id"`
		Type    string `json:"type"`
		Deleted bool   `json:"deleted"`
		Pending bool   `json:"pending"`
		Content struct {
			Raw string `json:"raw"`
		} `json:"content"`
	} `json:"comment"`
	Actor struct {
		UUID string `json:"uuid"`
	} `json:"actor"`
}

type BitbucketClient struct {
	BearerToken string
}

func (bb BitbucketClient) Parse(h http.Header, bdy io.ReadCloser) (types.GitHook, error) {
	var hook BitbucketClient
	var e types.GitHook

 	err := json.NewDecoder(bdy).Decode(&hook)
	if err != nil {
		return e, err
	}

	///* Parse Event */
	e.Repository = fmt.Sprintf("https://bitbucket.org/%s.git", hook.Repository.FullName)
	e.Author = hook.Actor.UUID
	e.PullRequest.Id = hook.PullRequest.Id
	e.PullRequest.SourceBranch = hook.PullRequest.Source.Branch.Name
	e.PullRequest.DestinationBranch = hook.PullRequest.Destination.Branch.Name

	// Comment
	if hook.Comment.Id > 0 && !hook.Comment.Pending && !hook.Comment.Deleted {
		e.CommentId = hook.Comment.Id
		e.Comment = hook.Comment.Content.Raw
	}

	// EventType
	eventKey := headers.Get("X-Event-Key")
	switch strings.ToLower(eventKey) {
	case "pullrequest:created":
		e.Type = event.EventTypeOpened
	case "pullrequest:updated":
		e.Type = event.EventTypeUpdated
	case "pullrequest:fulfilled":
		e.Type = event.EventTypeMerged
	case "pullrequest:rejected":
		e.Type = event.EventTypeDeclined
	case "pullrequest:comment_created":
		e.Type = event.EventTypeCommented
	default:
		e.Type = event.EventTypeUnknown
	}

	// Approved and Request changeds
	e.PullRequest.Approved = 0
	e.PullRequest.RequestChanged = 0
	for _, p := range hook.PullRequest.Participants {
		if p.Role == "REVIEWER" {
			e.PullRequest.Reviewers++
		}

		if p.Role == "REVIEWER" && p.Approved {
			e.PullRequest.Approved++
		}

		if p.Role == "REVIEWER" && p.State == "changes_requested" {
			e.PullRequest.RequestChanged++
		}
	}

	return e, err
}


