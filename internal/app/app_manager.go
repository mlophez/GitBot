package app

// AppManager abstracts read and write operations on ArgoCD applications.
// Implemented by adapter.ArgoAppManager for the Kubernetes/ArgoCD backend.
type AppManager interface {
	// List returns all applications currently tracked in the cluster.
	List() ([]Application, error)

	// Lock points the application at targetBranch and marks it as locked by prID.
	// The application must not already be locked.
	Lock(app Application, targetBranch string, prID int) error

	// Unlock restores the application to the branch it held before the lock
	// and removes all lock-related annotations.
	Unlock(app Application) error
}
