---
name: go-backend
description: Go backend patterns for FinGuard. Covers project layout, HTTP handler conventions, Chi routing, middleware, error handling, configuration, and dependency injection. Use when writing Go code, creating handlers, adding routes, or modifying backend logic.
---

# Go Backend Patterns

## Project Layout

```
cmd/finguard/main.go     -- Entry point only: wire dependencies, start server, handle shutdown
internal/<domain>/       -- Private packages (auth, server, store, config, plugin, etc.)
pkg/<domain>/            -- Public API types and interfaces (api, event, plugin)
migrations/              -- SQL migration files (embedded at build time)
protos/                  -- Protocol buffer definitions
```

- Business logic goes in `internal/`. Never in `cmd/`.
- Types shared with the frontend or external consumers go in `pkg/api/types.go`.
- One package per bounded domain. Keep packages focused.

## Server & Dependency Injection

All dependencies are fields on the `Server` struct, injected via the constructor. No global state.

```go
type Server struct {
    cfg        *config.Config
    store      store.Store
    auth       *auth.Manager
    logger     *slog.Logger
    // ... other dependencies
}

func New(cfg *config.Config, st store.Store, am *auth.Manager, logger *slog.Logger) *Server {
    s := &Server{cfg: cfg, store: st, auth: am, logger: logger}
    s.router = s.routes()
    return s
}
```

## Handler Pattern

Handlers are methods on `*Server`. Follow this structure:

```go
func (s *Server) handleCreateThing(w http.ResponseWriter, r *http.Request) {
    // 1. Parse input
    var req struct {
        Name string `json:"name"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
        return
    }

    // 2. Validate
    if req.Name == "" {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
        return
    }

    // 3. Execute business logic via store/service
    thing := &models.Thing{Name: req.Name}
    if err := s.store.CreateThing(r.Context(), thing); err != nil {
        s.logger.Error("failed to create thing", "error", err)
        writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create thing"})
        return
    }

    // 4. Respond
    writeJSON(w, http.StatusCreated, thing)
}
```

Key rules:
- Decode request bodies into anonymous structs with JSON tags.
- Use `chi.URLParam(r, "paramName")` for path parameters.
- Return early on errors. Log the real error with `s.logger.Error()`, send a safe message to the client.
- Use `writeJSON()` for all responses. Never write raw bytes.
- For GET list endpoints, return empty slice `[]` not `null`: `if items == nil { items = []*T{} }`.

## Response Helpers

```go
func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(v)
}
```

Error responses use `map[string]string{"error": "message"}`. Success responses return the resource or a wrapper like `map[string]any{"projects": projects}`.

## Routing

Routes are registered in `Server.routes()` using Chi.

```go
func (s *Server) routes() chi.Router {
    r := chi.NewRouter()
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
    r.Use(middleware.Recoverer)
    r.Use(middleware.Compress(5))

    // Unauthenticated health/auth endpoints at root
    r.Get("/healthz", s.handleHealthz)

    // All API routes under /api/v1 with auth middleware
    r.Route("/api/v1", func(r chi.Router) {
        r.Use(s.auth.Middleware)

        r.Post("/things", s.handleCreateThing)
        r.Get("/things", s.handleListThings)
        r.Route("/things/{thingID}", func(r chi.Router) {
            r.Get("/", s.handleGetThing)
            r.Put("/", s.handleUpdateThing)
            r.Delete("/", s.handleDeleteThing)
        })
    })
    return r
}
```

Rules:
- All API routes nest under `/api/v1`.
- Sub-resources nest with `r.Route()`.
- Auth middleware applied to the `/api/v1` group.
- Health/readiness probes are at root (`/healthz`, `/readyz`).
- Handler file naming: group related handlers in a file named after the resource (e.g., `projects.go`, `budgets.go`).

## Middleware Order

1. `middleware.RequestID` -- assigns request ID
2. `middleware.RealIP` -- extracts real client IP
3. `middleware.Recoverer` -- catches panics
4. `middleware.Compress(5)` -- gzip compression
5. `s.auth.Middleware` -- OIDC session validation (on `/api/v1` only)
6. RBAC middleware per-route as needed: `auth.RequireProjectRole(models.RoleEditor)`

## Error Handling

- Use `log/slog` (structured logging). The logger is a field on `Server`.
- Log internal details: `s.logger.Error("failed to create project", "error", err)`
- Return safe messages to client: `{"error": "failed to create project"}`
- Never expose stack traces, SQL errors, or internal paths to the client.

## Configuration

All config via environment variables with `FINGUARD_` prefix. Defined in `internal/config/config.go`.

```go
type Config struct {
    HTTPAddr    string
    DatabaseDSN string
    DevMode     bool
    // ...
}

