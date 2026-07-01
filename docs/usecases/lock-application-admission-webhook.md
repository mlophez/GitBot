# Lock Application Admission Webhook

## Purpose

Enforces deployment locks at the Kubernetes API level. When an ArgoCD `Application` is locked by the
bot, no other actor (including ArgoCD itself) may change its `spec.source.targetRevision`. The bot
registers a `ValidatingWebhookConfiguration` that intercepts every `UPDATE` on ArgoCD
`applications` and denies the ones that would move a locked application to a different target
revision, unless the change comes from the bot's own service account.

This guarantees the lock cannot be bypassed by editing the Application directly or by an ArgoCD
auto-sync, which HTTP-level bot logic alone cannot prevent.

## Actors

- **Kubernetes API server** — calls the webhook over HTTPS for every matching `UPDATE` admission
  request and enforces its allow/deny decision.
- **kubeops-agent (bot)** — serves the webhook, bootstraps its own TLS material, and injects the CA
  into the webhook configuration.
- **Any user or controller** (ArgoCD, an operator running `kubectl`, another automation) whose
  Application update is subject to admission.

## Preconditions

- The `kubeops-agent` server is running with a usable Kubernetes client
  (`config.Config.ClientSet != nil`). Without a client set, the webhook is not started and only the
  plain HTTP API on `HTTP_PORT` is served.
- The bot's service account has RBAC to:
  - `get`/`create` Secrets in its own namespace (namespaced `Role`/`RoleBinding` `gitbot-webhook-tls`).
  - `get`/`update` the `validatingwebhookconfigurations` named `lock-application-webhook`
    (rule on the `gitbot` `ClusterRole`).
- The `ValidatingWebhookConfiguration` named `lock-application-webhook` exists in the cluster
  (`manifests/gitbot-validating-webhook.yaml`). Its `caBundle` is intentionally left empty in the
  manifest; the bot populates it at runtime.
- The `gitbot` Service exposes port `8443` (targeting container port `8443`) so the API server can
  reach the webhook.
- `BOT_KUBERNETES_USERNAME` is configured with the bot's Kubernetes username (e.g.
  `system:serviceaccount:argocd:gitbot`), used to recognize the bot's own changes.

## Main flow

### Startup: TLS bootstrap and CA injection

1. On startup, after building the shared HTTP handler, `cmd/server/main.go` runs the webhook TLS
   bootstrap when `c.ClientSet != nil`.
2. `webhookNamespace()` reads the pod namespace from
   `/var/run/secrets/kubernetes.io/serviceaccount/namespace`, falling back to `argocd` for local
   development.
3. The serving certificate SANs are derived as `gitbot.<namespace>.svc` and
   `gitbot.<namespace>.svc.cluster.local` (constant `webhookService = "gitbot"`).
4. `webhooktls.EnsureTLSSecret` reads the Secret `gitbot-webhook-tls`. On first startup it generates
   a self-signed CA plus a serving certificate signed by that CA and stores both in the Secret. If a
   concurrent replica already created the Secret (`AlreadyExists`), it re-reads the winner's Secret,
   so all replicas serve the same certificate and expose the same `ca.crt`.
5. `webhooktls.PatchWebhookCABundle` writes the CA (`bundle.CACert`) into `clientConfig.caBundle` on
   every webhook of the `lock-application-webhook` `ValidatingWebhookConfiguration`. This is
   idempotent and runs on every startup.
6. The serving certificate/key are loaded with `tls.X509KeyPair` and an HTTPS server is prepared on
   `:8443`, sharing the same router as the HTTP listener (through `requestID` and the optional
   `CONTEXT_ROOT` strip).
7. Both listeners start: plain HTTP on `HTTP_PORT` always, HTTPS on `:8443` only when the bootstrap
   succeeded.

### Request: admission decision

1. The API server sends an `AdmissionReview` `POST` to the webhook path
   `/api/v1/admission/apps/validate` over HTTPS on port `8443`, for any `UPDATE` on ArgoCD
   `applications` (`argoproj.io/v1alpha1`, `Namespaced` scope).
2. `app.ValidateApp` (`internal/app/app_validate_v1.go`) decodes the request and reads: the
   requesting `userInfo.username`, the old and new `spec.source.targetRevision`, and the
   `bot.gitbot.io/locked` annotation on the old object.
3. The decision, in order:
   - If `username == BOT_KUBERNETES_USERNAME`, **allow** ("Change allowed from '<bot>'").
   - Else if the old and new `targetRevision` are equal, **allow** ("TargetRevision did not change").
   - Else if the application is not locked (`bot.gitbot.io/locked` != `"true"`), **allow**
     ("Application is not locked").
   - Otherwise **deny** ("TargetRevision change denied: Application is locked, target revision
     changed and username is not '<bot>'").
4. The handler always responds HTTP `200` with an `AdmissionReview` body whose `response.allowed` is
   `true` or `false`, as required by the admission webhook protocol.

## Alternative and error flows

- **TLS bootstrap fails** (no client set, missing RBAC, API error): the failure is logged
  ("... serving HTTP only") and the HTTPS listener is not started. The bot keeps serving the plain
  HTTP API on `HTTP_PORT`; the admission webhook is simply absent until the next successful startup.
- **Malformed admission request body**: `ValidateApp` cannot decode the JSON, logs the error, and
  responds `allowed=false` with UID `"0000"` and message "Server Error: invalid request body".
- **Webhook unreachable while `failurePolicy: Fail`**: because the `ValidatingWebhookConfiguration`
  is configured to fail closed, if the bot is down or the webhook cannot be reached, the API server
  rejects the matching `UPDATE`s. This preserves the lock guarantee at the cost of blocking
  Application updates during a bot outage.
- **`CONTEXT_ROOT` set on the deployment**: the webhook `clientConfig.service.path` in
  `manifests/gitbot-validating-webhook.yaml` must include the same prefix, otherwise the API server
  hits a 404 and (under `failurePolicy: Fail`) blocks all matching Application updates.

## References

- Admission handler and decision logic: `internal/app/app_validate_v1.go`
- Route registration and TLS bootstrap wiring: `cmd/server/main.go`
- TLS material generation, Secret persistence, and caBundle injection:
  `pkg/webhooktls/webhooktls.go`
- Webhook configuration manifest: `manifests/gitbot-validating-webhook.yaml`
- RBAC (Secret Role/RoleBinding and webhook ClusterRole rule): `manifests/gitbot-rbac.yaml`
- Service/Deployment 8443 port: `manifests/gitbot-service.yaml`, `manifests/gitbot-deployment.yaml`
