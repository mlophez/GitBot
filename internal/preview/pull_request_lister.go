package preview

import "context"

// PullRequestLister lists the open pull requests of a repository.
// It abstracts the Git provider for the preview generator.
// Implemented by adapters in the event slice (e.g. BitbucketClient) and wired in cmd.
type PullRequestLister interface {
	// ListOpenPullRequests returns every open pull request of workspace/repo,
	// across all result pages, each one validated at its creation point.
	// It must return either the complete list or an error, never a partial list:
	// the caller cannot tell a truncated list from a real one, and ArgoCD prunes
	// the previews of any pull request missing from the result.
	ListOpenPullRequests(ctx context.Context, workspace, repo string) ([]PullRequest, error)
}
