package adapters

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

type ArgoAppManager struct {
	clientset *kubernetes.Clientset
}

func NewArgoAppManager(cs *kubernetes.Clientset) types.AppManager {
	return &ArgoAppManager{clientset: cs}
}

func (a *ArgoAppManager) List() ([]types.Application, error) {
	data, err := a.clientset.RESTClient().Get().
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

func (a *ArgoAppManager) Lock(app types.Application, targetBranch string, prID int) error {
	locked := app.Lock(targetBranch, prID)
	return a.update(locked)
}

func (a *ArgoAppManager) Unlock(app types.Application) error {
	unlocked := app.Unlock()
	if err := a.update(unlocked); err != nil {
		return err
	}
	return a.clean(app.Name)
}

func (a *ArgoAppManager) update(app types.Application) error {
	body, err := json.Marshal(toRequest(app))
	if err != nil {
		return err
	}
	_, err = a.clientset.RESTClient().Patch(ktypes.MergePatchType).
		SetHeader("User-Agent", fieldManager).
		Body(body).
		AbsPath("/apis/argoproj.io/v1alpha1").
		Namespace(argoNamespace).
		Resource("applications").
		Name(app.Name).
		DoRaw(context.Background())
	return err
}

func (a *ArgoAppManager) clean(name string) error {
	jsonPatch := []byte(`[
		{ "op": "remove", "path": "/metadata/annotations/bot.gitbot.io~1locked" },
		{ "op": "remove", "path": "/metadata/annotations/bot.gitbot.io~1pull-request" },
		{ "op": "remove", "path": "/metadata/annotations/bot.gitbot.io~1rollback" }
	]`)
	_, err := a.clientset.RESTClient().Patch(ktypes.JSONPatchType).
		SetHeader("User-Agent", fieldManager).
		Body(jsonPatch).
		AbsPath("/apis/argoproj.io/v1alpha1").
		Namespace(argoNamespace).
		Resource("applications").
		Name(name).
		DoRaw(context.Background())
	return err
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

func NewListApps(cs *kubernetes.Clientset) func() ([]types.Application, error) {
	m := &ArgoAppManager{clientset: cs}
	return m.List
}

func GetApp(cs *kubernetes.Clientset) func(name string) (types.Application, error) {
	m := &ArgoAppManager{clientset: cs}
	return func(name string) (types.Application, error) {
		apps, err := m.List()
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
