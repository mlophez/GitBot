# Code Style

## Formatting & linting

Use the standard Go toolchain only; there is no external linter configured.

- Format before committing: `gofmt -w .`
- Static analysis: `go vet ./...`

## Naming conventions

- Follow standard Go naming: `PascalCase` for exported identifiers,
  `camelCase` for unexported ones, short receiver names.
- Files are named after the single object they contain (snake_case). Use-case
  files carry a `_v1.go` suffix (e.g. `app_lock_v1.go`). Adapter files are named
  by their role (e.g. `app_manager_argo.go`, `provider_bitbucket.go`).
- Every exported function, type, method and constant must have a Go doc comment
  starting with the symbol name. Every package must have a package comment on the
  `package` declaration in its most representative file.
- `*_v1.go` handler comments must state the HTTP method and path, what they do on
  success, and which 4xx codes they return and why. Interface comments must state
  what the interface abstracts and who implements it.

## File organization

- Code is organised as vertical slices under `internal/`, one Go package per
  business concern (`app`, `event`, `status`, plus cross-cutting `config` and
  `logger`).
- The layout inside each slice is flat: there are no `domain/` or `adapter/`
  subpackages. The Clean Architecture split is enforced by file convention and by
  imports — domain files import only the standard library, adapter files import
  infra SDKs, and `*_v1.go` use cases wire the two together.
- One exported object per file. Adapter-internal DTOs (e.g. Bitbucket webhook
  JSON shapes) stay co-located with their adapter.
- Dependencies are wired manually in `cmd/server/main.go`. No dependency
  injection framework.

## Domain objects

Domain types (the pure-core value types of each slice, e.g. `Application`,
`Event`, `PullRequest`, `EventResponse`) follow these rules:

- **Validate at creation, always.** Every domain type implements `Validate() error`
  and is validated at its creation point — the place where it is built from its
  source data (a transport DTO, or a record read from a backend). This is the
  single standardized rule; construction stays as a plain struct literal (no
  mandatory constructor).
- **Validation co-locates with the DTO→domain transformation.** Where the object
  is created decides where `Validate()` runs:
  - When the transformation lives in the **use case** (the common case), the use
    case validates the object right after building it.
  - When the object is built inside an **adapter** — either an input adapter whose
    parsing is standardized across providers (`Provider.ParseEvent`, shared by
    Bitbucket/GitHub) or a backend read adapter (a repository / storage /
    Kubernetes client) — that adapter validates at the creation point and returns
    the error.
- **Error handling depends on the source.** Input received from the outside (a
  webhook) propagates the validation error to the entry use case, which answers
  `400 Bad Request` and does not enqueue or act on the object. Objects hydrated in
  bulk from a backend (e.g. `AppManager.List`) are filtered with the
  `keepValidApps` helper: an invalid record is discarded and logged so one
  malformed item cannot fail the whole listing.
- **Read freely, mutate through methods.** Fields are exported and may be read
  directly (they are also serialised to JSON). State is never mutated from outside
  through field assignment; it changes only through the type's methods, which
  express use-case operations (e.g. `Application.Lock`, `Unlock`, `Sanitize`) and
  return a new copy rather than mutating the receiver. Do not add generic setters.
- **Minimal, safe invariants.** `Validate()` checks only what is unequivocally
  invalid (required identifiers, recognized enum values). It does not enforce
  speculative rules that legitimate source data might break.

## Error handling & logging

- Return `error` values up the call stack; do not panic for expected failures.
  Use cases translate domain errors into the appropriate HTTP status code.
- Logging uses the standard library `log/slog` with a JSON handler (configured in
  `cmd/server/main.go`). Obtain a request-scoped logger via
  `logger.WithRequestID` so log lines carry the request ID set by the middleware.
- Adapters report observable facts (e.g. an API call failed); use cases decide
  what to do about them.

## Testing

- Tests use the standard library `testing` package, with table-driven tests for
  the pure core (e.g. `parseAction`, the event filters, `Application` methods).
- Test files live next to the code they cover and mirror the source file name:
  `foo.go` is tested in `foo_test.go` (e.g. `app.go` → `app_test.go`,
  `event_response.go` → `event_response_test.go`). One test file per source file,
  matching the one-object-per-file rule — do not group tests for several source
  files into a single `*_test.go`.
- Required for a change to be accepted: `go test ./...` must pass. Tests are
  recommended for new logic but are not mandatory per change. There is no
  coverage threshold.
- Note: the Go test files under `tests/` reference the legacy module path and are
  not part of the current suite.

## Dependencies

- Managed with Go modules (`go.mod` / `go.sum`); add one with `go get` and tidy
  with `go mod tidy`.
- Prefer the standard library. Add a third-party dependency only when it removes
  meaningful complexity, and never to introduce a dependency injection framework.

## Git workflow

- Trunk-based: work is committed to the `dev` branch and integrated into `main`
  via pull request.
- Commit messages follow Conventional Commits (`feat:`, `fix:`, `refactor:`, …) —
  recommended but not strictly enforced.
- Do not add co-author trailers to commits.

## Forbidden patterns

- Business logic, policy or domain-state branching inside adapters — adapters are
  translators only.
- `app` importing `event` (the dependency direction is one-way).
- More than one exported object per file.
- Exported symbols without a doc comment; packages without a package comment.
- Dependency injection frameworks.
