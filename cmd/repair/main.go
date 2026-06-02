// Package main is the entrypoint for the kubeops-agent reconciliation job.
// It runs once and exits: it lists all ArgoCD applications, queries the real
// state of the pull request that holds each lock, and unlocks any app whose PR
// is no longer open (merged or declined). Intended to run as a Kubernetes
// CronJob so that orphaned locks — left behind when a provider webhook is missed
// — are eventually corrected.
package main

import (
	"log/slog"
	"os"

	"gitbot/internal/app"
	"gitbot/internal/config"
	"gitbot/internal/event"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	c := config.NewEnvConfigLoader().Load()

	localManager := app.NewArgoAppManager(c.ClientSet, c.ClusterName)
	appManager := buildAppManager(c, localManager)

	// BitbucketClient satisfies app.PullRequestStateChecker structurally.
	bitbucket := event.NewBitbucketClient(c.BitbucketBearerToken, c.BitbucketBotUUID)

	slog.Info("repair: starting lock reconciliation", "cluster", c.ClusterName)

	res, err := app.RepairLocks(appManager, bitbucket)
	if err != nil {
		slog.Error("repair: failed to reconcile locks", "error", err)
		os.Exit(1)
	}

	slog.Info("repair: reconciliation finished",
		"checked", res.Checked, "unlocked", res.Unlocked, "skipped", res.Skipped, "failed", res.Failed)

	if res.Failed > 0 {
		os.Exit(1)
	}
}

// buildAppManager returns a MultiClusterAppManager when the config defines remote
// agent clusters, or the local manager directly when there are none.
// Mirrors the wiring in cmd/server/main.go so the job reconciles across all clusters.
func buildAppManager(c *config.Config, local app.AppManager) app.AppManager {
	var remotes []config.ClusterConfig
	for _, cl := range c.Clusters {
		if cl.Auth.Type == "agent" {
			remotes = append(remotes, cl)
		}
	}
	if len(remotes) == 0 {
		return local
	}

	multi := app.NewMultiClusterAppManager()

	// Register the local cluster. Prefer the name from config; fall back to CLUSTER_NAME.
	localName := c.ClusterName
	for _, cl := range c.Clusters {
		if cl.Auth.Type == "serviceaccount" {
			localName = cl.Name
			break
		}
	}
	multi.Add(localName, local)

	// Register each remote agent cluster.
	for _, cl := range remotes {
		multi.Add(cl.Name, app.NewRemoteAppManager(cl.Auth.URL, cl.Name, cl.Auth.InsecureSkipTLSVerify, c.APIToken))
	}

	return multi
}
