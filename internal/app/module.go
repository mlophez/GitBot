package app

import (
	"context"
	"fmt"
	"gitbot/internal/app/argocd"
	. "gitbot/types"
	"log/slog"

	"k8s.io/client-go/kubernetes"
)

func GetAllPulRequestApps(clientset *kubernetes.Clientset, e Event) ([]Application, error) {
	ctx := context.TODO()
	apps, err := argocd.List(clientset, ctx)
	if err != nil {
		slog.Error("Error at get all apps from kubernetes", "module", "app", "function", "GetAllPulRequestApps", "error", err)
		return nil, err
	}

	apps = filterAppByRepoAndFiles(apps, e.Repository, e.PullRequestFilesChanged)

	return apps, nil
}

func LockApp(clientset *kubernetes.Clientset, app Application, targetBranch string, prID int) error {
	if app.Locked {
		return fmt.Errorf("Application %s already locked", app.Name)
	}

	app = lockApp(app, targetBranch, prID)

	ctx := context.TODO()
	if _, err := argocd.Update(clientset, ctx, app); err != nil {
		slog.Error("Error at update app", "module", "app", "function", "LockApp", "error", err)
		return err
	}

	return nil
}

func UnlockApp(clientset *kubernetes.Clientset, app Application) error {
	if !app.Locked {
		return fmt.Errorf("Application %s already unlocked", app.Name)
	}

	app = unlockApp(app)

	ctx := context.TODO()
	if _, err := argocd.Update(clientset, ctx, app); err != nil {
		slog.Error("Error at update app", "module", "app", "function", "LockApp", "error", err)
		return err
	}

	if _, err := argocd.Clean(clientset, ctx, app); err != nil {
		slog.Error("Error at clean app", "module", "app", "function", "UnLockApp", "error", err)
		return err
	}

	return nil
}

func LockApps(clientset *kubernetes.Clientset, apps []Application, targetBranch string, prID int) error {
	for _, a := range apps {
		err := LockApp(clientset, a, targetBranch, prID)
		if err != nil {
			return err
		}
	}
	return nil
}

func UnLockApps(clientset *kubernetes.Clientset, apps []Application) error {
	for _, a := range apps {
		err := UnlockApp(clientset, a)
		if err != nil {
			return err
		}
	}
	return nil
}
