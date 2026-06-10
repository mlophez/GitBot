package event

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// bitbucketBody builds a minimal Bitbucket webhook JSON payload with the given
// repository full name and pull request id, enough to exercise ParseEvent.
func bitbucketBody(fullName string, prID int) io.ReadCloser {
	json := `{
		"repository": {"full_name": "` + fullName + `"},
		"pullrequest": {"id": ` + strconv.Itoa(prID) + `, "source": {"branch": {"name": "feat"}}, "destination": {"branch": {"name": "main"}}},
		"actor": {"uuid": "{abc}", "display_name": "dev"}
	}`
	return io.NopCloser(strings.NewReader(json))
}

// headerWithEventKey builds an http.Header with the X-Event-Key field set to key,
// simulating the header that Bitbucket attaches to every webhook request.
func headerWithEventKey(key string) http.Header {
	h := http.Header{}
	h.Set("X-Event-Key", key)
	return h
}

// TestParseEventValidatesAtCreation checks that ParseEvent maps the webhook payload
// into a domain Event and validates it at this creation point: a well-formed webhook
// yields the mapped event, a recognized PR event carrying an invalid pull request
// returns a validation error, and an irrelevant (unknown-type) webhook is accepted
// even without a pull request so it can be discarded downstream.
func TestParseEventValidatesAtCreation(t *testing.T) {
	client := NewBitbucketClient("token", "")

	t.Run("maps and accepts a created event", func(t *testing.T) {
		e, err := client.ParseEvent(headerWithEventKey("pullrequest:created"), bitbucketBody("org/repo", 5))
		if err != nil {
			t.Fatalf("ParseEvent() error = %v, want nil", err)
		}
		if e.Type != EventTypeOpened {
			t.Errorf("Type = %d, want EventTypeOpened", e.Type)
		}
		if e.Repository != "https://bitbucket.org/org/repo.git" {
			t.Errorf("Repository = %q", e.Repository)
		}
		if e.PullRequest.Id != 5 {
			t.Errorf("PullRequest.Id = %d, want 5", e.PullRequest.Id)
		}
	})

	t.Run("recognized event with invalid pull request returns error", func(t *testing.T) {
		_, err := client.ParseEvent(headerWithEventKey("pullrequest:fulfilled"), bitbucketBody("org/repo", 0))
		if err == nil {
			t.Fatal("ParseEvent() error = nil, want validation error")
		}
	})

	t.Run("missing repository returns error (no fabricated URL)", func(t *testing.T) {
		// A payload without repository full_name must not yield a hollow
		// "https://bitbucket.org/.git" URL that passes validation.
		_, err := client.ParseEvent(headerWithEventKey("pullrequest:created"), bitbucketBody("", 5))
		if err == nil {
			t.Fatal("ParseEvent() error = nil, want validation error for missing repository")
		}
	})

	t.Run("irrelevant unknown event is accepted", func(t *testing.T) {
		_, err := client.ParseEvent(headerWithEventKey("repo:push"), bitbucketBody("org/repo", 0))
		if err != nil {
			t.Fatalf("ParseEvent() error = %v, want nil for unknown event", err)
		}
	})
}
