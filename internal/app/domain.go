package app

type Application struct {
	Name          string
	Repository    string
	Branch        string
	Paths         []string
	Locked        bool
	PullRequestId int
	ProviderId    int
	LastBranch    string
}