func Load() *Config {
    return &Config{
        HTTPAddr:    envOr("FINGUARD_ADDR", ":8080"),
        DatabaseDSN: envOr("FINGUARD_DB_DSN", "sqlite:///tmp/finguard.db"),
        DevMode:     envBool("FINGUARD_DEV_MODE"),
    }
}
```

When adding a new config value:
1. Add the field to the `Config` struct.
2. Read it in `Load()` using `envOr()`, `envBool()`, or `envSlice()`.
3. Document the env var name and default.

## Model Conventions

Models live in `internal/models/`. Use both `json` and `db` struct tags:

```go
type Thing struct {
    ID        string    `json:"id" db:"id"`
    Name      string    `json:"name" db:"name"`
    CreatedAt time.Time `json:"createdAt" db:"created_at"`
    UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}
```

- JSON tags are camelCase.
- DB tags are snake_case matching the SQL column.
- Use `omitempty` for optional/nullable fields.
- Use pointer types (`*time.Time`, `*string`) for nullable DB columns.
- For `json.RawMessage` fields, add `swaggertype:"object"` so swag can parse them:
  ```go
  Config json.RawMessage `json:"config" db:"config_json" swaggertype:"object"`
  ```

## Swagger / Swag Annotations

We use [swaggo/swag](https://github.com/swaggo/swag) to auto-generate OpenAPI docs from Go comments. Run `make swagger` to regenerate (also runs automatically before `make build`).

### Top-Level API Metadata

Defined once in `cmd/finguard/main.go` above the `main()` function:

```go
// @title           FinGuard API
// @version         1.0
// @description     FinGuard cloud cost management platform API.
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey  SessionAuth
// @in                          cookie
// @name                        finguard_session
func main() {
```

The generated docs package is imported as a blank import in `main.go`:

```go
import _ "github.com/inelson/finguard/docs/swagger"
```

### Annotating Handlers

Every handler that serves an API endpoint **must** have swag annotations directly above the function. Follow this template:

```go
// @Summary      Short one-line description
// @Description  Longer explanation of what the endpoint does
// @Tags         ResourceGroup
// @Accept       json          (include only if handler reads a request body)
// @Produce      json
// @Param        paramName  path/query/body  type  required  "description"
// @Success      200  {object}  ResponseType
// @Failure      400  {object}  object{error=string}
// @Failure      500  {object}  object{error=string}
// @Security     SessionAuth   (include for authenticated endpoints)
// @Router       /route [method]
func (s *Server) handleDoThing(w http.ResponseWriter, r *http.Request) {
```

### Annotation Rules

1. **@Tags** -- group by resource: `Projects`, `CostSources`, `Members`, `Costs`, `Cluster`, `Health`, `Auth`, `Plugins`.
2. **@Router** -- path is relative to `@BasePath` (`/api/v1`). Use `{paramName}` for path params.
3. **@Param for path params**: `// @Param  projectID  path  string  true  "Project ID"`
4. **@Param for query params**: `// @Param  limit  query  int  false  "Max results"  default(50)`
5. **@Param for request body**: Use inline object syntax for simple bodies:
   `// @Param  body  body  object{name=string,description=string}  true  "Fields"`
   Or reference a model: `// @Param  body  body  models.Thing  true  "The thing"`
6. **@Success / @Failure** -- use `{object}` for JSON responses. For collections wrapped in a key, use inline syntax:
   `// @Success  200  {object}  object{projects=[]models.Project}`
7. **@Security SessionAuth** -- include on every endpoint under `/api/v1` (except health probes).
8. Endpoints outside `/api/v1` (like `/healthz`) omit `@Security` and use the full path in `@Router`.

### Generated Files

`make swagger` produces three files in `docs/swagger/`:
- `docs.go` -- Go source registering the spec with swag (imported via blank import)
- `swagger.json` -- OpenAPI 2.0 JSON spec
- `swagger.yaml` -- OpenAPI 2.0 YAML spec

These are **generated files**. Do not edit them by hand. Always regenerate with `make swagger`.

### Swagger UI

Served at `/swagger/index.html` via `httpSwagger.WrapHandler` in `server.go`. The Vite dev server proxies `/swagger` to the Go backend.

### Common Pitfalls

- **`json.RawMessage` fields** cause swag parse errors. Always add `swaggertype:"object"` to the struct tag.
- **The swag library version must match the CLI version.** Both should be v1.16.4+. If you see `unknown field LeftDelim` errors, run `go get github.com/swaggo/swag@v1.16.4`.
- **The `--exclude` flag** in the Makefile prevents swag from scanning vendored/third-party dirs (`headlamp`, `opencost`).
