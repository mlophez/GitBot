package app

// AppResponse is the HTTP response representation of an ArgoCD application.
// It exposes the fields relevant to API consumers and to remote agent
// communication (Paths is needed by the central instance for file-matching).
type AppResponse struct {
	Name          string   `json:"name"`
	Cluster       string   `json:"cluster,omitempty"`
	Repository    string   `json:"repository"`
	Branch        string   `json:"branch"`
	Paths         []string `json:"paths,omitempty"`
	Locked        bool     `json:"locked"`
	PullRequestId int      `json:"pull_request_id"`
	Environment   string   `json:"environment"`
}

// toAppResponse converts a domain Application into its HTTP response form.
func toAppResponse(app Application) AppResponse {
	return AppResponse{
		Name:          app.Name,
		Cluster:       app.Cluster,
		Repository:    app.Repository,
		Branch:        app.Branch,
		Paths:         app.Paths,
		Locked:        app.Locked,
		PullRequestId: app.PullRequestId,
		Environment:   app.Environment,
	}
}
