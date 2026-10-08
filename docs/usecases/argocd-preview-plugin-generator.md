# ArgoCD Preview Plugin Generator

## Purpose

Feeds an ArgoCD `ApplicationSet` that creates one preview environment per pull request, but only
for pull requests that are **open and ready for review** (not draft). ArgoCD's native
`pullRequest.bitbucket` generator ignores the Bitbucket `draft` flag and cannot filter on it, so
every draft pull request spins up a full preview (namespace, database, ingress, DNS).

kubeops-agent implements the ArgoCD
[plugin generator](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/Generators-Plugin/)
contract: ArgoCD polls `POST /api/v1/getparams.execute` and receives one parameter set per eligible
pull request. The parameter names match the native `pullRequest` generator, so the ApplicationSet
template does not change when switching generators.

## Actors

- **ArgoCD ApplicationSet controller** — polls the endpoint every `requeueAfterSeconds` and
  creates, updates or prunes the preview Applications from the returned list.
- **kubeops-agent (bot)** — authenticates the call, lists the open pull requests from Bitbucket
  Cloud and selects the eligible ones.
- **Bitbucket Cloud REST API** — source of truth for pull request state and draft flag.

## Preconditions

- `ARGOCD_PLUGIN_TOKEN` is set on the bot (key in the `gitbot` Secret consumed through `envFrom`
  in `manifests/gitbot-deployment.yaml`). It is a token dedicated to this endpoint, distinct from
  `API_TOKEN` and `WEBHOOK_TOKEN`.
- The bot has a Bitbucket token that can read pull requests of the target repository:
  `BITBUCKET_<WORKSPACE>_<REPO>_TOKEN` (e.g. `BITBUCKET_FIRMAPRO_PLATFORM_TOKEN`, a repository
  access token of `firmapro/platform` with pull request read scope), or the fallback
  `BITBUCKET_BEARER_TOKEN` if that one has access.
