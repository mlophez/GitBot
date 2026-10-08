package preview

import (
	"encoding/json"
	"net/http"

	"gitbot/internal/logger"
)

// maxRequestBytes caps the request body; the generator input is a handful of strings.
const maxRequestBytes = 1 << 20

// getParamsRequest is the body ArgoCD's plugin generator sends to
// POST /api/v1/getparams.execute. input.parameters is copied verbatim from the
// ApplicationSet generator block.
type getParamsRequest struct {
	ApplicationSetName string `json:"applicationSetName"`
	Input              struct {
		Parameters struct {
			Workspace    string `json:"workspace"`
			Repo         string `json:"repo"`
			TargetBranch string `json:"targetBranch"`
		} `json:"parameters"`
	} `json:"input"`
}

// getParamsResponse is the body ArgoCD expects back: one parameter map per
// Application to generate, nested under output.parameters.
type getParamsResponse struct {
	Output struct {
		Parameters []map[string]string `json:"parameters"`
	} `json:"output"`
}

// GetParams handles POST /api/v1/getparams.execute, the fixed path of ArgoCD's
// ApplicationSet plugin generator. It lists the open pull requests of the
// workspace/repo given in input.parameters and answers 200 with one parameter set
// per pull request that is open, not draft and targets input.parameters.targetBranch,
// using the parameter names of ArgoCD's native pullRequest generator.
//
// Returns 400 if the body cannot be decoded or workspace, repo or targetBranch is
// empty (the provider is not called). Returns 502 if the provider listing fails for
// any reason: it never answers with an empty or partial list on error, because the
// ApplicationSet prunes every preview missing from a successful response, while on
// an error ArgoCD keeps the current Applications and retries.
// Authentication is enforced by the router middleware, not here.
func GetParams(lister PullRequestLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.Logger(r.Context())

		var req getParamsRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBytes)).Decode(&req); err != nil {
			log.Error("GetParams: bad request", "error", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		in := req.Input.Parameters
		if in.Workspace == "" || in.Repo == "" || in.TargetBranch == "" {
			log.Error("GetParams: missing input parameters",
				"applicationSet", req.ApplicationSetName, "workspace", in.Workspace, "repo", in.Repo, "targetBranch", in.TargetBranch)
			http.Error(w, "Bad Request: workspace, repo and targetBranch are required", http.StatusBadRequest)
			return
		}

		prs, err := lister.ListOpenPullRequests(r.Context(), in.Workspace, in.Repo)
		if err != nil {
			// Fail closed: an error status makes ArgoCD keep the existing previews.
			log.Error("GetParams: failed to list pull requests",
				"applicationSet", req.ApplicationSetName, "workspace", in.Workspace, "repo", in.Repo, "error", err)
			http.Error(w, "Bad Gateway: failed to list pull requests", http.StatusBadGateway)
			return
		}

		selected := selectPreviewable(prs, in.TargetBranch)

		var resp getParamsResponse
		// Always a JSON array: a legitimate "no previews" result is [], never null.
		resp.Output.Parameters = make([]map[string]string, 0, len(selected))
		for _, pr := range selected {
			resp.Output.Parameters = append(resp.Output.Parameters, toParams(pr))
		}

		log.Info("GetParams: returning parameters",
			"applicationSet", req.ApplicationSetName, "workspace", in.Workspace, "repo", in.Repo,
			"targetBranch", in.TargetBranch, "open", len(prs), "selected", len(selected))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
