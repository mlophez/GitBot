package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"gitbot/internal/types"
)

const (
	argoNamespace = "argocd"
	fieldManager  = "gitbot"
)

func ListApps(cs *kubernetes.Clientset) func() ([]types.Application, error) {
	return func() ([]types.Application, error) {
		data, err := cs.RESTClient().Get().
			AbsPath("/apis/argoproj.io/v1alpha1/applications").
			DoRaw(context.Background())
		if err != nil {
			return nil, err
		}

		var list argoAppList
		if err := json.Unmarshal(data, &list); err != nil {
			return nil, err
		}

		apps := make([]types.Application, 0, len(list.Items))
		for _, item := range list.Items {
			apps = append(apps, toApplication(item))
		}
		return apps, nil
	}
}

func GetApp(cs *kubernetes.Clientset) func(name string) (types.Application, error) {
	listApps := ListApps(cs)
	return func(name string) (types.Application, error) {
		apps, err := listApps()
		if err != nil {
			return types.Application{}, err
		}
		for _, a := range apps {
			if a.Name == name {
				return a, nil
			}
		}
		return types.Application{}, fmt.Errorf("app %q not found", name)
	}
}

func UpdateApp(cs *kubernetes.Clientset) func(types.Application) error {
	return func(app types.Application) error {
		body, err := json.Marshal(toRequest(app))
		if err != nil {
			return err
		}
		_, err = cs.RESTClient().Patch(ktypes.MergePatchType).
			SetHeader("User-Agent", fieldManager).
			Body(body).
			AbsPath("/apis/argoproj.io/v1alpha1").
			Namespace(argoNamespace).
			Resource("applications").
			Name(app.Name).
			DoRaw(context.Background())
		return err
	}
}

func CleanApp(cs *kubernetes.Clientset) func(name string) error {
	return func(name string) error {
		jsonPatch := []byte(`[
			{ "op": "remove", "path": "/metadata/annotations/bot.gitbot.io~1locked" },
			{ "op": "remove", "path": "/metadata/annotations/bot.gitbot.io~1pull-request" },
			{ "op": "remove", "path": "/metadata/annotations/bot.gitbot.io~1rollback" }
		]`)
		_, err := cs.RESTClient().Patch(ktypes.JSONPatchType).
			SetHeader("User-Agent", fieldManager).
			Body(jsonPatch).
			AbsPath("/apis/argoproj.io/v1alpha1").
			Namespace(argoNamespace).
			Resource("applications").
			Name(name).
			DoRaw(context.Background())
		return err
	}
}

// ── Kubernetes JSON types ─────────────────────────────────────────────────

type argoApp struct {
	Metadata struct {
		Name        string `json:"name"`
		Annotations struct {
			Locked        string `json:"bot.gitbot.io/locked"`
			PullRequestId string `json:"bot.gitbot.io/pull-request"`
			Rollback      string `json:"bot.gitbot.io/rollback"`
			BasePath      string `json:"argocd.argoproj.io/manifest-generate-paths"`
			Environment   string `json:"gitbot.io/env"`
		} `json:"annotations"`
	} `json:"metadata"`
	Spec struct {
		Source struct {
			RepoUrl        string `json:"repoUrl"`
			TargetRevision string `json:"targetRevision"`
			Path           string `json:"path"`
		} `json:"source"`
	} `json:"spec"`
}

type argoAppList struct {
	Items []argoApp `json:"items"`
}

type argoAppPatch struct {
	Metadata struct {
		Annotations struct {
			Locked        string `json:"bot.gitbot.io/locked"`
			PullRequestId string `json:"bot.gitbot.io/pull-request"`
			Rollback      string `json:"bot.gitbot.io/rollback"`
		} `json:"annotations"`
	} `json:"metadata"`
	Spec struct {
		Source struct {
			TargetRevision string `json:"targetRevision"`
		} `json:"source"`
	} `json:"spec"`
}

func toApplication(a argoApp) types.Application {
	prID, _ := strconv.Atoi(a.Metadata.Annotations.PullRequestId)

	env := a.Metadata.Annotations.Environment
	if env == "" {
		if clusterName := os.Getenv("CLUSTER_NAME"); clusterName != "" {
			env = strings.ToLower(clusterName)
		} else {
			env = "unknown"
		}
	}

	return types.Application{
		Name:          a.Metadata.Name,
		Repository:    a.Spec.Source.RepoUrl,
		Branch:        a.Spec.Source.TargetRevision,
		Paths:         []string{a.Spec.Source.Path, a.Metadata.Annotations.BasePath + "/base", a.Metadata.Annotations.BasePath + "/components"},
		Locked:        strings.ToLower(a.Metadata.Annotations.Locked) == "true",
		PullRequestId: prID,
		LastBranch:    a.Metadata.Annotations.Rollback,
		Environment:   strings.ToLower(env),
	}
}

func toRequest(app types.Application) argoAppPatch {
	var p argoAppPatch
	p.Metadata.Annotations.Rollback = app.LastBranch
	p.Metadata.Annotations.PullRequestId = strconv.Itoa(app.PullRequestId)
	if app.Locked {
		p.Metadata.Annotations.Locked = "true"
	} else {
		p.Metadata.Annotations.Locked = "false"
	}
	p.Spec.Source.TargetRevision = app.Branch
	return p
}
