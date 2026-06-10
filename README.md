# kubeops-agent (GitBot)

## Description

kubeops-agent (GitBot) is a webhook bot for Bitbucket and GitHub that listens for
pull request events and manages ArgoCD/FluxCD application deployments. When a PR
is opened or a user comments a command, the bot points the ArgoCD app to the PR
branch so the change can be tested in a real environment before merge, and it
locks the PR to block merges until the deployment is unlocked. It also ships an
embedded operator web panel and a Kubernetes admission webhook.

## Requirements

- Go 1.22.2 (module name is `gitbot`)
- A Kubernetes cluster with ArgoCD installed (the bot patches ArgoCD
  `Application` resources via the Kubernetes API)
- `kubectl` with access to the target cluster
- A Bitbucket Cloud account and an API token for the bot
- Podman (only to build and push the container image)

## Setup

1. Clone the repository:

   ```bash
   git clone <repo-url>
   cd kubeops-agent
   ```

2. Create a local environment file (`env.local.ini`) with at least:

   ```ini
   HTTP_PORT=8080
   BITBUCKET_BEARER_TOKEN=<bitbucket-api-token>
   CONFIG_FILE=config.yaml
   CLUSTER_NAME=<fallback-env-label>
   ```

3. Create a `config.yaml` describing the clusters and (optionally) the security
   rules — see `docs/architecture.md` for the configuration model.

4. Fetch Go dependencies:

   ```bash
   go mod download
   ```

## Usage

Run the server locally:

```bash
CONFIG_FILE=config.yaml go run ./cmd/server/main.go
```

The HTTP server listens on `HTTP_PORT` (default 8080) and exposes:

- `GET /` — the embedded operator web panel
- `GET /api/v1/status` — health check
- `POST /api/v1/webhook/bitbucket` — Bitbucket webhook receiver
- `GET /api/v1/apps`, `POST /api/v1/apps/{id}/lock`, `POST /api/v1/apps/{id}/unlock`
- `POST /api/v1/notification` — ArgoCD deployment notifications
- `POST /api/v1/admission/apps/validate` — Kubernetes admission webhook

From a PR comment, drive the bot with commands such as `#argo deploy`,
`#argo deploy dev my-app` or `#argo unlock`.

To build and publish the container image:

```bash
make build-image
make publish-image
```

## Development

- Run: `CONFIG_FILE=config.yaml go run ./cmd/server/main.go`
- Test: `go test ./...`
- Format: `gofmt -w .`
- Vet: `go vet ./...`

## Documentation

- [Architecture](docs/architecture.md)
- [Code style](docs/code-style.md)
- [Design](docs/design.md)
