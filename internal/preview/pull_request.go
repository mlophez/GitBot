// Package preview serves the ArgoCD ApplicationSet plugin generator that drives
// per-pull-request preview environments. It lists the open pull requests of a
// repository through a provider adapter, keeps only those eligible for a preview
// (open, not draft, targeting the configured branch) and renders them with the
// same parameter names as ArgoCD's native pullRequest generator.
package preview

import (
	"fmt"

	"gitbot/internal/app"
)

// PullRequest is a pull request as seen by the preview generator: the identity
// and branches needed to build a preview environment, plus the state and draft
// flag used to decide whether it is eligible for one.
type PullRequest struct {
	Number       int
	Title        string
	Branch       string // source branch, deployed as the preview's target revision
	TargetBranch string // destination branch the pull request will be merged into
	HeadSHA      string // commit hash of the source branch head
	Author       string
	State        app.PullRequestState
	Draft        bool
}

// Validate reports whether the pull request carries the minimal information the
// generator needs: a positive number and both branch names. Without them no
// preview can be named or deployed.
func (pr PullRequest) Validate() error {
	if pr.Number <= 0 {
		return fmt.Errorf("pull request number must be positive, got %d", pr.Number)
	}
	if pr.Branch == "" {
		return fmt.Errorf("pull request %d has no source branch", pr.Number)
	}
	if pr.TargetBranch == "" {
		return fmt.Errorf("pull request %d has no target branch", pr.Number)
	}
	return nil
}
