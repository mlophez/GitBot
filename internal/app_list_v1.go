package internal

import (
	"encoding/json"
	"net/http"

	"gitbot/internal/types"
)

type AppResponse struct {
	Name          string `json:"name"`
	Repository    string `json:"repository"`
	Branch        string `json:"branch"`
	Locked        bool   `json:"locked"`
	PullRequestId int    `json:"pull_request_id"`
	Environment   string `json:"environment"`
}

func toAppResponse(app types.Application) AppResponse {
	return AppResponse{
		Name:          app.Name,
		Repository:    app.Repository,
		Branch:        app.Branch,
		Locked:        app.Locked,
		PullRequestId: app.PullRequestId,
		Environment:   app.Environment,
	}
}

func ListApps(getApps func() ([]types.Application, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apps, err := getApps()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := make([]AppResponse, len(apps))
		for i, a := range apps {
			resp[i] = toAppResponse(a)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
