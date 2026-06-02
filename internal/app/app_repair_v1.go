package app

import "log/slog"

// PullRequestState is the observed state of the pull request that holds a lock.
// It mirrors the states reported by the Git provider (1:1) so that adapters only
// translate, while the policy that decides which states release a lock lives here.
type PullRequestState int

const (
	// PullRequestStateUnknown is returned when the provider state cannot be mapped
	// to a known value. Locks are never released on this state (fail-safe).
	PullRequestStateUnknown PullRequestState = iota
	// PullRequestStateOpen means the PR is still open; its lock must be kept.
	PullRequestStateOpen
	// PullRequestStateMerged means the PR was merged; its lock should be released.
	PullRequestStateMerged
	// PullRequestStateDeclined means the PR was rejected; its lock should be released.
	PullRequestStateDeclined
	// PullRequestStateSuperseded means the PR was superseded; its lock should be released.
	PullRequestStateSuperseded
)

// String returns the human-readable name of the state, used for logging.
func (s PullRequestState) String() string {
	switch s {
	case PullRequestStateOpen:
		return "open"
	case PullRequestStateMerged:
		return "merged"
	case PullRequestStateDeclined:
		return "declined"
	case PullRequestStateSuperseded:
		return "superseded"
	default:
		return "unknown"
	}
}

// LogValue makes slog render the state as its text name instead of the numeric
// enum value, including in the JSON handler.
func (s PullRequestState) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

// PullRequestStateChecker reports the current state of a pull request.
// It abstracts the Git provider for reconciliation purposes.
// Implemented by adapters in the event slice (e.g. BitbucketClient) and wired in cmd.
type PullRequestStateChecker interface {
	// GetPullRequestState returns the current state of the pull request prId in repo.
	GetPullRequestState(repo string, prId int) (PullRequestState, error)
}

// RepairResult summarises a single reconciliation run.
type RepairResult struct {
	Checked  int // locked apps inspected
	Unlocked int // apps released because their PR was closed
	Skipped  int // left as-is (PR still open, unknown, or API error)
	Failed   int // unlock attempts that errored
}

// RepairLocks inspects every locked application, queries the real state of the PR that
// holds the lock via checker, and unlocks apps whose PR is no longer open (merged,
// declined or superseded). This corrects orphaned locks left behind when the provider
// webhook that would normally trigger the unlock was missed.
//
// Errors querying a single PR are logged and skipped — an app is only unlocked on a
// confirmed closed state, never on an API error or an open/unknown state (fail-safe).
// Returns an error only when the application list cannot be fetched at all.
func RepairLocks(manager AppManager, checker PullRequestStateChecker) (RepairResult, error) {
	var res RepairResult

	apps, err := manager.List()
	if err != nil {
		return res, err
	}

	for _, a := range apps {
		a = a.Sanitize()
		if !a.Locked || a.PullRequestId <= 0 {
			continue
		}
		res.Checked++

		slog.Info("RepairLocks: checking locked app",
			"app", a.Name, "cluster", a.Cluster, "pullRequest", a.PullRequestId, "branch", a.Branch, "repository", a.Repository)

		state, err := checker.GetPullRequestState(a.Repository, a.PullRequestId)
		if err != nil {
			slog.Warn("RepairLocks: failed to query pull request state, keeping lock",
				"app", a.Name, "cluster", a.Cluster, "pullRequest", a.PullRequestId, "error", err)
			res.Skipped++
			continue
		}

		if !shouldUnlock(state) {
			slog.Info("RepairLocks: pull request still open, keeping lock",
				"app", a.Name, "cluster", a.Cluster, "pullRequest", a.PullRequestId, "state", state)
			res.Skipped++
			continue
		}

		slog.Info("RepairLocks: pull request closed, unlocking app",
			"app", a.Name, "cluster", a.Cluster, "pullRequest", a.PullRequestId, "state", state)
		if err := manager.Unlock(a); err != nil {
			slog.Error("RepairLocks: failed to unlock app",
				"app", a.Name, "cluster", a.Cluster, "pullRequest", a.PullRequestId, "error", err)
			res.Failed++
			continue
		}
		res.Unlocked++
	}

	return res, nil
}

// shouldUnlock reports whether a PR in the given state should release its lock.
// Open and unknown states keep the lock; merged, declined and superseded release it.
func shouldUnlock(state PullRequestState) bool {
	return state == PullRequestStateMerged ||
		state == PullRequestStateDeclined ||
		state == PullRequestStateSuperseded
}
