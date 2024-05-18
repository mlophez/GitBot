package app

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/types"
)

const (
	namespace    = "argocd"
	kubeconfig   = "/home/mlr/Documents/Code/argocd-bot/kubeconfig"
	fieldManager = "gitbot"
)

type argocd struct{}

func (a argocd) FindAll() ([]Application, error) {
	type Response struct {
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

	type ResponseList struct {
		Items []Response `json:"items"`
	}

	clientset := getClientSet(kubeconfig)

	d, err := clientset.RESTClient().Get().AbsPath("/apis/argoproj.io/v1alpha1/applications").DoRaw(context.TODO())
	if err != nil {
		return nil, err
	}

	appList := ResponseList{}
	if err := json.Unmarshal(d, &appList); err != nil {
		return nil, err
	}

	var apps []Application
	for _, argoApp := range appList.Items {
		var paths []string
		paths = append(paths, argoApp.Spec.Source.Path)
		paths = append(paths, argoApp.Metadata.Annotations.BasePath+"/base")
		paths = append(paths, argoApp.Metadata.Annotations.BasePath+"/components")
		prId, _ := strconv.Atoi(argoApp.Metadata.Annotations.PullRequestId)
		apps = append(apps, Application{
			Name:          argoApp.Metadata.Name,
			Repository:    argoApp.Spec.Source.RepoUrl,
			Branch:        argoApp.Spec.Source.TargetRevision,
			Paths:         paths,
			Locked:        strings.ToLower(argoApp.Metadata.Annotations.Locked) == "true",
			PullRequestId: prId,
			LastBranch:    argoApp.Metadata.Annotations.Rollback,
		})
	}

	return apps, nil
}

func (a argocd) Update(app Application) (Application, error) {
	var result Application

	argoApp := ArgoCDApplicationRequest{}
	argoApp.FromApplication(app)

	body, err := json.Marshal(argoApp)
	if err != nil {
		return result, err
	}

	ctx := context.TODO()
	//ctx = context.WithValue(ctx, "managerName", "argocd-bot")
	//slog.Info(string(body))

	clientset := getClientSet(kubeconfig)
	req, err := clientset.RESTClient().Patch(types.MergePatchType).
		SetHeader("User-Agent", fieldManager).
		Body(body).
		AbsPath("/apis/argoproj.io/v1alpha1").
		Namespace(namespace).
		Resource("applications").
		Name(app.Name).
		DoRaw(ctx)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(req, &result); err != nil {
		return result, err
	}

	return result, nil
}

func (a argocd) Clean(app Application) (Application, error) {
	var result Application

	ctx := context.TODO()
	jsonPatch := []byte(`[
		{ "op": "remove", "path": "/metadata/annotations/bot.gitbot.io~1locked" },
		{ "op": "remove", "path": "/metadata/annotations/bot.gitbot.io~1pull-request" },
		{ "op": "remove", "path": "/metadata/annotations/bot.gitbot.io~1rollback" }
	]`)

	clientset := getClientSet(kubeconfig)
	req, err := clientset.RESTClient().Patch(types.JSONPatchType).
		SetHeader("User-Agent", fieldManager).
		Body(jsonPatch).
		AbsPath("/apis/argoproj.io/v1alpha1").
		Namespace(namespace).
		Resource("applications").
		Name(app.Name).
		DoRaw(ctx)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(req, &result); err != nil {
		return result, err
	}

	return result, nil

}

type ArgoCDApplicationRequest struct {
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

func (self *ArgoCDApplicationRequest) FromApplication(app Application) ArgoCDApplicationRequest {
	self.Metadata.Annotations.Rollback = app.LastBranch
	self.Metadata.Annotations.PullRequestId = strconv.Itoa(app.PullRequestId)
	if app.Locked {
		self.Metadata.Annotations.Locked = "true"
	} else {
		self.Metadata.Annotations.Locked = "false"
	}
	self.Spec.Source.TargetRevision = app.Branch
	return *self
}
