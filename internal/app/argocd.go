package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	NAMESPACE     = "argocd"
	FIELD_MANAGER = "gitbot"
)

type KubeRepository struct {
	clientset *kubernetes.Clientset
}

func (k KubeRepository) List(ctx context.Context) ([]Application, error) {
	d, err := k.clientset.RESTClient().Get().AbsPath("/apis/argoproj.io/v1alpha1/applications").DoRaw(ctx)
	if err != nil {
		slog.Error("Error at make rest client request", "module", "argocd", "function", "List", "error", err)
		return nil, err
	}

	appList := ArgoAppResponseList{}
	if err := json.Unmarshal(d, &appList); err != nil {
		slog.Error("Error at parse kubernetes client response", "module", "argocd", "function", "List", "error", err)
		return nil, err
	}

	var apps []Application
	for _, argoApp := range appList.Items {
		apps = append(apps, argoApp.ConvertToApplication())
	}

	return apps, nil
}

func (k KubeRepository) Update(ctx context.Context, app Application) (Application, error) {
	var result Application

	argoApp := ArgoCDAppRequest{}.ConvertToRequest(app)
	body, err := json.Marshal(argoApp)
	if err != nil {
		slog.Error("Error at parse kubernetes client request", "response", "module", "argocd", "function", "Update", "error", err)
		return result, err
	}

	req, err := k.clientset.RESTClient().Patch(types.MergePatchType).
		SetHeader("User-Agent", FIELD_MANAGER).
		Body(body).
		AbsPath("/apis/argoproj.io/v1alpha1").
		Namespace(NAMESPACE).
		Resource("applications").
		Name(app.Name).
		DoRaw(ctx)
	if err != nil {
		slog.Error("Error at make kubernetes client request", "response", "module", "argocd", "function", "Update", "error", err)
		return result, err
	}

	if err := json.Unmarshal(req, &result); err != nil {
		slog.Error("Error at parse kubernetes client response", "response", "module", "argocd", "function", "Update", "error", err)
		return result, err
	}

	return result, nil
}

func (k KubeRepository) Clean(ctx context.Context, app Application) (Application, error) {
	var result Application

	jsonPatch := []byte(`[
		{ "op": "remove", "path": "/metadata/annotations/bot.gitbot.io~1locked" },
		{ "op": "remove", "path": "/metadata/annotations/bot.gitbot.io~1pull-request" },
		{ "op": "remove", "path": "/metadata/annotations/bot.gitbot.io~1rollback" }
	]`)

	req, err := k.clientset.RESTClient().Patch(types.JSONPatchType).
		SetHeader("User-Agent", FIELD_MANAGER).
		Body(jsonPatch).
		AbsPath("/apis/argoproj.io/v1alpha1").
		Namespace(NAMESPACE).
		Resource("applications").
		Name(app.Name).
		DoRaw(ctx)
	if err != nil {
		slog.Error("Error at make kubernetes call", "response", "module", "argocd", "function", "Clean", "error", err)
		return result, err
	}

	if err := json.Unmarshal(req, &result); err != nil {
		slog.Error("Error at parse kubernetes response", "response", "module", "argocd", "function", "Clean", "error", err)
		return result, err
	}

	return result, nil
}

type ArgoAppResponse struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		// Annotations map[string]string `json:"annotations"`
		Annotations struct {
			Locked        string `json:"bot.gitbot.io/locked"`
			PullRequestId string `json:"bot.gitbot.io/pull-request"`
			// ProviderId    string `json:"bot.gitbot.io/provider"`
			Rollback    string `json:"bot.gitbot.io/rollback"`
			BasePath    string `json:"argocd.argoproj.io/manifest-generate-paths"`
			Environment string `json:"gitbot.io/env"`
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

type ArgoAppResponseList struct {
	Items []ArgoAppResponse `json:"items"`
}

type ArgoCDAppRequest struct {
	Metadata struct {
		Annotations struct {
			Locked        string `json:"bot.gitbot.io/locked"`
			PullRequestId string `json:"bot.gitbot.io/pull-request"`
			// ProviderId    string `json:"bot.gitbot.io/provider"`
			Rollback string `json:"bot.gitbot.io/rollback"`
		} `json:"annotations"`
	} `json:"metadata"`
	Spec struct {
		Source struct {
			TargetRevision string `json:"targetRevision"`
		} `json:"source"`
	} `json:"spec"`
}

func (resp ArgoAppResponse) ConvertToApplication() Application {
	var paths []string
	paths = append(paths, resp.Spec.Source.Path)
	paths = append(paths, resp.Metadata.Annotations.BasePath+"/base")
	paths = append(paths, resp.Metadata.Annotations.BasePath+"/components")

	prId, _ := strconv.Atoi(resp.Metadata.Annotations.PullRequestId)

	/* Parse environment of application */
	var env string
	if resp.Metadata.Annotations.Environment != "" {
		env = strings.ToLower(resp.Metadata.Annotations.Environment)
	} else {
		clusterName := os.Getenv("CLUSTER_NAME")
		if clusterName != "" {
			env = strings.ToLower(clusterName)
		} else {
			env = "unknown"
		}
	}
	/* ******************************* */

	return Application{
		Name:          resp.Metadata.Name,
		Repository:    resp.Spec.Source.RepoUrl,
		Branch:        resp.Spec.Source.TargetRevision,
		Paths:         paths,
		Locked:        strings.ToLower(resp.Metadata.Annotations.Locked) == "true",
		PullRequestId: prId,
		LastBranch:    resp.Metadata.Annotations.Rollback,
		Environment:   env,
	}
}

func (req ArgoCDAppRequest) ConvertToRequest(app Application) ArgoCDAppRequest {
	request := ArgoCDAppRequest{}
	request.Metadata.Annotations.Rollback = app.LastBranch
	request.Metadata.Annotations.PullRequestId = strconv.Itoa(app.PullRequestId)
	if app.Locked {
		request.Metadata.Annotations.Locked = "true"
	} else {
		request.Metadata.Annotations.Locked = "false"
	}
	request.Spec.Source.TargetRevision = app.Branch
	return request
}
