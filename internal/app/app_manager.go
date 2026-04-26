package app

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
