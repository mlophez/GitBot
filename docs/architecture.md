# System Architecture

GitBot consists of several components that interact to manage the lifecycle of pull requests in GitOps workflows.

## Key Components

- **Webhook Listener**: Receives and processes events from GitHub or Bitbucket.
- **Command Processor**: Executes commands like `deploy` or `rollback`.
- **Kubernetes Integration**: Communicates with ArgoCD or Flux to manage the application deployments.

## Flow of Events

1. A webhook is received from GitHub or Bitbucket.
2. The webhook is parsed and a `GitHook` object is created.
3. Based on the command (e.g., `deploy` or `rollback`), the bot interacts with the GitOps system to trigger deployments or rollbacks.
