package adapters

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gitbot/internal/types"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// BitbucketClient implements types.Provider for Bitbucket webhooks and API calls.
// It parses incoming webhook payloads, enriches events with diff and commit data,
// and writes comments back to pull requests via the Bitbucket REST API.
type BitbucketClient struct {
	bearerToken string
}

// NewBitbucketClient creates a BitbucketClient authenticated with the given bearer token.
func NewBitbucketClient(token string) *BitbucketClient {
	return &BitbucketClient{
		bearerToken: token,
	}
}

// ValidateWebhookToken verifies the HMAC-SHA256 signature that Bitbucket includes
// in the X-Hub-Signature header (format: "sha256=<hex>").
// Returns nil when secret is empty (validation disabled) or the signature matches.
// Returns an error if the header is missing or the signature does not match.
func (b BitbucketClient) ValidateWebhookToken(secret string, headers http.Header, body []byte) error {
	if secret == "" {
		return nil
	}
	sig := headers.Get("X-Hub-Signature")
	if sig == "" {
		return fmt.Errorf("missing X-Hub-Signature header")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return fmt.Errorf("webhook signature mismatch")
	}
	return nil
}

// ParseEvent parses a Bitbucket webhook request into a types.Event.
// Returns an error if the request body cannot be decoded.
func (b BitbucketClient) ParseEvent(headers http.Header, body io.ReadCloser) (types.Event, error) {
	var webhook bpWebhookRequest
	var e types.Event

	err := json.NewDecoder(body).Decode(&webhook)
	if err != nil {
		return e, err
	}

	e.Repository = fmt.Sprintf("https://bitbucket.org/%s.git", webhook.Repository.FullName)
	e.Author = webhook.Actor.UUID
	e.PullRequest.Id = webhook.PullRequest.Id
	e.PullRequest.SourceBranch = webhook.PullRequest.Source.Branch.Name
	e.PullRequest.DestinationBranch = webhook.PullRequest.Destination.Branch.Name

	if webhook.Comment.Id > 0 && !webhook.Comment.Pending && !webhook.Comment.Deleted {
		e.CommentId = webhook.Comment.Id
		e.Comment = webhook.Comment.Content.Raw
	}

	eventKey := headers.Get("X-Event-Key")
	switch strings.ToLower(eventKey) {
	case "pullrequest:created":
		e.Type = types.EventTypeOpened
	case "pullrequest:updated":
		e.Type = types.EventTypeUpdated
	case "pullrequest:fulfilled":
		e.Type = types.EventTypeMerged
	case "pullrequest:rejected":
		e.Type = types.EventTypeDeclined
	case "pullrequest:comment_created":
		e.Type = types.EventTypeCommented
	default:
		e.Type = types.EventTypeUnknown
	}

	e.PullRequest.Approved = 0
	e.PullRequest.RequestChanged = 0
	for _, p := range webhook.PullRequest.Participants {
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

// GetData enriches an event with the files changed and commits behind from the Bitbucket API.
func (b BitbucketClient) GetData(e types.Event) (types.Event, error) {
	filesChanged, err := b.GetFilesChanged(e.Repository, e.PullRequest.Id)
	if err != nil {
		return e, err
	}
	e.PullRequest.FilesChanged = filesChanged

	commitsBehind, err := b.CompareBranchCommitTotal(e.Repository, e.PullRequest.DestinationBranch, e.PullRequest.SourceBranch)
	if err != nil {
		return e, err
	}
	e.PullRequest.CommitsBehind = commitsBehind

	return e, nil
}

// GetFilesChanged returns the list of file paths modified in the given pull request.
func (b BitbucketClient) GetFilesChanged(repo string, pullRequestId int) ([]string, error) {
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/pullrequests/%d/diffstat", b.getSlug(repo), pullRequestId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []string{}, err
	}
	req.Header.Add("Authorization", "Bearer "+b.bearerToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		slog.Info(string(body))
		return nil, fmt.Errorf("GetFilesChanged: unexpected status %d", resp.StatusCode)
	}

	var respJSON bpDiffStatResponse
	if err := json.NewDecoder(resp.Body).Decode(&respJSON); err != nil {
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

// WriteComment posts a comment on the pull request. If parentId > 0 the comment
// is posted as a reply to that comment thread.
func (b BitbucketClient) WriteComment(repo string, prId int, parentId int, msg string) error {
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/pullrequests/%d/comments", b.getSlug(repo), prId)

	var payload []byte
	if parentId > 0 {
		var body bpWriteCommentRequestParent
		body.Parent.Id = parentId
		body.Content.Raw = msg
		p, err := json.Marshal(&body)
		if err != nil {
			return err
		}
		payload = p
	} else {
		var body bpWriteCommentRequest
		body.Content.Raw = msg
		p, err := json.Marshal(&body)
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
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		str, _ := io.ReadAll(resp.Body)
		slog.Info(string(str))
		return fmt.Errorf("WriteComment: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// WriteEventResponse formats resp as a Markdown comment and posts it on the pull request.
// clusterName is used as a fallback display label when apps carry no cluster information.
func (b BitbucketClient) WriteEventResponse(repo string, prId int, parentId int, resp *types.EventResponse, clusterName string) error {
	return b.WriteComment(repo, prId, parentId, b.formatEventResponse(resp, clusterName))
}

// formatEventResponse renders an EventResponse as a Markdown string suitable for
// posting as a Bitbucket pull request comment.
func (b BitbucketClient) formatEventResponse(resp *types.EventResponse, clusterName string) string {
	if resp.Environments != nil {
		return bbFormatHelp(resp.Environments)
	}

	if resp.Message != "" {
		return fmt.Sprintf("### Status: **%s**\n\n%s.  \n", bbStatus(resp.Success), resp.Message)
	}

	grouped := bbGroupByCluster(resp.Summary)

	// Single cluster (or no cluster tag): compact header.
	if len(grouped) <= 1 {
		name := clusterName
		for k := range grouped {
			if k != "" {
				name = k
			}
		}
		var msg string
		if name != "" {
			msg = fmt.Sprintf("**[%s]** => **%s**\n\n", strings.ToUpper(name), bbStatusUpper(resp.Success))
		} else {
			msg = fmt.Sprintf("### Status: **%s**\n\n", bbStatus(resp.Success))
		}
		for _, app := range resp.Summary {
			msg += fmt.Sprintf("- **%s:** %s.  \n", strings.ToUpper(app.Name), app.Message)
		}
		return msg
	}

	// Multiple clusters: one section per cluster.
	var msg string
	for _, name := range bbSortedKeys(grouped) {
		msg += fmt.Sprintf("**[%s]** => **%s**\n\n", strings.ToUpper(name), bbStatusUpper(resp.Success))
		for _, app := range grouped[name] {
			msg += fmt.Sprintf("- **%s:** %s.  \n", strings.ToUpper(app.Name), app.Message)
		}
		msg += "\n"
	}
	return msg
}

func bbFormatHelp(envs []string) string {
	var envDisplay string
	if len(envs) == 0 {
		envDisplay = "_none found_"
	} else {
		quoted := make([]string, len(envs))
		for i, e := range envs {
			quoted[i] = "`" + e + "`"
		}
		envDisplay = strings.Join(quoted, ", ")
	}
	return "### GitBot Help\n\n" +
		"**Available environments:** " + envDisplay + "\n\n" +
		"**Commands:**\n\n" +
		"| Command | Description |\n" +
		"|---------|-------------|\n" +
		"| `#argo deploy` | Lock apps in all environments |\n" +
		"| `#argo deploy <env>` | Lock apps in a specific environment |\n" +
		"| `#argo deploy <env> <app>` | Lock a specific app |\n" +
		"| `#argo unlock` | Unlock apps in all environments |\n" +
		"| `#argo unlock <env>` | Unlock apps in a specific environment |\n" +
		"| `#argo help` | Show this help |\n"
}

func bbGroupByCluster(apps []types.EventAppStatus) map[string][]types.EventAppStatus {
	result := make(map[string][]types.EventAppStatus)
	for _, a := range apps {
		result[a.Cluster] = append(result[a.Cluster], a)
	}
	return result
}

func bbSortedKeys(m map[string][]types.EventAppStatus) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

func bbStatus(success bool) string {
	if success {
		return "Success"
	}
	return "Failed"
}

func bbStatusUpper(success bool) string {
	if success {
		return "SUCCESS"
	}
	return "FAILED"
}

// CompareBranchCommitTotal returns the number of commits that exclude is behind include.
func (b BitbucketClient) CompareBranchCommitTotal(repository string, include string, exclude string) (int, error) {
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/commits?include=%s&exclude=%s", b.getSlug(repository), include, exclude)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+b.bearerToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("CompareBranchCommitTotal: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Values []struct{} `json:"values"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	return len(result.Values), nil
}

func (b BitbucketClient) getSlug(repo string) string {
	repo = strings.Replace(repo, "https://bitbucket.org/", "", -1)
	repo = strings.Replace(repo, ".git", "", -1)
	return repo
}

// ── Bitbucket API request/response types ─────────────────────────────────────

type bpWebhookRequest struct {
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	PullRequest struct {
		Id    int    `json:"id"`
		Title string `json:"title"`
		State string `json:"state"`
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
	Parent struct {
		Id int `json:"id"`
	} `json:"parent"`
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
}

type bpWriteCommentRequest struct {
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
}
