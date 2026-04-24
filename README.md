# Routix

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![Release](https://img.shields.io/github/v/release/ramusaaa/routix?color=00ADD8)](https://github.com/ramusaaa/routix/releases)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/ramusaaa/routix)](https://goreportcard.com/report/github.com/ramusaaa/routix)

A fast HTTP framework for Go with a Laravel-inspired CLI. Build APIs and web services without boilerplate.

```go
package main

import "github.com/ramusaaa/routix"

func main() {
    r := routix.New()

    r.GET("/", func(c *routix.Context) error {
        return c.JSON(200, map[string]any{"message": "hello"})
    })

    r.Start(":8080")
}
```

---

## Installation

### CLI tool

**Linux / macOS**
```bash
curl -fsSL https://raw.githubusercontent.com/ramusaaa/routix/main/install.sh | bash
```

**Windows (PowerShell)**
```powershell
irm https://raw.githubusercontent.com/ramusaaa/routix/main/install.ps1 | iex
```

**Manual (any platform with Go installed)**
```bash
go install github.com/ramusaaa/routix/cmd/routix@latest
```

> Requires Go 1.21 or newer. Download at [go.dev/dl](https://go.dev/dl/).

### Library only

```bash
go get github.com/ramusaaa/routix@latest
```

---

## Getting started

```bash
routix new my-api
cd my-api
routix serve
```

`routix new` walks you through an interactive setup — choose a template, database, auth, Docker, and more. When done, your project is ready to run.

---

## Routing

```go
r := routix.New()

r.GET("/users",      listUsers)
r.POST("/users",     createUser)
r.GET("/users/:id",  getUser)
r.PUT("/users/:id",  updateUser)
r.DELETE("/users/:id", deleteUser)

// Wildcard
r.GET("/files/*", func(c *routix.Context) error {
    path := c.Params["*"]
    return c.JSON(200, map[string]any{"path": path})
})

r.Start(":8080")
```

### Route groups

```go
api := r.Group("/api/v1")
api.Use(authMiddleware)
api.GET("/users", listUsers)
api.POST("/users", createUser)

// Sub-groups
admin := api.Group("/admin")
admin.Use(adminOnly)
admin.DELETE("/users/:id", deleteUser)
```

### RESTful resources

```go
r.Resource("/articles", routix.ResourceController{
    Index:  listArticles,
    Create: createArticle,
    Show:   getArticle,
    Update: updateArticle,
    Delete: deleteArticle,
})
// Registers: GET /articles, POST /articles,
//            GET /articles/:id, PUT /articles/:id, DELETE /articles/:id
```

---

## Context

### Parameters and query

```go
r.GET("/users/:id", func(c *routix.Context) error {
    id     := c.Params["id"]         // URL param
    page   := c.Query["page"]        // ?page=2
    search := c.QueryParamDefault("q", "")

    idInt, err := c.ParamInt("id")   // parsed as int
    _ = idInt
    return c.JSON(200, map[string]any{"id": id, "page": page})
})
```

### Request body

```go
r.POST("/users", func(c *routix.Context) error {
    // Pre-parsed from application/json body
    name, _ := c.Body["name"].(string)

    // Or decode into a struct
    var req struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }
    if err := c.ParseJSON(&req); err != nil {
        return c.BadRequest(err.Error())
    }
    return c.Created(map[string]any{"name": req.Name})
})
```

### Passing values between middleware

```go
r.Use(func(next routix.Handler) routix.Handler {
    return func(c *routix.Context) error {
        c.Set("user_id", 42)
        return next(c)
    }
})

r.GET("/me", func(c *routix.Context) error {
    userID := c.MustGet("user_id").(int)
    return c.JSON(200, map[string]any{"id": userID})
})
```

---

## Responses

```go
// Status codes
c.JSON(200, data)
c.Created(data)      // 201
c.Accepted(data)     // 202
c.NoContent()        // 204

// Errors
c.BadRequest("validation failed")    // 400
c.Unauthorized("")                   // 401 — default message
c.Forbidden("")                      // 403
c.NotFound("user not found")         // 404

// Other formats
c.HTML(200, "<h1>Hello</h1>")
c.String(200, "plain text")
c.Redirect(302, "/login")

// Wrapped success/error envelope (includes timestamp)
c.Success(data)
c.Paginated(data, page, totalPages)

// HTTP cache headers
c.Cache(1 * time.Hour)  // Cache-Control: public, max-age=3600
```

---

## Middleware

### Built-in

```go
r.Use(
    routix.Logger(),           // method, path, status, latency
    routix.Recovery(),         // recover from panics
    routix.CORS(),             // Access-Control-Allow-* headers
    routix.Compress(),         // gzip for clients that accept it
    routix.RateLimit(100, time.Minute),
    routix.Timeout(30 * time.Second),
)
```

### Authentication

```go
validate := func(token string) bool {
    // verify your JWT here
    return token != ""
}

protected := r.Group("/api")
protected.Use(routix.Auth(validate))
protected.GET("/profile", getProfile)
```

### Writing middleware

```go
func RequireAdmin(next routix.Handler) routix.Handler {
    return func(c *routix.Context) error {
        role, _ := c.Get("role")
        if role != "admin" {
            return c.Forbidden("")
        }
        return next(c)
    }
}
```

---

## API builder

For production services, `NewAPI()` gives you a fluent builder:

```go
app := routix.NewAPI().
    Prod().                     // Recovery + Logger + Compress
    CORS().
    JSON().                     // sets Content-Type: application/json globally
    RateLimit(1000, "1m").
    Health("/health").          // GET /health
    Metrics("/metrics")         // GET /metrics

app.GET("/users", listUsers)
app.Start(":8080")
```

Shortcuts for common setups:

```go
routix.QuickAPI()   // Prod + CORS + JSON + Health + Metrics
routix.MicroAPI()   // JSON + Health + RateLimit
routix.DevAPI()     // Dev + CORS + JSON + Health
```

---

## CLI reference

### Create a project

```
routix new <name>
```

Starts an interactive wizard. Choose:

| Option | Choices |
|--------|---------|
| Template | API, Full-stack, Microservice, Minimal |
| Database | PostgreSQL, MySQL, SQLite |
| Auth | JWT + bcrypt |
| Caching | Redis |
| Extras | WebSocket, job queue, Docker, Swagger, tests, CORS, rate limiting |

### Code generation

```bash
routix make:controller UserController
routix make:controller UserController --resource   # CRUD scaffold
routix make:model      User
routix make:model      User --migration            # model + migration
routix make:middleware Auth
routix make:migration  create_users_table
routix make:seeder     UserSeeder
routix make:service    EmailService
routix make:request    CreateUserRequest
routix make:resource   UserResource
routix make:test       UserTest
routix make:module     Auth
routix make:job        SendEmailJob
```

### Database

```bash
routix migrate              # run pending migrations
routix migrate rollback     # roll back the last batch
routix migrate reset        # roll back all
routix migrate fresh        # reset + re-run
routix migrate status       # show migration table
routix seed                 # run all seeders
```

### Development

```bash
routix serve                   # start with hot reload
routix serve --port=3000       # custom port
routix route:list              # print registered routes
routix test                    # run tests
routix test unit
routix test integration
routix test --coverage
```

---

## Project structure

```
my-api/
├── app/
│   ├── controllers/       HTTP handlers
│   ├── middleware/        custom middleware
│   ├── models/            GORM models
│   ├── requests/          request validators
│   ├── resources/         API response transformers
│   └── services/          business logic
├── config/                environment-based config
├── database/
│   ├── migrations/        schema migrations
│   └── seeders/           seed data
├── routes/                route registration
├── storage/
│   ├── cache/
│   └── logs/
├── tests/
│   ├── unit/
│   └── integration/
├── .env
├── .env.example
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── main.go
```

---

## Docker

Generated projects include a multi-stage Dockerfile and compose files:

```bash
# development
docker compose -f docker-compose.dev.yml up

# production
docker compose up -d
```

---

## Contributing

1. Fork the repo
2. Create a branch: `git checkout -b feature/your-feature`
3. Make your changes and add tests
4. Run `go test ./...` — all tests must pass
5. Open a pull request

Bug reports and feature requests are welcome via [GitHub Issues](https://github.com/ramusaaa/routix/issues).

---

## License

MIT — see [LICENSE](LICENSE).
