package adapters

import (
	"fmt"
	"log/slog"

	"gitbot/internal/types"
)

// MultiClusterAppManager implements types.AppManager by aggregating multiple
// backend managers — one per cluster. List merges results from all clusters.
// Lock and Unlock route to the correct backend based on the application's Cluster field.
type MultiClusterAppManager struct {
	managers map[string]types.AppManager
}

// NewMultiClusterAppManager creates an empty MultiClusterAppManager.
// Use Add to register backends before calling List, Lock, or Unlock.
func NewMultiClusterAppManager() *MultiClusterAppManager {
	return &MultiClusterAppManager{managers: make(map[string]types.AppManager)}
}

// Add registers a backend AppManager for the given cluster name.
func (m *MultiClusterAppManager) Add(clusterName string, manager types.AppManager) {
	m.managers[clusterName] = manager
}

// List returns the merged list of applications from all registered clusters.
// If a backend fails, its error is logged and the remaining clusters are still returned.
// Returns an error only when every backend fails.
func (m *MultiClusterAppManager) List() ([]types.Application, error) {
	var all []types.Application
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

// Lock routes the lock operation to the backend that owns app.Cluster.
func (m *MultiClusterAppManager) Lock(app types.Application, targetBranch string, prID int) error {
	mgr, ok := m.managers[app.Cluster]
	if !ok {
		return fmt.Errorf("unknown cluster %q for app %q", app.Cluster, app.Name)
	}
	return mgr.Lock(app, targetBranch, prID)
}

// Unlock routes the unlock operation to the backend that owns app.Cluster.
func (m *MultiClusterAppManager) Unlock(app types.Application) error {
	mgr, ok := m.managers[app.Cluster]
	if !ok {
		return fmt.Errorf("unknown cluster %q for app %q", app.Cluster, app.Name)
	}
	return mgr.Unlock(app)
}
