package event

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