- ArgoCD holds the same token in a Secret and references it from the plugin ConfigMap (see
  [ArgoCD-side configuration](#argocd-side-configuration)).

## Endpoint contract

`POST /api/v1/getparams.execute` (path fixed by ArgoCD), header `Authorization: Bearer <token>`.

Request body (sent by ArgoCD; `input.parameters` comes verbatim from the generator block):

```json
{
  "applicationSetName": "platform-preview",
  "input": {
    "parameters": {
      "workspace": "firmapro",
      "repo": "platform",
      "targetBranch": "dev"
    }
  }
}
```

All three input parameters are required, so the endpoint is not tied to a specific repository or
branch.

Response body (`200`), one object per eligible pull request, every value a string:

```json
{
  "output": {
    "parameters": [
      {
        "number": "123",
        "title": "Add feature",
        "branch": "feature/my_branch",
        "branch_slug": "feature-my-branch",
        "target_branch": "dev",
        "target_branch_slug": "dev",
        "head_sha": "abc123def456",
        "head_short_sha": "abc123de",
        "head_short_sha_7": "abc123d",
        "author": "jdoe"
      }
    ]
  }
}
```

Parameter semantics mirror ArgoCD's native generator: `branch_slug` / `target_branch_slug` are
slugified with `gosimple/slug` (max 50 characters, `_` turned into `-`), `head_short_sha` is the
first 8 characters of `head_sha` (7 for `head_short_sha_7`), and `author` is the Bitbucket
nickname. When no pull request is eligible the response is `{"output":{"parameters":[]}}`.

Status codes:

- `200` — complete list of eligible pull requests (possibly empty).
- `400` — body not decodable, or `workspace`, `repo` or `targetBranch` empty. Bitbucket is not
  called.
- `401` — `Authorization` header missing or not matching `ARGOCD_PLUGIN_TOKEN`. Bitbucket is not
  called.
- `502` — the Bitbucket listing failed (see error flows). No parameter list is returned.
- `503` — `ARGOCD_PLUGIN_TOKEN` is not configured. The endpoint fails closed instead of being
  left open.

## Main flow

1. ArgoCD sends the request with the Bearer token from its plugin ConfigMap.
2. `requiredTokenAuth` (`cmd/server/middleware.go`) rejects the request with `503` when the token
   is not configured, or `401` when it does not match (constant-time comparison).
3. `preview.GetParams` (`internal/preview/get_params_v1.go`) decodes and validates the input.
4. `BitbucketClient.ListOpenPullRequests` (`internal/event/provider_bitbucket.go`) calls
   `GET /2.0/repositories/{workspace}/{repo}/pullrequests?state=OPEN&pagelen=50` and follows the
   `next` link until the last page. Each pull request is translated to `preview.PullRequest`
   (including `draft`) and validated.
5. The pure function `selectPreviewable` keeps pull requests that are `OPEN`, `draft == false` and
   whose destination branch equals `targetBranch`; `toParams` renders each one.
6. The bot answers `200` with the parameter list. ArgoCD reconciles the preview Applications.

Filtering is done client-side on purpose. Bitbucket documents the `draft` field on the
`pullrequest` object and also accepts `q=draft=false` on the listing, but the draft rule is
business policy and lives in the pure core, not in the adapter. It also avoids depending on how
Bitbucket interprets `q` for drafts, which already changed once (BCLOUD-23659, fixed April 2025).

Resulting behaviour:

- Open, ready pull request against `targetBranch` → preview created.
- Draft pull request → no preview. Once marked ready, the preview appears on the next poll
  (`requeueAfterSeconds`).
- Ready pull request moved back to draft → it disappears from the list and ArgoCD deletes its
  preview. This is accepted behaviour.
- Pull request against another branch, or merged / declined / superseded → no preview.

## Alternative and error flows

The ApplicationSet uses `prune: true`: any pull request missing from a successful response has its
preview deleted, so an empty or truncated list on a Bitbucket failure would wipe every open
preview. The endpoint therefore **fails closed**: on any of the following it answers `502` and
never returns a list. ArgoCD then keeps the existing Applications and retries.

- Network error or request timeout (the request context is cancelled when ArgoCD's
  `requestTimeout` expires).
- Bitbucket answers anything other than `200` (`401`, `429`, `5xx`, …) on any page.
- A page body is not valid JSON.
- A pull request fails validation (non-positive id or missing branch). One invalid record fails
  the whole listing instead of being skipped, unlike `keepValidApps` for applications, because a
  skipped pull request would lose its preview.
- A `next` link that does not point to the Bitbucket API (the bearer token is never sent to
  another host), or a pagination chain longer than 100 pages.

## ArgoCD-side configuration

These resources live in the `argocd` namespace and are applied from the `platform`/ArgoCD side,
not from this repository.

Secret holding the shared token (ArgoCD requires the `app.kubernetes.io/part-of: argocd` label to
resolve `$<secret>:<key>` references):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gitbot-preview-plugin
  namespace: argocd
  labels:
    app.kubernetes.io/part-of: argocd
type: Opaque
stringData:
  token: <same value as ARGOCD_PLUGIN_TOKEN>
```

Plugin ConfigMap. `baseUrl` points to the in-cluster `gitbot` Service (port `80` → container
`8080`, `manifests/gitbot-service.yaml`, deployed in `argocd` by `manifests/kustomization.yaml`).
If the bot runs with `CONTEXT_ROOT`, append that prefix to `baseUrl`.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gitbot-preview-plugin
  namespace: argocd
data:
  token: "$gitbot-preview-plugin:token"
  baseUrl: "http://gitbot.argocd.svc.cluster.local"
  requestTimeout: "30"
```

Generator block replacing `pullRequest.bitbucket` in the ApplicationSet (the template keeps using
`{{.number}}` and `{{.branch}}`):

```yaml
generators:
  - plugin:
      configMapRef:
        name: gitbot-preview-plugin
      input:
        parameters:
          workspace: firmapro
          repo: platform
          targetBranch: dev
      requeueAfterSeconds: 120
```

## References

- Use case handler: `internal/preview/get_params_v1.go`
- Eligibility rule and parameter rendering: `internal/preview/previewable.go`
- Domain type and provider interface: `internal/preview/pull_request.go`,
  `internal/preview/pull_request_lister.go`
- Bitbucket listing adapter: `internal/event/provider_bitbucket.go` (`ListOpenPullRequests`)
- Fail-closed authentication: `cmd/server/middleware.go` (`requiredTokenAuth`)
- Route registration: `cmd/server/main.go`
