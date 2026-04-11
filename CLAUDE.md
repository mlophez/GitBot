# kubeops-agent (GitBot)

A webhook bot for Bitbucket and GitHub that listens for pull request events and manages ArgoCD/FluxCD app deployments. When a PR is opened or a user comments a command, the bot points the ArgoCD app to the PR branch so changes can be tested before merging. It also blocks (locks) the PR to prevent merges until unlocked.

## Documentation Requirements

All new code must be documented. Follow these rules without exception:

- **Package comment** — every package must have a comment on the `package` declaration explaining its role in the architecture and what it contains. Place it in the most representative file of the package.
- **Exported functions and types** — every exported symbol must have a Go doc comment starting with the symbol name. Include: what it does, what it returns, and any notable HTTP status codes or error conditions for handlers.
- **Inline comments** — add comments only where logic is non-obvious. Do not comment self-evident code.
- **Use case files** — each `*_v1.go` file must document: the HTTP method + path it handles, preconditions that return 4xx, and what the success response contains.

Example for a use case:
```go
// LockApp handles POST /api/v1/apps/{id}/lock.
// Points the ArgoCD app to the given branch and marks it as locked by the PR.
// Returns 404 if the app does not exist, 409 if it is already locked.
func LockApp(manager types.AppManager) http.HandlerFunc {
```

## Architecture: Pure Core / Imperative Shell

The codebase follows the **Pure Core / Imperative Shell** pattern:

- **Pure core** — domain logic lives in pure functions with no side effects. Example: `internal/app/usecase.go` (`lockApp`, `unlockApp`, `filterAppByRepoAndFiles`). These are tested without mocks.
- **Imperative shell** — use cases in services orchestrate calls to pure functions and then call repositories/providers that perform I/O. Example: `internal/app/service.go` calls `lockApp()` (pure) then `repository.Update()` (I/O).
- **Use cases** are in `service.go` per package. Each use case: fetch data → call pure function → persist result.
- **No dependency injection frameworks.** Wire dependencies manually in `cmd/server/main.go`.

## Directory Structure

```
cmd/
  server/main.go      # HTTP server entrypoint. Wires all dependencies.
  repair/main.go      # Utility to reconcile app state (in progress).

internal/             # Use cases (imperative shell) — package "internal"
                      # One file per use case, named {domain}_{action}_{version}.go
  app_list_v1.go      # GET  /api/v1/apps            — list all ArgoCD apps
  app_lock_v1.go      # POST /api/v1/apps/{id}/lock   — lock an app to a PR branch
  app_unlock_v1.go    # POST /api/v1/apps/{id}/unlock — unlock and restore branch

  types/              # Pure domain — no external dependencies, no I/O
    application.go    # Application type + methods: Lock(), Unlock(), Sanitize()
    application_manager.go  # AppManager interface: List, Lock, Unlock

  adapter/            # Infrastructure implementations of types interfaces
    argocd.go         # ArgoAppManager: implements AppManager against Kubernetes API

  app/                # Legacy: Application domain (ArgoCD apps) — to be migrated
    domain.go         # Application struct
    usecase.go        # Pure functions: lockApp, unlockApp, filterAppByRepoAndFiles
    service.go        # Service: orchestrates pure fns + repository calls
    argocd.go         # KubeRepository: talks to Kubernetes API to CRUD ArgoCD CRDs

  event/              # Event processing pipeline
    event.go          # EventType enum and Event struct
    domain.go         # PullRequest, QueueItem, Queue/Provider interfaces, SecurityRule
    service.go        # Service.Process(): determines action, validates PR, calls app service
    handler.go        # HTTP handler: parses webhook, enqueues item
    worker.go         # Worker: dequeues, enriches event, calls service, writes comment
    provider/
      bitbucket.go    # BitbucketProvider: parses webhooks, fetches diff/commits, writes comments

  event/queue/
    memory.go         # Generic thread-safe in-memory queue

  config/
    service.go        # Loads config from env + YAML, initializes k8s clientset
    file.go           # ConfigFile YAML schema + validation + conversion to SecurityRule[]

  notification/
    notification.go   # HTTP handler for deployment status notifications from ArgoCD

pkg/
  utils/utils.go      # Generic helpers: contains(), IFTernary()
  argocd/             # Mostly unused/commented out. Ignore.

tests/
  webhooks/bitbucket/ # Sample Bitbucket webhook JSON payloads for manual/test use
```

## Key Domain Types

```go
// An ArgoCD application tracked by the bot
type Application struct {
    Name          string
    Repository    string   // e.g. "https://bitbucket.org/org/repo.git"
    Branch        string   // current targetRevision
    Paths         []string // file paths this app is responsible for
    Locked        bool
    PullRequestId int      // PR that locked this app (-1 if unlocked)
    LastBranch    string   // branch before lock (for rollback on unlock)
    Environment   string   // e.g. "dev", "prod" — from gitbot.io/env annotation
    ContainOther  bool     // app-of-apps pattern
}

// A pull request event
type Event struct {
    Type        EventType  // Opened, Updated, Declined, Merged, Commented
    Repository  string
    Author      string
    Comment     string
    CommentId   int
    PullRequest PullRequest
}

// Actions derived from events
const (
    LOCK_ACTION   Action = iota  // triggered by: PR opened/updated, or #argo deploy/lock/test
    UNLOCK_ACTION                // triggered by: PR merged/declined, or #argo unlock/undeploy/rollback
    UNKNOWN_ACTION
)
```

