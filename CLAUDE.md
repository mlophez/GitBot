# kubeops-agent (GitBot)

A webhook bot for Bitbucket and GitHub that listens for pull request events and manages ArgoCD/FluxCD app deployments. When a PR is opened or a user comments a command, the bot points the ArgoCD app to the PR branch so changes can be tested before merging. It also blocks (locks) the PR to prevent merges until unlocked.

## Documentation Requirements

**All generated Go code must include documentation. No exceptions.**

### Package comments
Every package must have a comment on the `package` declaration explaining its role in the architecture and what it contains. Place it in the most representative file of the package.

```go
// Package app is the vertical slice for ArgoCD application management.
// It contains the pure domain (Application, AppManager), the adapters that
// implement the manager against Kubernetes/ArgoCD, and the use cases exposed
// via HTTP (list, lock, unlock, validate).
package app
```

### Exported symbols
Every exported function, type, method, and constant must have a Go doc comment starting with the symbol name.

```go
// AppResponse is the HTTP response representation of an ArgoCD application.
type AppResponse struct { ... }

// ListApps handles GET /api/v1/apps.
// Returns the list of all ArgoCD applications tracked in the cluster.
func ListApps(manager AppManager) http.HandlerFunc { ... }
```

### Use case handlers
Each `*_v1.go` handler comment must include:
- HTTP method and path
- What it does on success
- Which 4xx codes it returns and why

```go
// LockApp handles POST /api/v1/apps/{id}/lock.
// Points the ArgoCD app to the given branch and marks it as locked by the PR.
// Returns 404 if the app does not exist, 409 if it is already locked.
func LockApp(manager AppManager) http.HandlerFunc {
```

### Interfaces
Every interface must document what it abstracts and who implements it.

```go
// AppManager abstracts read and write operations on ArgoCD applications.
// Implemented by ArgoAppManager (same package) for the Kubernetes/ArgoCD backend.
type AppManager interface { ... }
```

### Inline comments
Add inline comments only where logic is non-obvious. Do not comment self-evident code.

## Architecture: Clean Architecture + Vertical Slice + Screaming Architecture

The codebase is organised as **vertical slices** under `internal/`. Each slice is a self-contained Go package named after the business concern it owns (`app`, `event`, `status`) — the directory tree screams what the system does, not which framework it uses.

Inside each slice we keep the classic **two-layer Clean Architecture** split:

- **Pure core (domain)** — value types and interfaces with no I/O and no external dependencies. Tested without mocks. Examples: `internal/app/application.go`, `internal/event/event.go`, `internal/event/event_process_v1.go` (the `parseAction`, `applyLock`, `applyUnlock`, `filterByRepoAndFiles`, etc. helpers).
- **Imperative shell** — use cases (`*_v1.go`) that orchestrate the pure core and then call adapters; and adapters (manager/provider/queue implementations) that perform I/O against Kubernetes, Bitbucket, HTTP, etc.

Both layers live **flat** inside the slice package — no `domain/` or `adapter/` subpackages. The split is enforced by file convention and by what each file is allowed to import:

- domain files import only the standard library (and other domain files in the same slice);
- adapter files import infra SDKs (Kubernetes, HTTP, etc.) and translate to/from domain types;
- use case files (`*_v1.go`) wire the two together — fetch data via adapter → call pure function → persist via adapter.

**One object per file.** Each exported type, interface, or struct lives in its own file. Adapter-internal DTOs (e.g. Bitbucket webhook JSON shapes) stay co-located with their adapter.

**No dependency injection frameworks.** Wire dependencies manually in `cmd/server/main.go`.

### Allowed dependencies between slices

```
event ──► app          (event uses app.AppManager / app.Application)
config, logger         (cross-cutting, imported by anyone)
```

`app` must NOT import `event`. Notification handling lives in `event` precisely to keep this direction one-way.

### Adapters are translators, not decision-makers

Every adapter (e.g. `ArgoAppManager`, `BitbucketClient`, `EnvConfigLoader`, `MemoryQueue`, `RemoteAppManager`, `MultiClusterAppManager`) implements one of the interfaces in its slice. Its only job is to translate between the external world (HTTP, Kubernetes API, Bitbucket API, etc.) and the domain types. Adapters must not contain business logic or decisions.

- **Allowed in adapters:** parsing payloads, serialising responses, making API calls, mapping external structs to domain types, setting observable facts on the event (e.g. `BotGenerated = true`).
- **Not allowed in adapters:** deciding what action to take, filtering events based on business rules, enforcing policy, branching on domain state.

If you find yourself writing an `if` in an adapter that decides whether something *should happen*, move that decision to a use case or a pure function in the same slice.

**Why this matters:** adapters are swapped out (Bitbucket → GitHub, ArgoCD → FluxCD). Logic placed in an adapter must be re-implemented for every new provider. Logic placed in the use case is inherited automatically.

