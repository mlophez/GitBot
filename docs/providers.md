# Git Provider Support

GitBot supports two Git providers: GitHub and Bitbucket. Each provider has its own webhook format, and GitBot uses a specific parser to handle the incoming webhooks.

## GitHub

When GitHub sends a webhook, the bot parses the event and identifies the type of action (e.g., `opened`, `closed`, `merged`). Based on the event, the bot can trigger actions like deploying or rolling back.

## Bitbucket

Bitbucket webhooks have a different structure. The bot parses the webhook to identify pull request events, and then it triggers the appropriate action in the GitOps system.

### Parsing Implementation

Each provider has its own implementation of the `HookParser` interface, which defines the method for parsing the webhook body.

- **GitHub**: `GitHubParser` in `github.go`.
- **Bitbucket**: `BitbucketParser` in `bitbucket.go`.
