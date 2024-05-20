package argocd

import (
	"context"
	"encoding/json"
	. "gitbot/types"
	"log/slog"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	NAMESPACE     = "argocd"
	FIELD_MANAGER = "gitbot"
)

type ArgoAppResponse struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		// Annotations map[string]string `json:"annotations"`
		Annotations struct {
			Locked        string `json:"bot.gitbot.io/locked"`
			PullRequestId string `json:"bot.gitbot.io/pull-request"`
			Rollback      string `json:"bot.gitbot.io/rollback"`
			BasePath      string `json:"argocd.argoproj.io/manifest-generate-paths"`
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
			Rollback      string `json:"bot.gitbot.io/rollback"`
		} `json:"annotations"`
	} `json:"metadata"`
	Spec struct {
		Source struct {
			TargetRevision string `json:"targetRevision"`
		} `json:"source"`
	} `json:"spec"`
}

func List(clientset *kubernetes.Clientset, ctx context.Context) ([]Application, error) {
	d, err := clientset.RESTClient().Get().AbsPath("/apis/argoproj.io/v1alpha1/applications").DoRaw(ctx)
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
		apps = append(apps, ConvertToApplication(argoApp))
	}

	return apps, nil
}

func Update(clientset *kubernetes.Clientset, ctx context.Context, app Application) (Application, error) {
	var result Application

	argoApp := ConvertToRequest(app)
	body, err := json.Marshal(argoApp)
	if err != nil {
		slog.Error("Error at parse kubernetes client request", "response", "module", "argocd", "function", "Update", "error", err)
		return result, err
	}

	req, err := clientset.RESTClient().Patch(types.MergePatchType).
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

func Clean(clientset *kubernetes.Clientset, ctx context.Context, app Application) (Application, error) {
	var result Application

	jsonPatch := []byte(`[
		{ "op": "remove", "path": "/metadata/annotations/bot.gitbot.io~1locked" },
		{ "op": "remove", "path": "/metadata/annotations/bot.gitbot.io~1pull-request" },
		{ "op": "remove", "path": "/metadata/annotations/bot.gitbot.io~1rollback" }
	]`)

	req, err := clientset.RESTClient().Patch(types.JSONPatchType).
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
