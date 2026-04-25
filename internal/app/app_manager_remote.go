package app

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

)

// RemoteAppManager implements AppManager by calling a remote GitBot
// agent instance's REST API over HTTP. Used by the central instance to
// manage ArgoCD applications on remote clusters.
type RemoteAppManager struct {
	baseURL     string
	clusterName string
	apiToken    string
	client      *http.Client
}

// NewRemoteAppManager creates a RemoteAppManager that talks to the agent at baseURL.
// clusterName is stamped on every Application returned by List.
// apiToken is sent as a Bearer token on every outgoing request; pass an empty string to disable.
// When insecureSkipTLSVerify is true the HTTP client skips certificate validation —
// use only in non-production environments.
func NewRemoteAppManager(baseURL, clusterName string, insecureSkipTLSVerify bool, apiToken string) AppManager {
	transport := http.DefaultTransport
	if insecureSkipTLSVerify {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // intentional, user-configured
		}
	}
	return &RemoteAppManager{
		baseURL:     baseURL,
		clusterName: clusterName,
		apiToken:    apiToken,
		client:      &http.Client{Timeout: 10 * time.Second, Transport: transport},
	}
}

// newRequest builds an HTTP request with the Content-Type and Authorization headers
// pre-set when an API token is configured.
func (r *RemoteAppManager) newRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if r.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+r.apiToken)
	}
	return req, nil
}

// remoteAppResponse mirrors the JSON shape returned by the agent's GET /api/v1/apps.
type remoteAppResponse struct {
	Name          string   `json:"name"`
	Cluster       string   `json:"cluster,omitempty"`
	Repository    string   `json:"repository"`
	Branch        string   `json:"branch"`
	Paths         []string `json:"paths,omitempty"`
	Locked        bool     `json:"locked"`
	PullRequestId int      `json:"pull_request_id"`
	Environment   string   `json:"environment"`
}

// remoteLockRequest is the JSON body sent to the agent's POST /api/v1/apps/{id}/lock.
type remoteLockRequest struct {
	Branch        string `json:"branch"`
	PullRequestId int    `json:"pull_request_id"`
}

// List fetches all applications from the remote agent and returns them
// with the Cluster field set to the configured cluster name.
func (r *RemoteAppManager) List() ([]Application, error) {
	req, err := r.newRequest(http.MethodGet, r.baseURL+"/api/v1/apps", nil)
	if err != nil {
		return nil, fmt.Errorf("remote agent %q: failed to build request: %w", r.clusterName, err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("remote agent %q unreachable: %w", r.clusterName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("remote agent %q returned %d: %s", r.clusterName, resp.StatusCode, string(body))
	}

	var items []remoteAppResponse
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("remote agent %q: failed to decode response: %w", r.clusterName, err)
	}

	apps := make([]Application, 0, len(items))
	for _, item := range items {
		apps = append(apps, Application{
			Name:          item.Name,
			Cluster:       r.clusterName,
			Repository:    item.Repository,
			Branch:        item.Branch,
			Paths:         item.Paths,
			Locked:        item.Locked,
			PullRequestId: item.PullRequestId,
			Environment:   item.Environment,
		})
	}
	return apps, nil
}

// Lock tells the remote agent to lock the application to targetBranch for prID.
func (r *RemoteAppManager) Lock(app Application, targetBranch string, prID int) error {
	body, err := json.Marshal(remoteLockRequest{Branch: targetBranch, PullRequestId: prID})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/apps/%s/lock", r.baseURL, app.Name)
	req, err := r.newRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("remote agent %q: failed to build request: %w", r.clusterName, err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("remote agent %q unreachable: %w", r.clusterName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remote agent %q lock failed (%d): %s", r.clusterName, resp.StatusCode, string(msg))
	}
	return nil
}

// Unlock tells the remote agent to unlock the application.
func (r *RemoteAppManager) Unlock(app Application) error {
	url := fmt.Sprintf("%s/api/v1/apps/%s/unlock", r.baseURL, app.Name)
	req, err := r.newRequest(http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("remote agent %q: failed to build request: %w", r.clusterName, err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("remote agent %q unreachable: %w", r.clusterName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remote agent %q unlock failed (%d): %s", r.clusterName, resp.StatusCode, string(msg))
	}
	return nil
}
