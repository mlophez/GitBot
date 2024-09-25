package argocd

import (
	"context"
	"encoding/json"
	"gitbot/internal/app"
	"log/slog"
	"os"
	"strconv"
	"strings"

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

func GetApplication(ctx context.Context, cs *kubernetes.Clientset, appName string) (app.Application, error) {
	d, err := cs.RESTClient().Get().AbsPath("/apis/argoproj.io/v1alpha1").Namespace(NAMESPACE).Resource("applications").Name(appName).DoRaw(ctx)
	if err != nil {
		slog.Error("Error at make rest client request", "module", "argocd", "function", "Get", "error", err)
		return app.Application{}, err
	}

	a := ArgoAppResponse{}
	if err := json.Unmarshal(d, &a); err != nil {
		slog.Error("Error at parse kubernetes client response", "module", "argocd", "function", "Get", "error", err)
		return app.Application{}, err
	}

	return a.ConvertToApplication(), nil
}

func (resp ArgoAppResponse) ConvertToApplication() app.Application {
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

	return app.Application{
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

func (req ArgoCDAppRequest) ConvertToRequest(app app.Application) ArgoCDAppRequest {
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
