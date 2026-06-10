package event

import "fmt"

// PullRequest contains the state of a pull request at the time an event was received.
type PullRequest struct {
	Id                int
	SourceBranch      string
	DestinationBranch string
	Reviewers         int
	Approved          int
	RequestChanged    int
	CommitsBehind     int
	FilesChanged      []string
}

// Validate reports whether the pull request carries the minimal information the
// bot needs: a positive identifier. It is called when validating a PR-bearing
// Event (see Event.Validate).
func (pr PullRequest) Validate() error {
	if pr.Id <= 0 {
		return fmt.Errorf("pull request id must be positive, got %d", pr.Id)
	}
	return nil
}
