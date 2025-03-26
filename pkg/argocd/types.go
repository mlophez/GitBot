package argocd

type ArgoCDApp struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		// Annotations map[string]string `json:"annotations"`
		Annotations struct {
			Locked        string `json:"bot.gitbot.io/locked"`
			PullRequestId string `json:"bot.gitbot.io/pull-request"`
			// ProviderId    string `json:"bot.gitbot.io/provider"`
			Rollback         string `json:"bot.gitbot.io/rollback"`
			BasePath         string `json:"argocd.argoproj.io/manifest-generate-paths"`
			Environment      string `json:"gitbot.io/env"`
			ContainOtherApps string `json:"gitbot.io/contain-other-apps"`
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

type ArgoCDAppList struct {
	Items []ArgoCDApp `json:"items"`
}
