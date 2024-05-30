package app

import (
	"context"
	"fmt"
	"log/slog"

	"k8s.io/client-go/kubernetes"
)

type Service struct {
	repository Repository
}

type Repository interface {
	List(ctx context.Context) ([]Application, error)
	Update(ctx context.Context, app Application) (Application, error)
	Clean(ctx context.Context, app Application) (Application, error)
}

func NewService(clientset *kubernetes.Clientset) Service {
	return Service{
		repository: KubeRepository{clientset: clientset},
	}
}

func (s Service) FindAppsByRepoAndFiles(repo string, files []string) ([]Application, error) {
	ctx := context.TODO()

	apps, err := s.repository.List(ctx)
	if err != nil {
		slog.Error("Error at get all apps from kubernetes", "module", "app", "function", "FindAppsByRepoAndFiles", "error", err)
		return nil, err
	}

	apps = filterAppByRepoAndFiles(apps, repo, files)

	return apps, nil
}

func (s Service) LockApp(app Application, targetBranch string, prID int) error {
	if app.Locked {
		return fmt.Errorf("Application %s already locked", app.Name)
	}

	app = lockApp(app, targetBranch, prID)

	ctx := context.TODO()
	if _, err := s.repository.Update(ctx, app); err != nil {
		slog.Error("Error at update app", "module", "app", "function", "LockApp", "error", err)
		return err
	}

	return nil
}

func (s Service) UnlockApp(app Application) error {
	if !app.Locked {
		return fmt.Errorf("Application %s already unlocked", app.Name)
	}

	app = unlockApp(app)

	ctx := context.TODO()
	if _, err := s.repository.Update(ctx, app); err != nil {
		slog.Error("Error at update app", "module", "app", "function", "LockApp", "error", err)
		return err
	}

	if _, err := s.repository.Clean(ctx, app); err != nil {
		slog.Error("Error at clean app", "module", "app", "function", "UnLockApp", "error", err)
		return err
	}

	return nil
}

func (s Service) LockApps(apps []Application, targetBranch string, prID int) error {
	for _, a := range apps {
		err := s.LockApp(a, targetBranch, prID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s Service) UnLockApps(apps []Application) error {
	for _, a := range apps {
		err := s.UnlockApp(a)
		if err != nil {
			return err
		}
	}
	return nil
}