## Event Flow

```
Bitbucket webhook POST /api/v1/webhook/bitbucket
  → Handler.Handle(): parse event → enqueue QueueItem{Event, Provider}
  → Worker.Start(): dequeue → Provider.GetData() (fetch files changed + commits behind)
  → Service.Process(): determine action → validate PR → lock/unlock apps
  → Provider.WriteComment(): post result back to PR
```

## Bot Commands (PR comments)

Commands are parsed from PR comments matching:
```
(/|#)(argo|flux|bot) (lock|deploy|test|unlock|undeploy|rollback) [environment] [appname]
```

| Command | Action | Example |
|---------|--------|---------|
| `#argo deploy` | Lock apps in all envs | `#argo deploy` |
| `#argo deploy dev` | Lock apps in `dev` env | `#argo deploy dev` |
| `#argo deploy dev my-app` | Lock specific app | `#argo deploy dev my-app` |
| `#argo lock` / `#argo test` | Same as deploy | |
| `#argo unlock` / `#argo undeploy` / `#argo rollback` | Unlock | |

Lock/unlock also happen automatically on PR merge/decline (UNLOCK) and PR open (if configured).

## ArgoCD Annotations

The bot reads and writes these annotations on ArgoCD Application resources:

| Annotation | Purpose |
|---|---|
| `bot.gitbot.io/locked` | `"true"` when app is locked |
| `bot.gitbot.io/pull-request` | PR ID that locked the app |
| `bot.gitbot.io/rollback` | Branch to restore on unlock |
| `gitbot.io/env` | Environment label (e.g. `"dev"`) |
| `gitbot.io/contain-other-apps` | `"true"` for app-of-apps |
| `argocd.argoproj.io/manifest-generate-paths` | Base path for file matching |

Lock modifies `spec.source.targetRevision` to point to the PR source branch.

## Configuration (config.yaml)

```yaml
clusters:
  - name: tools
    auth:
      type: serviceaccount   # uses in-cluster service account

security:
  groups:
    - name: myteam
      users: [user@example.com]
  rules:
    - repository: https://bitbucket.org/org/repo.git
      filepattern: ["overlays/test/**"]
      action: ["lock", "unlock"]
      group: ["myteam"]
      user: [other@example.com]
```

Environment variables (set via `env.local.ini` or environment):
- `HTTP_PORT` — defaults to 8080
- `BITBUCKET_BEARER_TOKEN` — Bitbucket API token
- `CONFIG_FILE` — path to config.yaml
- `CLUSTER_NAME` — fallback env label when `gitbot.io/env` annotation is absent

## Running and Building

```bash
# Run locally
CONFIG_FILE=config.yaml go run ./cmd/server/main.go

# Run tests
go test ./...

# Build container image (uses Podman)
make build-image

# Push image to ECR
make publish-image

# Extract k8s service account token
make get-token
```

The Makefile `run` and `test` targets have stale paths — prefer the commands above directly.

## Adding a New Git Provider

To add GitHub (or another provider), implement the `event.Provider` interface:

```go
type Provider interface {
    ParseEvent(headers http.Header, body io.ReadCloser) (Event, error)
    GetData(Event) (Event, error)
    WriteComment(repo string, prId int, parentId int, msg string) error
}
```

Place the implementation in `internal/event/provider/github.go`, then register a new route and handler in `cmd/server/main.go` following the same pattern as Bitbucket.

## Adding a New CD Platform (FluxCD, etc.)

The `app.Repository` interface abstracts the CD platform:

```go
type Repository interface {
    List(ctx context.Context) ([]Application, error)
    Update(ctx context.Context, app Application) (Application, error)
    Clean(ctx context.Context, app Application) (Application, error)
}
```

`KubeRepository` in `internal/app/argocd.go` implements this for ArgoCD. Add a new implementation for FluxCD and wire it in `app.NewService()`.

## Known Incomplete Areas

- `cmd/repair/main.go` — repair/reconcile utility is a stub, not functional.
- `internal/cluster/cluster.go` — empty struct, multi-cluster routing not implemented.
- `internal/comment/` — unused package, duplicates provider comment logic.
- `pkg/argocd/` — mostly commented out, conflicts with `internal/app/argocd.go`. Ignore it.
- `TODO` in `event/service.go:95` — double-lock for app-of-apps pattern is disabled to avoid infinite loops.
- Test files under `tests/` reference old module path `github.com/MLR96/argocd-bot` — legacy, not wired to current test suite.
- `SecurityRule` is parsed from config but not yet enforced in `event/service.go` (the `rules` field exists but enforcement code is commented out).

## Module Name

The Go module is `gitbot` (not `kubeops-agent`). All internal imports use `gitbot/internal/...`.
