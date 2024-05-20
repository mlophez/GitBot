package types

type Application struct {
	Name          string
	Repository    string
	Branch        string
	Paths         []string
	Locked        bool
	PullRequestId int
	LastBranch    string
}
