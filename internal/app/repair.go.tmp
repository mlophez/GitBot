package app

import (
	"context"
	"gitbot/pkg/argocd"
	"log/slog"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func RepairApp() {
	// Get all apps
	rest, _ := rest.InClusterConfig()
	cs, err := kubernetes.NewForConfig(rest)

	acd := argocd.NewArgoCDClient(cs)
	apps, err := acd.List(context.TODO())
	if err != nil {
		return
	}

	for _, app := range apps {
		if !app.Metadata.Annotations.Locked {
			continue
		}
		slog.Info("", "name", app.Metadata.Name)
		//app.Metadata.Annotations.PullRequestId != null
		app.Spec.Source.TargetRevision
		// Get info of pull request
		// if open, check target branch go to pull request
		// if not pull request check target branch is in default branch
	}
}
