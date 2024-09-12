package app

import (
	"strings"
)

func filterAppByRepoAndFiles(apps []Application, repo string, files []string) []Application {
	var result []Application

	for _, app := range apps {
		if matchAppByRepoAndFiles(app, repo, files) {
			result = append(result, app.Sanitize())
		}
	}

	return result
}

func matchAppByRepoAndFiles(app Application, repo string, files []string) bool {
	if app.Repository == repo {
		for _, file := range files {
			for _, path := range app.Paths { // if some file match with paths controlled by app return true
				if strings.Contains(cleanPaths(file), cleanPaths(path)) {
					return true
				}
			}
		}
	}
	return false
}

func cleanPaths(path string) string {
	if len(path) > 1 && path[0] == '.' && path[1] == '/' {
		path = strings.TrimPrefix(path, ".")
	}

	if len(path) > 0 && path[0] != '/' {
		path = "/" + path
	}

	if len(path) > 0 && path[len(path)-1] != '/' {
		path += "/"
	}

	return path
}

func lockApp(app Application, targetBranch string, prID int) Application {
	if app.Locked {
		return app
	}

	app.LastBranch = app.Branch
	app.Branch = targetBranch
	app.Locked = true
	app.PullRequestId = prID

	return app
}

func unlockApp(app Application) Application {
	if !app.Locked {
		return app
	}

	app.Branch = app.LastBranch
	app.LastBranch = ""
	app.Locked = false
	app.PullRequestId = -1

	return app
}
