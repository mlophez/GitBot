package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gitbot/internal/event"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

type BitbucketProvider struct {
	bearerToken string
}

func NewBitbucketProvider(token string) *BitbucketProvider {
	return &BitbucketProvider{
		bearerToken: token,
	}
}

func (b BitbucketProvider) ParseEvent(headers http.Header, body io.ReadCloser) (event.Event, error) {
	var webhook bpWebhookRequest
	var e event.Event

	err := json.NewDecoder(body).Decode(&webhook)
	if err != nil {
		return e, err
	}

	///* Parse Event */
	e.Repository = fmt.Sprintf("https://bitbucket.org/%s.git", webhook.Repository.FullName)
	e.Author = webhook.Actor.UUID
	e.PullRequest.Id = webhook.PullRequest.Id
	e.PullRequest.SourceBranch = webhook.PullRequest.Source.Branch.Name
	e.PullRequest.DestinationBranch = webhook.PullRequest.Destination.Branch.Name

	// Comment
	if webhook.Comment.Id > 0 && !webhook.Comment.Pending && !webhook.Comment.Deleted {
		e.CommentId = webhook.Comment.Id
		e.Comment = webhook.Comment.Content.Raw
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

	// Get Changelog
	filesChanged, err := b.GetFilesChanged(e.Repository, e.PullRequest.Id)
	if err != nil {
		return e, err
	}
	e.PullRequest.FilesChanged = filesChanged

	// Approved and Request changeds
	e.PullRequest.Approved = 0
	e.PullRequest.RequestChanged = 0
	for _, p := range webhook.PullRequest.Participants {
		if p.Role == "REVIEWER" && p.Approved {
			e.PullRequest.Approved++
		}

		if p.Role == "REVIEWER" && p.State == "changes_requested" {
			e.PullRequest.RequestChanged++
		}
	}

	return e, err
}

func (b BitbucketProvider) GetFilesChanged(repo string, pullRequestId int) ([]string, error) {
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/pullrequests/%d/diffstat", b.getSlug(repo), pullRequestId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []string{}, err
	}

	req.Header.Add("Authorization", "Bearer "+b.bearerToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error on response.\n[ERROR] -", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("Error while reading the response bytes:", err)
		}
		slog.Info(string([]byte(body)))
		return nil, fmt.Errorf("Error in get files changed")
	}

	var respJSON bpDiffStatResponse
	err = json.NewDecoder(resp.Body).Decode(&respJSON)
	if err != nil {
		return []string{}, err
	}

	var files []string
	for _, f := range respJSON.Values {
		if f.Old.Path == f.New.Path {
			files = append(files, f.Old.Path)
		} else {
			files = append(files, f.Old.Path)
			files = append(files, f.New.Path)
		}
	}

	return files, nil
}

func (b BitbucketProvider) WriteComment(repo string, prId int, parentId int, msg string) error {
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/pullrequests/%d/comments", b.getSlug(repo), prId)

	var payload []byte
	if parentId > 0 {
		var requestBody bpWriteCommentRequestParent
		requestBody.Parent.Id = parentId
		requestBody.Content.Raw = msg
		// requestBody.Content.Raw = "**" + msg + "**"
		p, err := json.Marshal(&requestBody)
		if err != nil {
			return err
		}
		payload = p
	} else {
		var requestBody bpWriteCommentRequest
		requestBody.Content.Raw = msg
		// requestBody.Content.Raw = "**" + msg + "**"
		p, err := json.Marshal(&requestBody)
		if err != nil {
			return err
		}
		payload = p
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+b.bearerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error on response.\n[ERROR] -", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		str, _ := io.ReadAll(resp.Body)
		slog.Info(string(str))
		return fmt.Errorf("Error in send comment to pull request")
	}

	return nil
}

func (b BitbucketProvider) GetAuthor(url string) (string, error) {
	author := ""
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return author, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+b.bearerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error on response.\n[ERROR] -", err)
	}
	defer resp.Body.Close()

	str, _ := io.ReadAll(resp.Body)
	slog.Info(string(str))

	//if resp.StatusCode != 201 {
	//	str, _ := io.ReadAll(resp.Body)
	//	slog.Info(string(str))
	//	return fmt.Errorf("Error in send comment to pull request")
	//}

	return author, nil
}

func (b BitbucketProvider) getSlug(repo string) string {
	// https://bitbucket.org/firmapro/platform-poc.git -> firmapro/platform-poc
	repo = strings.Replace(repo, "https://bitbucket.org/", "", -1)
	repo = strings.Replace(repo, ".git", "", -1)
	return repo
}

type bpWebhookRequest struct {
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

type bpDiffStatResponse struct {
	Values []struct {
		Old struct {
			Path string `json:"path"`
		} `json:"old"`
		New struct {
			Path string `json:"path"`
		} `json:"new"`
	} `json:"values"`
}

type bpWriteCommentRequestParent struct {
	//Type string `json:"type"`
	Parent struct {
		Id int `json:"id"`
	} `json:"parent"`
	Content struct {
		Raw string `json:"raw"`
		//Html string `json:"html"`
		//MarkUp string `json:"markup"`
	} `json:"content"`
}

type bpWriteCommentRequest struct {
	//Type string `json:"type"`
	Content struct {
		Raw string `json:"raw"`
		//Html string `json:"html"`
		//MarkUp string `json:"markup"`
	} `json:"content"`
}
