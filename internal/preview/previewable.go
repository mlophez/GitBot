package preview

import (
	"strconv"

	"github.com/gosimple/slug"

	"gitbot/internal/app"
)

// The slug package is configured through package-level variables. Set them once,
// exactly as ArgoCD's native pullRequest generator does, so branch_slug and
// target_branch_slug render identically and the ApplicationSet template does not
// change when switching generators: names capped at 50 characters (room for a
// suffix within the 63-character DNS label limit) and underscores turned into dashes.
func init() {
	slug.MaxLength = 50
	slug.CustomSub = map[string]string{"_": "-"}
}

// selectPreviewable returns the pull requests that must have a preview
// environment: open, not draft, and targeting targetBranch. Order is preserved.
func selectPreviewable(prs []PullRequest, targetBranch string) []PullRequest {
	selected := make([]PullRequest, 0, len(prs))
	for _, pr := range prs {
		if pr.State != app.PullRequestStateOpen || pr.Draft || pr.TargetBranch != targetBranch {
			continue
		}
		selected = append(selected, pr)
	}
	return selected
}

// toParams renders a pull request as an ApplicationSet parameter set, using the
// same keys and formats as ArgoCD's native pullRequest generator. Every value is a
// string, as the plugin generator contract expects.
func toParams(pr PullRequest) map[string]string {
	return map[string]string{
		"number":             strconv.Itoa(pr.Number),
		"title":              pr.Title,
		"branch":             pr.Branch,
		"branch_slug":        slug.Make(pr.Branch),
		"target_branch":      pr.TargetBranch,
		"target_branch_slug": slug.Make(pr.TargetBranch),
		"head_sha":           pr.HeadSHA,
		"head_short_sha":     shortSHA(pr.HeadSHA, 8),
		"head_short_sha_7":   shortSHA(pr.HeadSHA, 7),
		"author":             pr.Author,
	}
}

// shortSHA returns the first n characters of sha, or the whole sha when it is
// shorter (Bitbucket list responses may carry abbreviated hashes).
func shortSHA(sha string, n int) string {
	return sha[:min(len(sha), n)]
}
