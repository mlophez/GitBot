// Package app is the vertical slice for ArgoCD application management.
// It contains the pure domain (Application, AppManager), the adapters that
// implement the manager against Kubernetes/ArgoCD (local, multi-cluster, remote),
// and the use cases exposed via HTTP (list, lock, unlock, validate).
package app

import "fmt"

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

	// Status is the aggregated, display-ready synchronisation state observed on the
	// backend (e.g. "Healthy", "Degraded", "OutOfSync", "Progressing", "Missing",
	// "Suspended", "Unknown"). It is a read-only observed fact recomputed on every
	// List; it is not part of the lock/unlock state machine.
	Status string
	// StatusMessage is the message of the last sync/health error, or empty when the
	// app is healthy. Like Status, it is a read-only observed fact.
	StatusMessage string
}

// Validate reports whether the application satisfies the minimal domain
// invariants required to be usable by the bot: it must have a name and a
// repository. It is meant to be called right after an Application is hydrated
// from an external source (see the List adapters). Lock-state coherence is not
// enforced here on purpose, so apps read from ArgoCD with legitimate gaps
// (e.g. PullRequestId left at 0 when unlocked) are not rejected.
func (app Application) Validate() error {
	if app.Name == "" {
		return fmt.Errorf("application name is required")
	}
	if app.Repository == "" {
		return fmt.Errorf("application repository is required")
	}
	return nil
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
