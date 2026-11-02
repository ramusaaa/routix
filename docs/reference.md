# API and CLI reference

This is a source-oriented guide, not an exhaustive generated symbol listing. Use `go doc` for the exact signatures in your checkout.

## Router and handlers

`Handler` is `func(*Context) error`. `Middleware` is `func(Handler) Handler`. `New()` creates a bare `*Router`; `NewAPI()` creates a builder with process performance monitoring already installed.

Use `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS`, or `Handle` to register methods. `Group` composes path prefixes and scoped middleware. `Resource` registers REST-style handlers from `ResourceController`. `Routes()` returns a copied registration snapshot; `WriteRoutes(io.Writer)` exports sorted JSON.

`NewFastRouter()` wraps the same Router implementation; its name does not select an unrelated matching algorithm. `DisablePooling()` changes context lifetime for debugging.

### Route replacement

For one method, static segments plus dynamic-segment positions identify a shape; parameter names are metadata. `/users/:id` followed by `/users/:name` selects the second handler and parameter name. `Routes()` reports the latest path once. Updating a published route invalidates the dispatch snapshot. Per-route middleware is replaced with the latest registration's middleware. Other methods remain independent; this is a documented replacement rule, not conflict rejection.

### Middleware and errors

```go
func RequireRole(next routix.Handler) routix.Handler {
    return func(c *routix.Context) error {
        role, _ := c.Get("role")
        if role != "admin" {
            return routix.Forbidden("admin role required", nil)
        }
        return next(c)
    }
}
```

Use framework error constructors when you want outer middleware to receive and potentially transform an error. Context response helpers such as `c.Forbidden` write responses immediately; returning their result is not the same as returning a deferred framework error. Internal errors are rendered as generic HTTP messages rather than leaking raw details.

### Context helpers

| Task | API |
|---|---|
| Path values | `Param`, `ParamInt`, `ParamInt64`, public `Params` |
| Query | `QueryParam`, `QueryParamDefault`, integer variants, public `Query` |
| JSON body | `ParseJSON`, public `Body` for decoded objects |
| Scoped state | `Set`, `Get`, `MustGet` |
| Identity | `ClientIP`, `UserAgent`, `IsSecure`, `RoutePath` |
| Output | `JSON`, `FastJSON`, `String`, `HTML`, `Redirect`, `NoContent` |
| Envelopes | `Success`, `Paginated`, `Respond`, `RespondPaginated` |
| Response state | `Status`, `ResponseWritten`, `Response`, `Writer` |

Response helpers have distinct wire policies. Standard `JSON` uses encoding/json with HTML escaping disabled and a trailing newline. `FastJSON` has specialized primitive paths and falls back to JSON for other values. Preserve existing behavior when optimizing.

## Production configuration

### Proxies

```go
if err := r.SetTrustedProxies([]string{"10.0.0.0/8", "2001:db8::/32"}); err != nil {
    return err
}
```

Use the actual deployment's proxy addresses; the example networks are illustrative. Invalid IP/CIDR configuration returns an error. With an untrusted socket peer, forwarded headers do not override the client identity. Trusted X-Forwarded-For is traversed from the proxy side toward the first untrusted address. `X-Forwarded-Proto` is accepted only under the configured trust policy.

### Headers and CSRF

`SecureHeaders()` supplies default CSP, referrer, permissions, frame, MIME, and secure-request HSTS headers. Supplying an explicit `SecurityHeadersConfig` replaces that configuration rather than merging every nonzero field with defaults.

```go
r.Use(routix.CSRF(routix.CSRFConfig{
    Key: csrfSecret,
    SessionID: func(c *routix.Context) string {
        value, _ := c.Get("session_id")
        id, _ := value.(string)
        return id
    },
}))
```

`csrfSecret` must be at least 32 bytes. The default random key is instance-local; provide a stable shared key for multiple processes/restarts. Default secure cookies use the `__Host-` prefix. `AllowInsecureCookies` is an explicit local-development setting. Establish session identity before the CSRF layer when using session binding. Unsafe requests supply the signed cookie token through `X-CSRF-Token`; origin/fetch metadata is also checked. This middleware does not implement application sessions.

### Readiness

```go
r.GET("/ready", routix.Readiness(2*time.Second, map[string]routix.ReadinessCheck{
    "database": sqlDB.PingContext,
}))
```

`sqlDB` is your configured `*sql.DB`. Checks return errors and honor their contexts. Success returns 200; failure or deadline returns 503. The response is marked no-store. A liveness endpoint should be separate from dependency readiness.

### Telemetry

```go
observability, err := telemetry.New(telemetry.Config{})
if err != nil {
    return err
}
observability.Attach(r, "/metrics")
```

