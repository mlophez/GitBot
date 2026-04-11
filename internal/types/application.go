package types

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

func (app Application) Lock(targetBranch string, prID int) Application {
	if app.Locked {
		return app
	}
	app.LastBranch = app.Branch
	app.Branch = targetBranch
	app.Locked = true
	app.PullRequestId = prID
	return app
}

func (app Application) Unlock() Application {
	if !app.Locked {
		return app
	}
	app.Branch = app.LastBranch
	app.LastBranch = ""
	app.Locked = false
	app.PullRequestId = -1
	return app
}
