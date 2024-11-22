# GitBot

GitBot is a tool designed to manage the lifecycle of changes in pull requests for Git providers like ArgoCD and FluxCD.
This bot allows you to block pull requests before they are merged, enabling thorough testing of changes in controlled environments.
Ensure all modifications are verified and tested before integrating them into your main branch.

## Commands

argo/flux/bot

bot lock
bot deploy
bot test

bot unlock
bot undeploy
bot rollback

# Nuevos

bot deploy dev
bot rollback dev
bot decline
bot merge
