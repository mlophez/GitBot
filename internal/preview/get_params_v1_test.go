package preview

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stubLister is a PullRequestLister test double that returns a preset list (or
// error) and counts how many times it was called.
type stubLister struct {
	prs   []PullRequest
	err   error
	calls int
}

func (s *stubLister) ListOpenPullRequests(ctx context.Context, workspace, repo string) ([]PullRequest, error) {
	s.calls++
	return s.prs, s.err
}

// validBody is a well-formed plugin generator request for firmapro/platform → dev.
const validBody = `{"applicationSetName":"platform-preview","input":{"parameters":{"workspace":"firmapro","repo":"platform","targetBranch":"dev"}}}`

// callGetParams runs the GetParams handler with the given body and returns the recorder.
func callGetParams(lister PullRequestLister, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/getparams.execute", strings.NewReader(body))
	rec := httptest.NewRecorder()
	GetParams(lister)(rec, req)
	return rec
}

// decodeNumbers decodes a successful response and returns the "number" of every
// parameter set, failing the test if the body does not match the plugin contract.
func decodeNumbers(t *testing.T, rec *httptest.ResponseRecorder) []string {
	t.Helper()
	var resp struct {
		Output struct {
			Parameters []map[string]string `json:"parameters"`
		} `json:"output"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Output.Parameters == nil {
		t.Fatalf("output.parameters is null or missing, want an array")
	}
	numbers := make([]string, 0, len(resp.Output.Parameters))
	for _, p := range resp.Output.Parameters {
		numbers = append(numbers, p["number"])
	}
	return numbers
}

// TestGetParams checks the plugin generator contract end to end against a stub
// provider: eligible pull requests are returned with native parameter names,
// provider errors never yield a (possibly empty) list, and invalid input is
// rejected before calling the provider.
func TestGetParams(t *testing.T) {
	t.Run("returns only open, ready pull requests targeting dev", func(t *testing.T) {
		draft := openPR(2)
		draft.Draft = true
		other := openPR(3)
		other.TargetBranch = "main"
		lister := &stubLister{prs: []PullRequest{openPR(1), draft, other}}

		rec := callGetParams(lister, validBody)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		if got := decodeNumbers(t, rec); len(got) != 1 || got[0] != "1" {
			t.Errorf("numbers = %v, want [1]", got)
		}
	})

	t.Run("no eligible pull requests yields an empty array", func(t *testing.T) {
		rec := callGetParams(&stubLister{}, validBody)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"parameters":[]`) {
			t.Errorf("body = %s, want parameters as []", rec.Body.String())
		}
	})

	t.Run("provider error returns 502 and never a parameter list", func(t *testing.T) {
		// With prune: true an empty or partial list would delete previews; an
		// error status makes ArgoCD keep them and retry.
		lister := &stubLister{prs: []PullRequest{openPR(1)}, err: errors.New("bitbucket: unexpected status 429")}

		rec := callGetParams(lister, validBody)

		if rec.Code != http.StatusBadGateway {
			t.Errorf("status = %d, want 502", rec.Code)
		}
		if body := rec.Body.String(); strings.Contains(body, "output") || strings.Contains(body, "parameters") {
			t.Errorf("body = %q, must not contain a parameter list", body)
		}
	})

	t.Run("draft appears once marked ready and disappears when back to draft", func(t *testing.T) {
		pr := openPR(7)
		pr.Draft = true
		lister := &stubLister{prs: []PullRequest{pr}}

		// Poll 1: draft → no preview.
		if got := decodeNumbers(t, callGetParams(lister, validBody)); len(got) != 0 {
			t.Errorf("poll 1 (draft) numbers = %v, want []", got)
		}

		// Poll 2: marked ready → preview generated.
		pr.Draft = false
		lister.prs = []PullRequest{pr}
		if got := decodeNumbers(t, callGetParams(lister, validBody)); len(got) != 1 || got[0] != "7" {
			t.Errorf("poll 2 (ready) numbers = %v, want [7]", got)
		}

		// Poll 3: back to draft → preview no longer generated (accepted behaviour).
		pr.Draft = true
		lister.prs = []PullRequest{pr}
		if got := decodeNumbers(t, callGetParams(lister, validBody)); len(got) != 0 {
			t.Errorf("poll 3 (draft again) numbers = %v, want []", got)
		}
	})

	badInputs := []struct {
		name string
		body string
	}{
		{"invalid JSON", `{not json`},
		{"missing workspace", `{"input":{"parameters":{"repo":"platform","targetBranch":"dev"}}}`},
		{"missing repo", `{"input":{"parameters":{"workspace":"firmapro","targetBranch":"dev"}}}`},
		{"missing targetBranch", `{"input":{"parameters":{"workspace":"firmapro","repo":"platform"}}}`},
	}
	for _, tt := range badInputs {
		t.Run(tt.name+" returns 400 without calling the provider", func(t *testing.T) {
			lister := &stubLister{}

			rec := callGetParams(lister, tt.body)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", rec.Code)
			}
			if lister.calls != 0 {
				t.Errorf("provider calls = %d, want 0", lister.calls)
			}
		})
	}
}
