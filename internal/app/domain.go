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
	Environment   string
	ContainOther  bool
}

func (app Application) Sanitize() Application {
	if app.Locked && app.LastBranch == app.Branch {
		app.Locked = false
	}
	return app
}
