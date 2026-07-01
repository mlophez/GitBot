# Architecture

## Overview

kubeops-agent (GitBot) is a webhook bot for Bitbucket and GitHub that listens
for pull request events and manages ArgoCD/FluxCD application deployments. When
a PR is opened or a user comments a command, the bot points the ArgoCD app to
the PR branch so the change can be tested in a real environment before merge,
and it locks the PR to block merges until the deployment is unlocked.

## Stack

- Go 1.22.2 — Go module name is `gitbot` (all internal imports use `gitbot/internal/...`)
- Standard library `net/http` with Go 1.22 routing (`router.HandleFunc("METHOD /path", ...)`)
- Structured logging via the standard library `log/slog` (JSON handler)
- Kubernetes / ArgoCD: ArgoCD `Application` custom resources patched through the Kubernetes API
- Bitbucket Cloud REST API for PR data and comments
- Podman for building and pushing the container image to ECR

## Project layout

- `cmd/server/` — HTTP server entrypoint. `main.go` wires every dependency by
  hand and, when a Kubernetes client is available, bootstraps the admission
  webhook TLS (via `pkg/webhooktls`) and starts a second HTTPS listener on port
  8443 serving the same router as the plain HTTP listener; `middleware.go` holds
  request-ID and API-token middleware; `event_processor.go` is the background
  worker; `web.go` serves the embedded operator panel (`index.html`).
- `cmd/repair/` — entry point for the reconciliation CronJob; wires `RepairLocks`
  from `internal/app/app_repair_v1.go` against the configured clusters.
- `internal/` — vertical slices, one Go package per business concern. The layout
  inside each slice is flat (no `domain/` or `adapter/` subpackages).
  - `internal/app/` — ArgoCD application management (Application type, AppManager
    interface and its ArgoCD/multi-cluster/remote implementations, and the
    list/lock/unlock/validate/repair use cases).
  - `internal/event/` — pull request events (Event/PullRequest types, Provider and
    Queue interfaces, the Bitbucket adapter, the in-memory queue, and the
    create/process/notification use cases).
  - `internal/status/` — health check use case.
  - `internal/config/` — runtime configuration (Config, ClusterConfig,
    SecurityRule, and the env+YAML loader).
  - `internal/logger/` — request-ID-aware structured logging.
- `pkg/utils/` — generic helpers (`contains`, `IFTernary`).
- `pkg/webhooktls/` — bootstraps the admission webhook's TLS material: generates a self-signed CA
  and serving certificate, persists them in a Kubernetes Secret shared across replicas, and injects
  the CA into the `ValidatingWebhookConfiguration` caBundle at runtime.
- `pkg/argocd/` — mostly commented out, superseded by `internal/app`. Ignore.
- `manifests/` — Kubernetes manifests (Kustomize).
- `tests/` — sample Bitbucket webhook JSON payloads for manual use. The Go test
  files under here reference the legacy module path and are not wired to the
  current suite.

## Components

The codebase follows Clean Architecture + Vertical Slice + Screaming
Architecture. Each slice is self-contained and split into two layers, enforced
by file convention and by what each file may import:

- **Pure core (domain)** — value types and interfaces with no I/O and no external
  dependencies. Import only the standard library (and other domain files in the
  same slice). Examples: `internal/app/app.go`, `internal/event/event.go`, and
  the `parseAction`/`applyLock`/`applyUnlock`/`filterByRepoAndFiles` helpers in
  `internal/event/event_process_v1.go`. Domain value types implement
  `Validate() error` and are validated at their creation point; they are read
  directly but mutated only through use-case methods (see `docs/code-style.md`).
- **Imperative shell** — use cases (`*_v1.go`) that orchestrate the pure core and
  then call adapters, plus adapters (manager/provider/queue implementations)
  that perform I/O against Kubernetes, Bitbucket and HTTP.

Layer rules:

- Domain files import only the standard library.
- Adapter files import infra SDKs (Kubernetes, HTTP, etc.) and translate to and
  from domain types. Adapters are translators, not decision-makers: no business
  logic, no policy, no branching on domain state. If an `if` in an adapter
  decides whether something *should happen*, that decision belongs in a use case
  or a pure function.
