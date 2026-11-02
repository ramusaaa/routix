# Routix agent entry point

## Scope and navigation

This is the Routix Go framework repository, module `github.com/ramusaaa/routix`. It also contains a project CLI and Go source generators. Generated applications are separate modules; compiling this repository does not compile the source strings emitted by generators.

Read in this order:

1. [README](README.md): installation, supported capabilities, examples, and limits.
2. [Architecture](docs/architecture.md): ownership, request execution, snapshot publication, and subsystem map.
3. [Reference](docs/reference.md): public APIs, configuration, CLI, and contracts.
4. [Verification](docs/verification.md): runnable checks, local-only test coverage, fuzzing, and API compatibility.
5. [Performance](docs/performance.md): benchmark setup, measured scope, and remaining gaps.
6. [Limitations](docs/limitations.md): unresolved findings and release concerns.
7. [Performance target](docs/performance-target.md): the 48-cell Gin acceptance matrix and `tools/benchmark-target.py` checker.

Documentation is a navigation aid. Check the implementation before changing behavior. When code and documentation disagree, reproduce the behavior and update both. Do not claim complete understanding from this index alone.

## Repository map

| Area | Sources |
|---|---|
| Registration, matching, dispatch, server lifetime | `router.go`, `dispatch.go`, `route_prefix.go`, `route_list.go` |
| Context helpers and responses | `context.go`, `respond.go`, `errors.go` |
| Middleware and response-controller ownership | `middleware.go` |
| Proxy trust, security, readiness | `proxy.go`, `security.go`, `readiness.go` |
| Pooling and process metrics | `performance.go` |
| Gin performance acceptance | `tools/benchmark-target.py`, `docs/performance-target.md` |
| Builder and shortcuts | `builder.go`, `shortcuts.go` |
| Schema and validation | `base.go`, `validator.go`, `constraints.go` |
| Contracts and TypeScript | `codegen.go`, `contract_schema.go`, `contract_source.go`, `typescript.go` |
| Modules and dependency injection | `module.go` |
| Prometheus/OpenTelemetry integration | `telemetry/telemetry.go` |
| CLI entry and orchestration | `cmd/routix/main.go`, `cmd/routix/commands/` |
| Generated applications | `cmd/routix/generators/`; inspect emitted strings as well as generator code |
| Examples and HTTP test helpers | `example/`, `examples/`, `test/test.go` |

## Working rules

- Preserve public signatures, exported field types, handler semantics, HTTP output, and context lifetime unless an explicit API change is requested.
- Do not add code comments for this workstream. Explain behavior in documentation and meaningful tests.
- Test files remain outside commits unless the user explicitly changes that policy. Markdown documentation, including README and this guide, is included when committing requested changes. Do not silently discard local regression tests or documentation to obtain a clean worktree.
- Use `Improvements` for commits when committing has been requested. Editing alone does not authorize publishing or merging.
- Inspect `git status` and existing changes before editing. Work may include local-only tests and README changes.
- Do not optimize by caching handler responses, skipping middleware, retaining request data between requests, weakening cleanup, or changing public maps to private lazy substitutes.
- Avoid `unsafe` layout dependencies. Any lookup index must be bounded and preserve the general matcher's behavior.
- Verify the installed CLI/application source revision. `@latest`, `main`, and the CLI's static version string can identify different code.

## Invariants to preserve

- Registration is mutex-protected. Requests use an immutable atomically published routing snapshot. Every registration/configuration update that changes dispatch must invalidate it.
- Static > parameter > wildcard priority includes backtracking, slash behavior, explicit HEAD precedence, GET fallback, and 404/405 with sorted Allow.
- `Context.Params` is a normal mutable public map. Mutating, deleting, replacing, and adding entries must remain observable during the handler. The internally owned map is emptied after the request, even if the public field was replaced.
- Pooled contexts must release public request/writer/body/query fields and private request data. No stale identity, proxy result, error, or handler may reach a later request.
- Middleware wraps request execution before automatic JSON decoding. Authentication and timeout ordering must continue to cover body reading correctly.
- A timeout worker must not access a pooled parent context or a closed network writer after the response ends. Nested deadlines use the earliest deadline. Context cancellation does not forcibly stop arbitrary handler code.
- Return errors through middleware. Do not replace a response that has already begun. JSON encoding failures must not commit a success status prematurely.
- Cache must preserve identity/private-response exclusions, including mTLS, auth/cookies, response policy, and appropriate variation.
- Changes to shared metrics, caches, snapshots, modules, or goroutine ownership require race/concurrency verification.

## Change workflow

1. Trace the public entry point to the code that owns the state. Record the concrete behavior to preserve or fix.
2. Add a focused regression or independent reference/oracle test. Check generated applications as applications, not only generator strings.
3. Make the smallest justified change. Document intentional limits and unresolved cases.
4. Run the relevant checks from [verification](docs/verification.md). Run full race tests after routing/lifecycle changes. Fuzz matching changes and inspect API compatibility.
5. For performance changes, freeze sources before measuring. Compare equivalent status/headers/body, route counts, and allocations; include before/after and competitors. Never select only favorable repetitions.
6. Before any requested commit, inspect staged paths and diff. Exclude test files under the current policy; include the requested documentation. Confirm the committed production tree matches the tested tree.

## Updating this guide

Keep source paths and commands portable. Do not embed developer-specific absolute paths, machine credentials, temporary workspaces, or links to unavailable local reports. Keep measured claims in the performance document with their date, versions, workload, and uncertainty. Update the architecture document when state ownership or dispatch changes.
