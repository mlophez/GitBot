package argocd

import (
	. "gitbot/types"
	"strconv"
	"strings"
)

func ConvertToApplication(resp ArgoAppResponse) Application {
	var paths []string
	paths = append(paths, resp.Spec.Source.Path)
	paths = append(paths, resp.Metadata.Annotations.BasePath+"/base")
	paths = append(paths, resp.Metadata.Annotations.BasePath+"/components")

	prId, _ := strconv.Atoi(resp.Metadata.Annotations.PullRequestId)

	return Application{
		Name:          resp.Metadata.Name,
		Repository:    resp.Spec.Source.RepoUrl,
		Branch:        resp.Spec.Source.TargetRevision,
		Paths:         paths,
		Locked:        strings.ToLower(resp.Metadata.Annotations.Locked) == "true",
		PullRequestId: prId,
		LastBranch:    resp.Metadata.Annotations.Rollback,
	}
}

func ConvertToRequest(app Application) ArgoCDAppRequest {
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
