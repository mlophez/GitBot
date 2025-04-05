# 📥 Use Case: Pull Request Opened

When a new pull request is opened, the bot automatically adds an informative comment to guide the user through the available lifecycle management commands.

## 🔔 Bot Behavior

The bot responds with a comment like the following:

```
The lifecycle of this pull request is managed by the Git bot!
Environments: dev, test, demo, demosign, prod, santander, bbva, mercedes
The following commands are available:
    #gitbot deploy all
    #gitbot deploy <env>
    #gitbot deploy <env> <appname>
    #gitbot deploy-force all
    #gitbot deploy-force <env>
    #gitbot deploy-force <env> <appname>
    #gitbot rollback all
    #gitbot rollback <env>
    #gitbot rollback <env> <appname>
```

## 🧠 Purpose

This comment serves to:

- Confirm that the bot is active and monitoring the PR.
- Display the list of detected environments.
- Show available commands the user can trigger via PR comments.

## 🛠️ Notes

- The list of environments is extracted dynamically from the repository or configuration.
- The user must manually comment using one of the listed commands to trigger any deployment or rollback.
- Commands are executed per PR and are associated with the branch of the pull request.