Import `github.com/ramusaaa/routix/telemetry`. Supply a registry, tracer provider, and propagator when needed. `Attach` installs middleware and the metrics route. Counters/histograms use method, registered route pattern, and status; unmatched routes use a fixed label. Configure OpenTelemetry export/shutdown in your application. Do not also expose builder JSON metrics on the same path.

### Server lifecycle

`Router.Start` and builder `Start` use graceful shutdown. `APIBuilder.StartTLS(addr, certFile, keyFile)` uses the same shutdown flow for HTTPS. For custom timeouts or lifecycle ownership, pass the Router as `http.Server.Handler`. Custom server startup does not automatically participate in CLI route export.

## Contracts and validation

Prefer `OpenAPI() (map[string]any, error)` and `TypeScriptTypes() (string, error)`. Legacy `GenerateOpenAPISpec` / `GenerateTypeScriptTypes` panic when conversion fails.

`RouteDefinition` accepts method, path, summary, operation ID, status, and either request/response schemas or request/response Go values. `RequestType` and `RequestBody` are alternatives, as are `ResponseType` and `Response`. Explicit metadata must match the bound router's method/path registrations. Missing metadata still produces route entries without invented DTOs. Use `AddType` for additional named schemas.

`ParseSchemaFromFile(filename, name)` resolves source types into a schema. `ParseStructFromFile` supports reflect-oriented generation but cannot represent every Go type. Unsupported contracts should return conversion errors instead of broad `any` guesses. Recursive references and JSON field behavior are modeled; custom marshaling needs special care.

`ValidateStruct` and `Validate` middleware implement struct-tag validation; fluent schemas provide a separate path. Numeric min/max/len options are parsed and cached before field evaluation. Malformed, nonfinite, negative length, and fractional length constraints produce validation errors rather than silently becoming zero. Numeric fields still permit finite negative/fractional min/max bounds. Supported runtime rules and schema constraints currently differ in places. Check [limitations](limitations.md) before relying on unknown rules, comma-containing regex constraints, numeric enums, or nested traversal. Contract validation means the OpenAPI document is valid; it does not establish that every response produced by arbitrary handler code satisfies it.

## Modules and services

Modules combine routes, controllers, middleware, submodules, services, and imports. `Module.Register` prefixes and scopes registrations; controller registration is isolated before its routes are attached. Cyclic/nil module traversal is rejected.

Dependency injection uses reflect-based service registration/resolution. Treat constructor execution and error handling as observable behavior. Inspect `module.go` for the precise `ServiceContainer` contract and known lifecycle limits before adding a resource-owning factory.

## CLI and generated projects

### Source revision

CLI generators currently include fallback version logic and a static version string that can lag `main`. Inspect the generated `go.mod` before relying on recently added APIs. Upgrade the application dependency to a reviewed compatible revision, run `go mod tidy`, and build the result. A new release/tag is a separate action; documentation updates do not publish one.

### Hot reload and route listing

`routix serve --host=localhost --port=8080` sets development host/port environment variables and watches supported source/config changes. It builds each replacement into a temporary location, retains the current service on build failure, and stops managed processes on shutdown.

`routix route:list` and `routix route list` build the current main package and run it with `ROUTIX_ROUTE_LIST_FILE` pointing to a temporary output file. Routix startup exports registration data before listening. All initialization before that call still executes; use the appropriate application environment, including database access. The command has a two-minute context deadline and requires a compatible application framework version.

### Build and tests

`routix test` delegates to `go test ./...`; unit/integration scopes use `./tests/unit/...` and `./tests/integration/...`. `--coverage` writes coverage.out. Failures propagate for test/build/serve/route/database commands.

The current `routix build` requests output `app`. Generated projects also contain an `app/` directory, so Go can place the executable inside it while the CLI prints a misleading path. Until that output-path issue is fixed, use:

```sh
mkdir -p bin
go build -ldflags='-s -w' -o ./bin/server .
```

### Auth and migrations

Auth generation includes bcrypt password hashing, database persistence, bearer-token parsing, token expiry/revocation, refresh, and secret checks. Inspect the emitted configuration and establish a strong JWT secret before running. These files are application scaffolding; application authorization, session policy, deployment secrets, and abuse controls remain your responsibility.

Migrations support `migrate`, `status`, `rollback`, `reset`, `fresh`, and explicit dirty-state repair. Review the generated migration definitions and the configured database before destructive commands. A repair action records applied/pending state; it cannot undo a partially applied nontransactional DDL operation. Checksum/history and driver-specific locking are implemented in the generated runner, not the core router.

Scaffold options for jobs, Redis, Swagger, WebSockets, and CRUD do not imply complete production integrations. Generator naming/overwrite and Docker configuration have unresolved findings listed in [limitations](limitations.md). Inspect generated files before using them in an existing project or deployment.
