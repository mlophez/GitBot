package app

import (
	. "gitbot/types"
	"testing"
)

func TestCleanPaths(t *testing.T) {
	paths := [][]string{
		{"apps/kustomization.yaml", "/apps/kustomization.yaml/"},
		{"./apps/kustomization.yaml", "/apps/kustomization.yaml/"},
		{".apps/kustomization/", "/.apps/kustomization/"},
	}

	for _, path := range paths {
		cleaned := cleanPaths(path[0])
		if cleaned != path[1] {
			t.Fatalf("Expresion not match: %s != %s", cleaned, path[1])
		}
	}
}

func TestFilterAppByRepoAndFiles(t *testing.T) {
	var files []string
	var repo string
	app := Application{
		Name:       "application01",
		Repository: "https://bitbucket.org/gitbot/monorepo.git",
		Branch:     "main",
		Paths: []string{
			"application01/base",
			"application01/components",
			"application01/overlays/dev",
		},
		Locked:        false,
		PullRequestId: 0,
		LastBranch:    "",
	}

	// Check if with repo not match
	files = []string{}
	repo = "https://bitbucket.org/gitbot/monorep.git"
	if matchAppByRepoAndFiles(app, repo, files) {
		t.Fatalf("Repositories should no match")
	}

	files = []string{"/application01/overlays/dev/config.cfg"}
	repo = "https://bitbucket.org/gitbot/monorep.git"
	if matchAppByRepoAndFiles(app, repo, files) {
		t.Fatalf("Repository should no match")
	}

	files = []string{}
	repo = "https://bitbucket.org/gitbot/monorepo.git"
	if matchAppByRepoAndFiles(app, repo, files) {
		t.Fatalf("Files should no match")
	}

	files = []string{"/notmatching"}
	repo = "https://bitbucket.org/gitbot/monorepo.git"
	if matchAppByRepoAndFiles(app, repo, files) {
		t.Fatalf("Files should no match")
	}

	files = []string{"/application01/overlays/dev/config.cfg"}
	repo = "https://bitbucket.org/gitbot/monorepo.git"
	if !matchAppByRepoAndFiles(app, repo, files) {
		t.Fatalf("Files have match")
	}

	repo = "https://bitbucket.org/gitbot/monorepo.git"
	files = []string{"/holamundo", "/application01/base/config.cfg"}
	if !matchAppByRepoAndFiles(app, repo, files) {
		t.Fatalf("Files have match")
	}
}
