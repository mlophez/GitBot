# CLAUDE.md

## Purpose

kubeops-agent (GitBot) is a webhook bot for Bitbucket and GitHub that reacts to
pull request events and bot commands to point ArgoCD apps at a PR branch and lock
the PR until the deployment is unlocked.

## Main commands

- Run locally: `CONFIG_FILE=config.yaml go run ./cmd/server/main.go`
- Test: `go test ./...`
- Format: `gofmt -w .`
- Vet: `go vet ./...`
- Build image (Podman): `make build-image`
- Push image to ECR: `make publish-image`

## Docs

Sources of truth for this project — read them before planning or writing code:

- `docs/architecture.md` — project architecture.
- `docs/code-style.md` — coding conventions.
- `docs/design.md` — design system.

## Project notes

- The Go module is `gitbot` (not `kubeops-agent`); internal imports use
  `gitbot/internal/...`.
- The `Makefile` `run` and `test` targets have stale paths (`./cmd/main.go`,
  `cd src/`) — prefer the commands above directly.
- The container image targets ECR `eu-south-2`; image build/push uses Podman, not
  Docker.
- `make get-token` extracts the Kubernetes service account token used by the bot.
- `cmd/repair/main.go` is the reconciliation CronJob entrypoint: it runs once,
  lists apps and unlocks those whose PR is no longer open (uses
  `app.RepairLocks`).
- Known legacy: `pkg/argocd/` (superseded by `internal/app`, ignore), test files
  under `tests/` (legacy module path), and `SecurityRule` (parsed but not yet
  enforced in `internal/event/event_process_v1.go`).
