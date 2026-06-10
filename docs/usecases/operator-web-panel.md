# Operator Web Panel

## Purpose

Provides a browser-based dashboard that lets operators inspect ArgoCD application
state across all configured clusters and perform lock/unlock actions without
needing `kubectl` or direct ArgoCD access. The panel also shows the ArgoCD sync
status of each application and allows operators to configure how frequently it
auto-refreshes.

## Actors

- **Operator** — a human user with browser access to the bot's HTTP port and
  (optionally) a valid API token.

## Preconditions

- The `kubeops-agent` server is running and reachable in a browser
  (`GET /`, served from `cmd/server/index.html` via `//go:embed`).
- The operator has a valid API token if the server is configured with
  `API_TOKEN` authentication (the token is stored in `localStorage` under the
  key `kubeops.apiToken`).

## Main flows

### View application list

1. The browser loads `GET /` and executes the embedded JavaScript.
2. On boot, `load(true)` calls `GET /api/v1/apps`.
3. The response (JSON array of application objects) is stored in `allApps` and
   rendered as cards grouped by environment, sorted alphabetically within each
   group.
4. Each card shows the application name, ArgoCD sync status badge, lock state
   badge, repository URL, current branch, and cluster.
5. The header meta field shows the total count of apps and the number with a
   non-healthy status.

### Filter applications

1. The operator types in the search box (`#search`) to filter by name or
   repository, selects an environment from `#envFilter`, or activates the "Solo
   con problemas" toggle (`#onlyProblems`).
2. `render()` re-applies all active filters against `allApps` synchronously
   (no network call).

### Lock an application

1. The operator clicks "Bloquear" on a card. The lock modal opens.
2. The operator fills in the target branch, an optional PR ID, and optionally
   checks "Forzar" (to overwrite an existing lock).
3. Clicking "Bloquear" calls `POST /api/v1/apps/{name}/lock` (with
   `?force=true` when forced) with a JSON body `{ branch, pull_request_id }`.
4. On success a toast confirms the action and `load(false)` refreshes the list.

### Unlock an application

1. The operator clicks "Desbloquear" on a locked card and confirms the dialog.
2. `POST /api/v1/apps/{name}/unlock` is called.
3. On success a toast confirms and `load(false)` refreshes the list.

### Configure the API token

1. The operator clicks "Token" in the header. The token modal opens.
2. The operator enters the bearer token and saves it.
3. The token is written to `localStorage` under `kubeops.apiToken` via
   `setToken()` and all subsequent `api()` calls attach it as
   `Authorization: Bearer <token>`.
4. An empty value clears the stored token.

### Configure the auto-refresh interval

1. The header contains a `<select id="refreshInterval">` combobox whose options
   are populated at boot by the IIFE `populateRefreshInterval()` from the
   `REFRESH_OPTIONS` constant in `cmd/server/index.html`. Available intervals
   are 5 s, 10 s, 15 s (default), 30 s, 1 m, and 5 m.
2. The previously chosen interval is restored from `localStorage` (key
   `kubeops.refreshInterval`) via `getRefreshInterval()`. If the stored value is
   absent or not in `REFRESH_OPTIONS`, the default of 15 000 ms is used.
3. When the operator selects a different value, the `change` listener calls
   `setRefreshInterval(ms)` to persist the choice and then calls
   `scheduleRefresh()` to restart the timer with the new interval immediately.
4. The auto-refresh checkbox (`#autorefresh`) gates whether the timer actually
   triggers a reload: when unchecked, `scheduleRefresh()` keeps the timer
   running but `load(false)` is not called. When rechecked,
   `scheduleRefresh()` is called to resume with the current interval.
5. The timer also checks `document.visibilityState === "visible"` so hidden
   browser tabs do not fire background requests.

## Alternative and error flows

- **401 Unauthorized**: any `api()` call that receives a 401 response
  automatically opens the token modal and shows an "No autorizado" toast. The
  operator can supply a correct token and retry manually.
- **Network error on load**: if the initial load fails, the content area shows
  an "Error de red" block with the error message. Subsequent failures (while
  `allApps` is already populated) surface as a toast instead, so the stale
  data remains visible.
- **Stored interval not in REFRESH_OPTIONS**: `getRefreshInterval()` returns
  `REFRESH_DEFAULT_MS` (15 000 ms) and the combobox selects the 15 s option.
  This handles forward/backward compatibility if the option list changes.
- **Lock conflict without force**: the server returns a non-2xx response; the
  panel shows an "No se pudo bloquear" toast with the server error message.
  The operator can reopen the lock modal and check "Forzar".

## References

- Entry point (embed): `cmd/server/index.html`
- HTTP handler that serves the file: `cmd/server/web.go`
- Apps REST endpoint called by the panel: `internal/app/app_list_v1.go`
- Lock endpoint: `internal/app/app_lock_v1.go`
- Unlock endpoint: `internal/app/app_unlock_v1.go`
