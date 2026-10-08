package event

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"gitbot/internal/app"
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

// bbListPR renders one pull request object of the Bitbucket listing response.
func bbListPR(id int, draft bool, target string) string {
	return fmt.Sprintf(`{"id": %d, "title": "PR %d", "state": "OPEN", "draft": %t,
		"author": {"nickname": "jdoe"},
		"source": {"branch": {"name": "feature/%d"}, "commit": {"hash": "abc123def456"}},
		"destination": {"branch": {"name": %q}}}`, id, id, draft, id, target)
}

// bbPage is one page served by the fake Bitbucket API: status overrides the 200
// answer, raw replaces the generated body, otherwise values (comma-separated pull
// request objects) are wrapped in a listing response with its "next" link.
type bbPage struct {
	status int
	values string // comma-separated pull request objects
	raw    string // raw body, used instead of values when non-empty
}

// bbListServer is a fake Bitbucket API that serves the given pages in order (page N
// answers ...pullrequests?page=N, with "next" chaining to N+1). It records how many
// requests it received, the last Authorization header and the first query string.
type bbListServer struct {
	*httptest.Server
	requests int
	auth     string
	query    string
}

// newBBListServer starts a bbListServer for pages and closes it when the test ends.
func newBBListServer(t *testing.T, pages []bbPage) *bbListServer {
	t.Helper()
	s := &bbListServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.requests++
		s.auth = r.Header.Get("Authorization")
		if s.query == "" {
			s.query = r.URL.RawQuery
		}
		n, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if n == 0 {
			n = 1
		}
		p := pages[n-1]
		if p.status != 0 && p.status != http.StatusOK {
			w.WriteHeader(p.status)
			return
		}
		if p.raw != "" {
			io.WriteString(w, p.raw)
			return
		}
		next := ""
		if n < len(pages) {
			next = fmt.Sprintf("%s/2.0/repositories/firmapro/platform/pullrequests?state=OPEN&pagelen=50&page=%d", s.URL, n+1)
		}
		fmt.Fprintf(w, `{"values": [%s], "next": %q}`, p.values, next)
	}))
	t.Cleanup(s.Close)
	return s
}

// client returns a BitbucketClient pointed at the fake API.
func (s *bbListServer) client() BitbucketClient {
	return BitbucketClient{bearerToken: "secret", apiBaseURL: s.URL + "/2.0"}
}

// TestListOpenPullRequests checks the listing adapter: it follows pagination to
// the end, translates every field including the draft flag, and returns an error
// (never a partial list) on any failure, so the preview generator can fail closed.
func TestListOpenPullRequests(t *testing.T) {
	ctx := context.Background()

	t.Run("follows every page and maps all fields", func(t *testing.T) {
		srv := newBBListServer(t, []bbPage{
			{values: bbListPR(1, false, "dev") + "," + bbListPR(2, true, "dev")},
			{values: bbListPR(3, false, "main")},
			{values: bbListPR(4, false, "dev")},
		})

		prs, err := srv.client().ListOpenPullRequests(ctx, "firmapro", "platform")
		if err != nil {
			t.Fatalf("ListOpenPullRequests() error = %v", err)
		}
		if srv.requests != 3 {
			t.Errorf("requests = %d, want 3", srv.requests)
		}
		if len(prs) != 4 {
			t.Fatalf("got %d pull requests, want 4", len(prs))
		}
		if srv.auth != "Bearer secret" {
			t.Errorf("Authorization = %q, want Bearer secret", srv.auth)
		}
		if !strings.Contains(srv.query, "state=OPEN") {
			t.Errorf("first query = %q, want state=OPEN", srv.query)
		}

		first := prs[0]
		if first.Number != 1 || first.Title != "PR 1" || first.Branch != "feature/1" ||
			first.TargetBranch != "dev" || first.HeadSHA != "abc123def456" || first.Author != "jdoe" ||
			first.State != app.PullRequestStateOpen || first.Draft {
			t.Errorf("first = %+v, unexpected mapping", first)
		}
		if !prs[1].Draft {
			t.Errorf("prs[1].Draft = false, want true")
		}
		if prs[2].TargetBranch != "main" {
			t.Errorf("prs[2].TargetBranch = %q, want main", prs[2].TargetBranch)
		}
	})

	for _, status := range []int{http.StatusUnauthorized, http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(fmt.Sprintf("status %d on first page returns error", status), func(t *testing.T) {
			srv := newBBListServer(t, []bbPage{{status: status}})

			prs, err := srv.client().ListOpenPullRequests(ctx, "firmapro", "platform")
			if err == nil || prs != nil {
				t.Errorf("got (%v, %v), want (nil, error)", prs, err)
			}
		})
	}

	failingSecondPages := []struct {
		name string
		page bbPage
	}{
		{"server error", bbPage{status: http.StatusInternalServerError}},
		{"invalid JSON", bbPage{raw: `{"values": [`}},
		{"invalid pull request", bbPage{values: bbListPR(0, false, "dev")}},
	}
	for _, tt := range failingSecondPages {
		t.Run("second page "+tt.name+" returns error, not a partial list", func(t *testing.T) {
			srv := newBBListServer(t, []bbPage{{values: bbListPR(1, false, "dev")}, tt.page})

			prs, err := srv.client().ListOpenPullRequests(ctx, "firmapro", "platform")
			if err == nil || prs != nil {
				t.Errorf("got (%v, %v), want (nil, error)", prs, err)
			}
		})
	}

	t.Run("next page outside the API is not followed", func(t *testing.T) {
		foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("foreign host was called with Authorization %q", r.Header.Get("Authorization"))
		}))
		defer foreign.Close()
		srv := newBBListServer(t, []bbPage{{raw: fmt.Sprintf(`{"values": [%s], "next": %q}`, bbListPR(1, false, "dev"), foreign.URL+"/2.0/x")}})

		prs, err := srv.client().ListOpenPullRequests(ctx, "firmapro", "platform")
		if err == nil || prs != nil {
			t.Errorf("got (%v, %v), want (nil, error)", prs, err)
		}
	})
}
