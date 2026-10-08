package preview

import (
	"strings"
	"testing"

	"gitbot/internal/app"
)

// openPR builds a valid, open, non-draft pull request targeting dev; tests
// derive the variants they need from it.
func openPR(number int) PullRequest {
	return PullRequest{
		Number:       number,
		Title:        "Add feature",
		Branch:       "feature/x",
		TargetBranch: "dev",
		HeadSHA:      "0123456789abcdef0123456789abcdef01234567",
		Author:       "jdoe",
		State:        app.PullRequestStateOpen,
	}
}

// TestSelectPreviewable checks the eligibility rule: only open, non-draft pull
// requests targeting the requested branch get a preview.
func TestSelectPreviewable(t *testing.T) {
	draft := openPR(2)
	draft.Draft = true
	otherBranch := openPR(3)
	otherBranch.TargetBranch = "main"
	merged := openPR(4)
	merged.State = app.PullRequestStateMerged
	declined := openPR(5)
	declined.State = app.PullRequestStateDeclined
	superseded := openPR(6)
	superseded.State = app.PullRequestStateSuperseded
	unknown := openPR(7)
	unknown.State = app.PullRequestStateUnknown

	tests := []struct {
		name string
		in   PullRequest
		want bool
	}{
		{"open, ready, targeting dev", openPR(1), true},
		{"draft", draft, false},
		{"targeting another branch", otherBranch, false},
		{"merged", merged, false},
		{"declined", declined, false},
		{"superseded", superseded, false},
		{"unknown state", unknown, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectPreviewable([]PullRequest{tt.in}, "dev")
			if (len(got) == 1) != tt.want {
				t.Errorf("selected = %v, want %v", len(got) == 1, tt.want)
			}
		})
	}

	t.Run("empty input yields empty, non-nil result", func(t *testing.T) {
		got := selectPreviewable(nil, "dev")
		if got == nil || len(got) != 0 {
			t.Errorf("selectPreviewable(nil) = %#v, want empty slice", got)
		}
	})
}

// TestToParams checks that a pull request renders with the same keys and formats
// as ArgoCD's native pullRequest generator, all values as strings.
func TestToParams(t *testing.T) {
	pr := openPR(42)
	pr.Branch = "Feature/My_Branch"

	got := toParams(pr)

	want := map[string]string{
		"number":             "42",
		"title":              "Add feature",
		"branch":             "Feature/My_Branch",
		"branch_slug":        "feature-my-branch",
		"target_branch":      "dev",
		"target_branch_slug": "dev",
		"head_sha":           "0123456789abcdef0123456789abcdef01234567",
		"head_short_sha":     "01234567",
		"head_short_sha_7":   "0123456",
		"author":             "jdoe",
	}
	if len(got) != len(want) {
		t.Errorf("toParams() has %d keys, want %d: %v", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("toParams()[%q] = %q, want %q", k, got[k], v)
		}
	}

	t.Run("branch_slug is capped at 50 characters", func(t *testing.T) {
		long := openPR(1)
		long.Branch = "feature/" + strings.Repeat("abcdefghij-", 10)
		if s := toParams(long)["branch_slug"]; len(s) > 50 {
			t.Errorf("branch_slug length = %d, want <= 50 (%q)", len(s), s)
		}
	})

	t.Run("short hashes do not overflow", func(t *testing.T) {
		short := openPR(1)
		short.HeadSHA = "abc12"
		p := toParams(short)
		if p["head_short_sha"] != "abc12" || p["head_short_sha_7"] != "abc12" {
			t.Errorf("short sha params = %q / %q, want abc12", p["head_short_sha"], p["head_short_sha_7"])
		}
	})
}
