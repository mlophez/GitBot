package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	. "gitbot/types"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

type bitbucket struct {
	bearerToken string
}

type bitbucketWebhook struct {
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
}

func NewBitbucketProvider(token string) *bitbucket {
	return &bitbucket{
		bearerToken: token,
	}
}

func (b bitbucket) ParseEvent(headers http.Header, body io.ReadCloser) (Event, error) {
	var webhook bitbucketWebhook
	var e Event

	err := json.NewDecoder(body).Decode(&webhook)
	if err != nil {
		return e, err
	}

	///* Parse Event */
	e.Repository = fmt.Sprintf("https://bitbucket.org/%s.git", webhook.Repository.FullName)
	e.Author = "none@logalty.com"
	e.PullRequestId = webhook.PullRequest.Id
	e.PullRequestSourceBranch = webhook.PullRequest.Source.Branch.Name
	e.PullRequestDestinationBranch = webhook.PullRequest.Destination.Branch.Name
	e.PullRequestRepoSlug = webhook.Repository.FullName

	// Comment
	if webhook.Comment.Id > 0 && !webhook.Comment.Pending && !webhook.Comment.Deleted {
		e.CommentId = webhook.Comment.Id
		e.Comment = webhook.Comment.Content.Raw
	}

	// EventType
	eventKey := headers.Get("X-Event-Key")
	switch strings.ToLower(eventKey) {
	case "pullrequest:created":
		e.Type = EventTypeOpened
	case "pullrequest:updated":
		e.Type = EventTypeUpdated
	case "pullrequest:fulfilled":
		e.Type = EventTypeMerged
	case "pullrequest:rejected":
		e.Type = EventTypeDeclined
	case "pullrequest:comment_created":
		e.Type = EventTypeCommented
	default:
		e.Type = EventTypeUnknown
	}

	// Get Changelog
	filesChanged, err := b.getPrFilesChanged(e.PullRequestRepoSlug, e.PullRequestId)
	if err != nil {
		return e, err
	}
	e.PullRequestFilesChanged = filesChanged

	return e, err
}

func (b bitbucket) getPrFilesChanged(fullName string, pullRequestId int) ([]string, error) {
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/pullrequests/%d/diffstat", fullName, pullRequestId)

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

	type Response struct {
		Values []struct {
			Old struct {
				Path string `json:"path"`
			} `json:"old"`
			New struct {
				Path string `json:"path"`
			} `json:"new"`
		} `json:"values"`
	}
	var respJSON Response
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

func (b bitbucket) WriteComment(repo string, prId int, parentId int, msg string) error {
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/pullrequests/%d/comments", getSlug(repo), prId)

	slog.Info("", "url", url)

	type RequestWithParent struct {
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

	type Request struct {
		//Type string `json:"type"`
		Content struct {
			Raw string `json:"raw"`
			//Html string `json:"html"`
			//MarkUp string `json:"markup"`
		} `json:"content"`
	}

	var payload []byte
	if parentId > 0 {
		var requestBody RequestWithParent
		requestBody.Parent.Id = parentId
		requestBody.Content.Raw = "**" + msg + "**"
		p, err := json.Marshal(&requestBody)
		if err != nil {
			return err
		}
		payload = p
	} else {
		var requestBody Request
		requestBody.Content.Raw = "**" + msg + "**"
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

func getSlug(repo string) string {
	// https://bitbucket.org/firmapro/platform-poc.git -> firmapro/platform-poc
	repo = strings.Replace(repo, "https://bitbucket.org/", "", -1)
	repo = strings.Replace(repo, ".git", "", -1)
	return repo
}
