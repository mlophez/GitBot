package app

import (
	"fmt"
	"strings"
)

func cleanPaths(path string) string {
	path = strings.TrimPrefix(path, ".")
	if len(path) > 0 && path[0] != '/' {
		path = "/" + path
	}
	if len(path) > 0 && path[len(path)-1] != '/' {
		path += "/"
	}
	return path
}

func FindAppsByRepoAndFiles(repository string, files []string) ([]Application, error) {
	result := []Application{}

	argo := argocd{}

	apps, err := argo.FindAll()
	if err != nil {
		return nil, err
	}

	for _, app := range apps {
		match := false
		if app.Repository == repository {
			for _, file := range files {
				for _, path := range app.Paths {
					if strings.Contains(cleanPaths(file), cleanPaths(path)) {
						match = true
					}
				}
			}
		}
		if match {
			result = append(result, app)
		}
	}

	return result, nil
}

func LockApp(app Application, targetBranch string, prID int) (Application, error) {
	argo := argocd{}

	if app.Locked {
		return Application{}, fmt.Errorf("Application %s already locked", app.Name)
	}

	app.LastBranch = app.Branch
	app.Branch = targetBranch
	app.Locked = true
	app.PullRequestId = prID

	if _, err := argo.Update(app); err != nil {
		return Application{}, err
	}

	return app, nil
}

func UnlockApp(app Application) (Application, error) {
	if !app.Locked {
		return Application{}, fmt.Errorf("Application %s already unlocked", app.Name)
	}

	// Here mix logic with i/o. Should split in UnlockApp() and UpdateApp()
	argo := argocd{}

	app.Branch = app.LastBranch
	app.LastBranch = ""
	app.Locked = false
	app.PullRequestId = -1

	if _, err := argo.Update(app); err != nil {
		return Application{}, err
	}

	if _, err := argo.Clean(app); err != nil {
		return Application{}, err
	}

	return app, nil
}
