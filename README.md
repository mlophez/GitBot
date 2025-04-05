# GitOps Pull Request Bot

A Git provider-agnostic bot (Bitbucket, GitHub, etc.) designed to manage the lifecycle of pull requests for GitOps-based Kubernetes workflows using tools like Flux or ArgoCD.

## ✨ Key Features

- PR event handling: open, update, merge, decline, and comment-based triggers.
- Dynamic environment deployments via GitOps.
- Preview environments before merging to `main`.
- Works with Flux and ArgoCD.
- Built in Go with vertical slicing architecture.

## 📦 Use Case

When a pull request is opened on a GitOps repository, this bot listens for events but **does not automatically deploy changes**.

Instead, deployment actions are triggered explicitly by users through comments on the pull request.

### 🔁 Supported Commands

- `#gitbot deploy <environment>`
  Deploys the PR to the specified environment and locks all related apps to the PR branch.
  ❗ If the app is already locked by another PR, the deployment will be rejected.

- `#gitbot deploy <environment> <app-name>`
  Same as above, but limits the deployment to the specified application.

- `#gitbot deploy-force <environment>`
  Forces redeployment of all apps in the environment and **takes over any existing lock**, even if the app was previously deployed by another PR.

- `#gitbot deploy-force <environment> <app-name>`
  Same as above, but targets a specific app and **steals the lock** if necessary.

- `#gitbot rollback <environment>`
  Rolls back all apps in the specified environment to their previous state and removes any locks.

- `#gitbot rollback <environment> <app-name>`
  Rolls back a specific app in the given environment and removes its lock.

These commands enable precise and flexible GitOps workflows.
Locking ensures a single PR controls the lifecycle of each application in an environment.
The `deploy-force` command should be used with caution, as it can override active locks.

## 🔧 Supported Events

- `pull_request:open`
- `pull_request:update`
- `pull_request:merged`
- `pull_request:declined`
- `comment:deploy`
- `comment:rollback`
- `comment:deploy-force`

## 📂 Documentation

All usage details and capabilities are available in [`docs/`](./docs).

## 🚀 Getting Started

```bash
# clone the repo
git clone https://github.com/yourorg/gitops-pr-bot.git

# build
go build -o gitops-pr-bot ./cmd
