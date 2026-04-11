package adapters

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gitbot/internal/types"
)

// RemoteAppManager implements types.AppManager by calling a remote GitBot
// agent instance's REST API over HTTP. Used by the central instance to
// manage ArgoCD applications on remote clusters.
type RemoteAppManager struct {
	baseURL     string
	clusterName string
	client      *http.Client
}

// NewRemoteAppManager creates a RemoteAppManager that talks to the agent at baseURL.
// clusterName is stamped on every Application returned by List.
// When insecureSkipTLSVerify is true the HTTP client skips certificate validation —
// use only in non-production environments.
func NewRemoteAppManager(baseURL, clusterName string, insecureSkipTLSVerify bool) types.AppManager {
	// Clone DefaultTransport so all defaults (DialContext, timeouts, keep-alives)
	// are preserved and only TLS config is overridden when needed.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if insecureSkipTLSVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // intentional, user-configured
	}

	return &RemoteAppManager{
		baseURL:     baseURL,
		clusterName: clusterName,
		client:      &http.Client{Timeout: 10 * time.Second, Transport: transport},
	}
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

// newRequest builds an HTTP request with an explicit Host header derived from
// the base URL. This prevents the Host header from being empty when the request
// passes through a reverse proxy (e.g. HAProxy) that requires it for routing.
func (r *RemoteAppManager) newRequest(method, url string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, url, body)
}

// List fetches all applications from the remote agent and returns them
// with the Cluster field set to the configured cluster name.
func (r *RemoteAppManager) List() ([]types.Application, error) {
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

	apps := make([]types.Application, 0, len(items))
	for _, item := range items {
		apps = append(apps, types.Application{
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
func (r *RemoteAppManager) Lock(app types.Application, targetBranch string, prID int) error {
	payload, err := json.Marshal(remoteLockRequest{Branch: targetBranch, PullRequestId: prID})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/apps/%s/lock", r.baseURL, app.Name)
	req, err := r.newRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("remote agent %q: failed to build request: %w", r.clusterName, err)
	}
	req.Header.Set("Content-Type", "application/json")

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
func (r *RemoteAppManager) Unlock(app types.Application) error {
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