## Directory Structure

```
cmd/
  server/main.go            # HTTP server entrypoint. Wires all dependencies.
  server/middleware.go      # requestID and apiTokenAuth middleware.
  server/event_processor.go # Background worker: dequeues events, enriches, processes, writes comments.
  repair/main.go            # Utility to reconcile app state (stub, not functional).

internal/             # Vertical slices — one Go package per business concern.
                      # Layout inside each slice is FLAT (no domain/adapter subdirs).
                      # Conventions: one object per file; *_v1.go is a use case.

  app/                # ArgoCD application management — package "app"
    application.go              # Application type + methods (Lock/Unlock/Sanitize) — pure
    app_manager.go              # AppManager interface — pure
    argo_app_manager.go         # ArgoAppManager: AppManager against Kubernetes/ArgoCD — adapter
    multi_cluster_app_manager.go# MultiClusterAppManager: aggregates per-cluster backends — adapter
    remote_app_manager.go       # RemoteAppManager: delegates to a remote GitBot agent — adapter
    app_response.go             # AppResponse + toAppResponse (HTTP DTO) — shell
    app_list_v1.go              # GET  /api/v1/apps
    app_lock_v1.go              # POST /api/v1/apps/{id}/lock
    app_unlock_v1.go            # POST /api/v1/apps/{id}/unlock
    app_validate_v1.go          # POST /api/v1/admission/apps/validate (admission webhook)

  event/              # Pull request events — package "event"
    event.go                    # Event + EventType — pure
    pull_request.go             # PullRequest — pure
    event_response.go           # EventResponse + EventAppStatus — pure
    queue.go                    # Queue interface — pure
    queue_item.go               # QueueItem — pure
    provider.go                 # Provider interface — pure
    process_fn.go               # ProcessFn type — pure
    bitbucket_client.go         # BitbucketClient: Provider for Bitbucket webhooks — adapter
    memory_queue.go             # MemoryQueue: generic thread-safe in-memory queue — adapter
    event_create_v1.go          # POST /api/v1/webhook/bitbucket — parse + enqueue
    event_process_v1.go         # Worker use case: parseAction, applyLock/Unlock, filters
    notification_handle_v1.go   # POST /api/v1/notification — ArgoCD deployment notifications

  status/             # Health check — package "status"
    status_v1.go                # GET /api/v1/status

  config/             # Runtime configuration — package "config" (cross-slice)
    config.go                   # Config struct
    cluster_config.go           # ClusterConfig + ClusterAuth
    config_loader.go            # ConfigLoader interface
    security_rule.go            # SecurityRule
    env_config_loader.go        # EnvConfigLoader: env.ini + optional YAML — adapter

  logger/             # Structured logging with request ID — package "logger" (cross-cutting)
    logger.go                   # WithRequestID + Logger

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

To add GitHub (or another provider), implement the `event.Provider` interface (defined in `internal/event/provider.go`):

```go
type Provider interface {
    ValidateWebhookToken(secret string, headers http.Header, body []byte) error
    ParseEvent(headers http.Header, body io.ReadCloser) (Event, error)
    GetData(Event) (Event, error)
    WriteComment(repo string, prId int, parentId int, msg string) error
    WriteEventResponse(repo string, prId int, parentId int, resp *EventResponse, clusterName string) error
}
```

Place the implementation as a new file inside `internal/event/` (e.g. `github_client.go`), then register a new route and handler in `cmd/server/main.go` following the same pattern as `BitbucketClient`.

## Adding a New CD Platform (FluxCD, etc.)

The `app.AppManager` interface (defined in `internal/app/app_manager.go`) abstracts the CD platform:

```go
type AppManager interface {
    List() ([]Application, error)
    Lock(app Application, targetBranch string, prID int) error
    Unlock(app Application) error
}
```

`ArgoAppManager` in `internal/app/argo_app_manager.go` implements this for ArgoCD. Add a new implementation as a sibling file (e.g. `flux_app_manager.go`) and wire it in `cmd/server/main.go`.

## Known Incomplete Areas

- `cmd/repair/main.go` — repair/reconcile utility is a stub, not functional.
- `pkg/argocd/` — mostly commented out, superseded by `internal/app/argo_app_manager.go`. Ignore it.
- `TODO` in `internal/event/event_process_v1.go` — double-lock for app-of-apps pattern is disabled to avoid infinite loops.
- Test files under `tests/` reference the old module path `github.com/MLR96/argocd-bot` — legacy, not wired to the current test suite.
- `SecurityRule` is parsed from config (`internal/config/security_rule.go`) but not yet enforced in `internal/event/event_process_v1.go`.

## Module Name

The Go module is `gitbot` (not `kubeops-agent`). All internal imports use `gitbot/internal/...`.
