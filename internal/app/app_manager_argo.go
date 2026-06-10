package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	argoNamespace = "argocd"
	fieldManager  = "gitbot"
)

// ArgoAppManager implements AppManager for a single Kubernetes cluster
// running ArgoCD. It reads and writes ArgoCD Application resources via the
// Kubernetes REST API.
type ArgoAppManager struct {
	clientset   *kubernetes.Clientset
	clusterName string
}

// NewArgoAppManager creates an ArgoAppManager for the given Kubernetes clientset.
// clusterName is stamped on every Application returned by List so that callers
// can identify which cluster the app belongs to.
func NewArgoAppManager(cs *kubernetes.Clientset, clusterName string) AppManager {
	return &ArgoAppManager{clientset: cs, clusterName: clusterName}
}

// List returns all ArgoCD Application resources found in the argocd namespace,
// with the Cluster field set to the configured cluster name.
func (a *ArgoAppManager) List() ([]Application, error) {
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

	apps := make([]Application, 0, len(list.Items))
	for _, item := range list.Items {
		app := toApplication(item)
		app.Cluster = a.clusterName
		apps = append(apps, app)
	}
	// Validate at the creation point: drop apps that fail validation (and log them)
	// instead of failing the whole listing on a single malformed record.
	return keepValidApps(apps), nil
}

// Lock points the ArgoCD application at targetBranch and records prID as the lock holder.
// The force parameter is accepted for interface compliance but is not needed here: the
// underlying Kubernetes MergePatch always overwrites the current state regardless.
func (a *ArgoAppManager) Lock(app Application, targetBranch string, prID int, _ bool) error {
	locked := app.Lock(targetBranch, prID)
	return a.update(locked)
}

// Unlock restores the application to its pre-lock branch and removes the lock annotations.
func (a *ArgoAppManager) Unlock(app Application) error {
	unlocked := app.Unlock()
	if err := a.update(unlocked); err != nil {
		return err
	}
	return a.clean(app.Name)
}

func (a *ArgoAppManager) update(app Application) error {
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
	Status struct {
		Sync struct {
			Status string `json:"status"`
		} `json:"sync"`
		Health struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"health"`
		Conditions []struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"conditions"`
		OperationState struct {
			Phase   string `json:"phase"`
			Message string `json:"message"`
		} `json:"operationState"`
	} `json:"status"`
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

func toApplication(a argoApp) Application {
	prID, _ := strconv.Atoi(a.Metadata.Annotations.PullRequestId)

	env := a.Metadata.Annotations.Environment
	if env == "" {
		if clusterName := os.Getenv("CLUSTER_NAME"); clusterName != "" {
			env = strings.ToLower(clusterName)
		} else {
			env = "unknown"
		}
	}

	return Application{
		Name:          a.Metadata.Name,
		Repository:    a.Spec.Source.RepoUrl,
		Branch:        a.Spec.Source.TargetRevision,
		Paths:         []string{a.Spec.Source.Path, a.Metadata.Annotations.BasePath + "/base", a.Metadata.Annotations.BasePath + "/components"},
		Locked:        strings.ToLower(a.Metadata.Annotations.Locked) == "true",
		PullRequestId: prID,
		LastBranch:    a.Metadata.Annotations.Rollback,
		Environment:   strings.ToLower(env),
		Status:        deriveStatus(a),
		StatusMessage: deriveStatusMessage(a),
	}
}

// deriveStatus collapses the ArgoCD .status.sync and .status.health fields into a
// single display-ready value. An unhealthy health status takes precedence over the
// sync status, so a degraded app is never masked by being reported as OutOfSync.
// Falls back to "Unknown" when ArgoCD has not yet populated any status.
func deriveStatus(a argoApp) string {
	health := a.Status.Health.Status
	sync := a.Status.Sync.Status
	if health != "" && health != "Healthy" {
		return health
	}
	if sync != "" && sync != "Synced" {
		return sync
	}
	if health != "" {
		return health
	}
	if sync != "" {
		return sync
	}
	return "Unknown"
}

// deriveStatusMessage selects the most relevant error message from the ArgoCD status,
// preferring a failing condition, then a failed operation, then the health message.
// Returns an empty string when the app reports no problem.
func deriveStatusMessage(a argoApp) string {
	for _, c := range a.Status.Conditions {
		if strings.Contains(strings.ToLower(c.Type), "error") && c.Message != "" {
			return c.Message
		}
	}
	phase := a.Status.OperationState.Phase
	if (phase == "Failed" || phase == "Error") && a.Status.OperationState.Message != "" {
		return a.Status.OperationState.Message
	}
	if a.Status.Health.Status != "" && a.Status.Health.Status != "Healthy" {
		return a.Status.Health.Message
	}
	return ""
}

func toRequest(app Application) argoAppPatch {
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

// NewListApps returns a function that lists all ArgoCD applications in the cluster.
func NewListApps(cs *kubernetes.Clientset) func() ([]Application, error) {
	m := &ArgoAppManager{clientset: cs, clusterName: os.Getenv("CLUSTER_NAME")}
	return m.List
}

// GetApp returns a function that finds a single ArgoCD application by name.
func GetApp(cs *kubernetes.Clientset) func(name string) (Application, error) {
	m := &ArgoAppManager{clientset: cs, clusterName: os.Getenv("CLUSTER_NAME")}
	return func(name string) (Application, error) {
		apps, err := m.List()
		if err != nil {
			return Application{}, err
		}
		for _, a := range apps {
			if a.Name == name {
				return a, nil
			}
		}
		return Application{}, fmt.Errorf("app %q not found", name)
	}
}
