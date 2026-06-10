package app

import "log/slog"

// AppManager abstracts read and write operations on ArgoCD applications.
// Implemented by ArgoAppManager (same package) for the Kubernetes/ArgoCD backend,
// and by RemoteAppManager for delegating to a remote GitBot agent via HTTP.
type AppManager interface {
	// List returns all applications currently tracked in the cluster.
	List() ([]Application, error)

	// Lock points the application at targetBranch and marks it as locked by prID.
	// When force is true, the operation proceeds even if the app is already locked,
	// overwriting the current lock state directly (branch-to-branch transition).
	Lock(app Application, targetBranch string, prID int, force bool) error

	// Unlock restores the application to the branch it held before the lock
	// and removes all lock-related annotations.
	Unlock(app Application) error
}

// keepValidApps returns only the applications that pass Validate, logging a
// warning for each one discarded. List implementations call it so that a single
// malformed application read from the backend cannot fail the whole listing.
func keepValidApps(apps []Application) []Application {
	valid := make([]Application, 0, len(apps))
	for _, a := range apps {
		if err := a.Validate(); err != nil {
			slog.Warn("discarding invalid application", "name", a.Name, "error", err)
			continue
		}
		valid = append(valid, a)
	}
	return valid
}