- Use case files (`*_v1.go`) wire the two: fetch data via adapter, call a pure
  function, persist via adapter.

Dependency rules between slices:

- `event` may depend on `app` (uses `app.AppManager` / `app.Application`).
- `app` must NOT import `event`. Notification handling lives in `event` to keep
  this direction one-way.
- `config` and `logger` are cross-cutting and may be imported by anyone.

Validation rule: every domain object is validated at the point it is created from
its source data, always. The validation co-locates with the DTO→domain
transformation — normally in the use case, but in an adapter when parsing is
standardized across providers (`Provider.ParseEvent`) or when reading from a
backend (repository / storage / Kubernetes client). Objects parsed from external
input propagate the failure to the entry use case, which returns `400 Bad Request`
and does not act on the object; objects hydrated in bulk from a backend
(`AppManager.List`) discard and log invalid records instead of failing the whole
read — the shared `keepValidApps` helper in `internal/app/app_manager.go` handles
this filtering for all `AppManager` implementations. See `docs/code-style.md`.

Conventions: there are exactly two layers — the pure domain and the imperative
shell — and they live flat in the slice, separated by file convention rather than
by subpackages. One exported object per file. Each use case is written in its own
`*_v1.go` file: that file is where the two layers meet inside the vertical slice —
it calls an adapter to fetch/translate, runs pure domain logic (including
`Validate`), and calls an adapter to persist. No dependency injection frameworks
(dependencies are wired manually in `cmd/server/main.go`).

## External integrations

- **Bitbucket Cloud REST API** — the `BitbucketClient` adapter
  (`internal/event/provider_bitbucket.go`) parses incoming webhooks, fetches
  changed files and commits-behind, and writes status comments back on the PR.
  Authenticated with `BITBUCKET_BEARER_TOKEN`.
- **Kubernetes / ArgoCD API** — the `ArgoAppManager` adapter
  (`internal/app/app_manager_argo.go`) lists and patches ArgoCD `Application`
  resources (`spec.source.targetRevision` and `bot.gitbot.io/*` annotations).
- **Remote GitBot agents** — `RemoteAppManager`
  (`internal/app/app_manager_remote.go`) delegates lock/unlock to a remote GitBot
  instance over HTTP for the hub-and-spoke multi-cluster model;
  `MultiClusterAppManager` aggregates per-cluster backends.

## Persistence

There is no database. Deployment lock state is persisted as annotations on the
ArgoCD `Application` resources themselves:

- `bot.gitbot.io/locked` — `"true"` while the app is locked.
- `bot.gitbot.io/pull-request` — the PR ID that holds the lock.
- `bot.gitbot.io/rollback` — the branch to restore on unlock.

The event queue is an in-memory FIFO (`MemoryQueue` in
`internal/event/queue_memory.go`); it is not durable and is lost on restart.

## Configuration & environments

- Configuration is loaded by `EnvConfigLoader`
  (`internal/config/env_config_loader.go`) from environment variables (optionally
  sourced from an `env.ini` / `env.local.ini` file) plus an optional YAML file
  pointed to by `CONFIG_FILE` (clusters and security rules).
- Key environment variables: `HTTP_PORT` (default 8080),
  `BITBUCKET_BEARER_TOKEN`, `CONFIG_FILE`, `CLUSTER_NAME` (fallback environment
  label when the `gitbot.io/env` annotation is absent), `API_TOKEN`,
  `WEBHOOK_TOKEN`, `BOT_KUBERNETES_USERNAME`.
- Environments are identified per ArgoCD app via the `gitbot.io/env` annotation
  (e.g. `dev`, `prod`). The same binary runs as the central hub or as a remote
  agent depending on the cluster configuration.
- Secrets (Bitbucket token, API token) are supplied as environment variables; in
  Kubernetes they come from a `Secret`. Do not commit real tokens to `env.ini`.

## Architecture decisions

<!-- Cumulative mini-ADR log. Append one entry per relevant decision:
date, decision, reason. -->

None yet.
