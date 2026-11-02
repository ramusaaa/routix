# Routix

A Go HTTP framework with `net/http` integration, error-returning handlers, middleware, typed API contracts, and a project CLI.

[API reference](https://pkg.go.dev/github.com/ramusaaa/routix) · [Agent guide](AGENTS.md) · [Architecture](docs/architecture.md) · [Usage reference](docs/reference.md) · [Verification](docs/verification.md) · [Performance](docs/performance.md)

## Version and installation

Requires Go **1.25 or newer**, as declared by `go.mod`. Recent verification and the reported benchmarks use **Go 1.26.10**; benchmark results should not be extrapolated across Go versions.

This README describes the source on `main`. A release tag or `@latest` may contain older behavior. To use this revision, resolve `main` and retain the pseudo-version recorded in `go.mod`:

```sh
go get github.com/ramusaaa/routix@main
go install github.com/ramusaaa/routix/cmd/routix@main
```

For reproducible deployments, replace `main` with a reviewed commit or release. Install the CLI and application dependency from compatible revisions. The CLI's printed version alone does not identify the installed source revision; inspect it with `go version -m /path/to/routix`.

## First service

```go
package main

import (
    "log"
    "net/http"

    "github.com/ramusaaa/routix"
)

func main() {
    r := routix.New()
    r.Use(routix.Recovery(), routix.SecureHeaders())
    r.GET("/users/:id", func(c *routix.Context) error {
        return c.JSON(http.StatusOK, map[string]any{"id": c.Param("id")})
    })
    if err := r.Start(":8080"); err != nil {
        log.Fatal(err)
    }
}
```

`*Router` implements `http.Handler`; use it with your own `http.Server` when you need explicit server configuration. `New()` does not install logging, authentication, CORS, rate limiting, or telemetry automatically.

## Routing and context

```go
api := r.Group("/api/v1")
api.GET("/users/:id", getUser)
api.POST("/users", createUser)
api.GET("/files/*", func(c *routix.Context) error {
    return c.JSON(200, map[string]any{"path": c.Param("*")})
})
```

Static segments take priority over parameters, then wildcards, with fallback when a more specific branch cannot finish matching. Parameters use `:name`; terminal wildcards use `*`, read through `c.Param("*")`. Explicit HEAD routes take priority; otherwise HEAD falls back to GET. A path available under another method returns 405 with a sorted `Allow` header. Routix does not automatically register OPTIONS handlers.

`Context.Params`, `Query`, and `Body` remain public maps. Query values retain the first value per key. JSON object bodies are decoded before the route handler, inside the middleware chain; malformed and oversized JSON are rejected. The default body limit is 1 MiB. Set `r.MaxBodyBytes` before serving. `ParseJSON` supports repeated decoding of the buffered body.

For the same HTTP method, registering the same normalized route shape replaces its handler and metadata. Parameter names do not make shapes distinct:

```go
r.GET("/users/:id", firstHandler)
r.GET("/users/:name", secondHandler)
```

The second registration is active, receives `Params["name"]`, and replaces the route-list entry with `/users/:name`. It does not create two dispatch candidates or raise a conflict error. Different HTTP methods remain separate. This replacement policy preserves the existing API; use unique shapes when replacement is unintended.

Contexts are pooled and valid only during the synchronous handler call. Copy data needed by background work; do not retain the context or mutate request-owned maps concurrently. `NewFastRouter().DisablePooling()` is available for debugging lifetime issues.

## Middleware and production services

```go
r.Use(
    routix.Recovery(),
    routix.Logger(),
    routix.SecureHeaders(),
    routix.RateLimit(100, time.Minute),
    routix.Timeout(5*time.Second),
)
```

`Middleware` wraps `Handler`; earlier registered middleware runs outside later middleware. Return errors so outer middleware can transform them. A response that has already started cannot be replaced by an error document.

| Capability | Entry point | Behavior to configure |
|---|---|---|
| Trusted proxies | `Router.SetTrustedProxies` | Explicit IP/CIDR allowlist; empty by default |
| Security headers | `SecureHeaders` | CSP/referrer/HSTS policy; HSTS on secure requests |
| CSRF | `CSRF(CSRFConfig{...})` | Shared secret of at least 32 bytes for multiple instances; session binding when needed |
| Readiness | `Readiness` / `APIBuilder.Ready` | Named dependency checks with a timeout; 200 or 503 |
| Compression | `Compress` | Negotiated gzip with streaming/controller support |
| Cache | `Cache` | Process-local response cache with private-response exclusions |
| Rate limiting | `RateLimit` | Process-local IP-based limiter; not a distributed quota |
| Metrics and tracing | `telemetry.New` / `Attach` | Prometheus registry and OpenTelemetry provider |
| TLS | `APIBuilder.StartTLS` | Certificate/key files and graceful shutdown |

Use `Context.ClientIP()` with the trusted-proxy configuration. The legacy `GetRealIP` helper has different trust semantics. `APIBuilder.Metrics` exposes JSON process metrics; `telemetry` exposes Prometheus/OpenMetrics. They are separate APIs. See [production configuration](docs/reference.md#production-configuration).

## OpenAPI and TypeScript

Contracts bind to registered routes. Add request/response metadata explicitly; handler signatures alone cannot reveal every DTO.

```go
type User struct {
    ID string `json:"id"`
}

contracts := routix.NewCodeGenerator().BindRouter(r)
contracts.AddRoute(routix.RouteDefinition{
    Method: "GET",
    Path: "/api/v1/users/:id",
    ResponseType: User{},
    StatusCode: 200,
})
document, err := contracts.OpenAPI()
if err != nil {
    return err
}
typescript, err := contracts.TypeScriptTypes()
if err != nil {
    return err
}
```

`OpenAPI()` emits and validates OpenAPI 3.0.3. `TypeScriptTypes()` derives types from the same schema model. Source-file schema parsing is available through `ParseSchemaFromFile`. Generation is not a substitute for runtime request validation. Prefer the error-returning APIs over legacy convenience methods that panic on invalid contracts. See [contracts](docs/reference.md#contracts-and-validation).

## CLI

```sh
routix new my-api
cd my-api
routix serve --host=localhost --port=8080
routix route:list
routix test --coverage
```

| Task | Commands |
|---|---|
| Generate code | `make:controller`, `make:model`, `make:middleware`, `make:service`, `make:request`, `make:resource`, `make:module`, `make:job`, `make:test` |
| Database scaffolding | `make:migration`, `make:seeder`, `make:model User --migration` |
| Migrations | `migrate`, `migrate status`, `migrate rollback`, `migrate reset`, `migrate fresh` |
| Dirty history repair | `migrate repair <id> --applied` or `--pending` |
| Seeding | `seed` |
| Test scopes | `test`, `test unit`, `test integration`, `test --coverage` |
| Build | `build`; explicit output alternative: `go build -o ./bin/server .` |

`make controller Name` is an alias for `make:controller Name`. Run `routix help` for the installed CLI's commands.

Hot reload builds a replacement before stopping the current process; failed builds leave the running version available. `route:list` builds and executes the current application's main package to export actual registrations through `Router.Start`, `APIBuilder.Start`, or `StartTLS`. Initialization code still runs, including database connections and other startup side effects. A server started directly with `http.ListenAndServe` does not use this export hook.

Auth scaffolding produces database-backed users and token lifecycle code. Migration scaffolding supports history, checksums, locking, rollback, and dirty-state repair. Database transactions and DDL guarantees depend on the selected driver. Generated CRUD and optional integrations include scaffolding that needs application implementation; selecting a wizard option is not proof of a complete service. See [CLI details and limitations](docs/reference.md#cli-and-generated-projects).

## Performance and verification

Performance depends on route distribution, middleware, payload, CPU count, and Go version. Routix has measured wins and remaining gaps against Gin, Echo, and Chi; there is no claim of universal superiority. See [methodology and results](docs/performance.md) for matched workloads, allocations, limits, and reproducibility. The [1.5× Gin throughput target](docs/performance-target.md) has a complete 48-cell acceptance contract and checker; it is not yet achieved.

```sh
go test ./...
go test -race ./...
go vet ./...
go build ./...
```

The published test set and local regression suite currently differ because regression tests were excluded from previous commits. A clean checkout contains a stale rate-limit expectation of 400 where the implementation returns 429. [Verification](docs/verification.md) documents this gap and the additional local checks; a passing local run must not be presented as proof that a clean checkout passes.

## Working on Routix

Start at [AGENTS.md](AGENTS.md). It maps request execution, routing snapshots, middleware lifetimes, code generation, and the verification requirements. [Known limitations](docs/limitations.md) records unresolved issues separately from implemented behavior.

MIT — see [LICENSE](LICENSE).
