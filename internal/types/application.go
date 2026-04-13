// Package types contains the pure domain types and interfaces for kubeops-agent.
// It defines the core data structures (Application, Event, PullRequest) and the
// contracts (AppManager, Provider, ConfigLoader, Queue) implemented by adapters.
// This package has no external dependencies and no I/O — it is the pure core of the system.
package types

// Application represents an ArgoCD application tracked by the bot.
// It is a pure value type — all state transitions are done via methods
// that return a new copy, with no mutation of the receiver.
type Application struct {
	Name          string   // ArgoCD application name
	Cluster       string   // Cluster this application belongs to (e.g. "tools", "demo")
	Repository    string   // Git repository URL (e.g. "https://bitbucket.org/org/repo.git")
	Branch        string   // Current targetRevision set in the ArgoCD spec
	Paths         []string // File paths this app is responsible for (used for PR matching)
	Locked        bool     // Whether the app is currently locked by a PR
	PullRequestId int      // ID of the PR holding the lock (-1 when unlocked)
	ProviderId    int      // Internal provider identifier
	LastBranch    string   // Branch before the lock was applied (restored on unlock)
	Environment   string   // Deployment environment (e.g. "dev", "prod")
	ContainOther  bool     // True when this is an app-of-apps that manages other apps
}

// Sanitize corrects inconsistent state where the app is marked as locked
// but the branch has not actually changed. Returns a corrected copy.
func (app Application) Sanitize() Application {
	if app.Locked && app.LastBranch == app.Branch {
		app.Locked = false
	}
	return app
}

// Lock returns a copy of the application pointed at targetBranch and marked
// as locked by prID. The current branch is saved in LastBranch for rollback.
// If the app is already locked, it is returned unchanged.
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

// Unlock returns a copy of the application restored to its previous branch
// and marked as unlocked. If the app is not locked, it is returned unchanged.
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
