package app

import (
	"fmt"
	"log/slog"

)

// MultiClusterAppManager implements AppManager by aggregating multiple
// backend managers — one per cluster. List merges results from all clusters.
// Lock and Unlock route to the correct backend based on the application's Cluster field.
type MultiClusterAppManager struct {
	managers map[string]AppManager
}

// NewMultiClusterAppManager creates an empty MultiClusterAppManager.
// Use Add to register backends before calling List, Lock, or Unlock.
func NewMultiClusterAppManager() *MultiClusterAppManager {
	return &MultiClusterAppManager{managers: make(map[string]AppManager)}
}

// Add registers a backend AppManager for the given cluster name.
func (m *MultiClusterAppManager) Add(clusterName string, manager AppManager) {
	m.managers[clusterName] = manager
}

// List returns the merged list of applications from all registered clusters.
// If a backend fails, its error is logged and the remaining clusters are still returned.
// Returns an error only when every backend fails.
func (m *MultiClusterAppManager) List() ([]Application, error) {
	var all []Application
	failures := 0

	for cluster, mgr := range m.managers {
		apps, err := mgr.List()
		if err != nil {
			slog.Error("MultiClusterAppManager: failed to list apps from cluster", "cluster", cluster, "error", err)
			failures++
			continue
		}
		all = append(all, apps...)
	}

	if failures == len(m.managers) {
		return nil, fmt.Errorf("all %d cluster backends failed", failures)
	}
	return all, nil
}

// Lock routes the lock operation to the backend that owns app.Cluster,
// forwarding the force flag so remote backends can skip the already-locked check.
func (m *MultiClusterAppManager) Lock(app Application, targetBranch string, prID int, force bool) error {
	mgr, ok := m.managers[app.Cluster]
	if !ok {
		return fmt.Errorf("unknown cluster %q for app %q", app.Cluster, app.Name)
	}
	return mgr.Lock(app, targetBranch, prID, force)
}

// Unlock routes the unlock operation to the backend that owns app.Cluster.
func (m *MultiClusterAppManager) Unlock(app Application) error {
	mgr, ok := m.managers[app.Cluster]
	if !ok {
		return fmt.Errorf("unknown cluster %q for app %q", app.Cluster, app.Name)
	}
	return mgr.Unlock(app)
}
