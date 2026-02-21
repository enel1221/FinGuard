---
name: rest-api-design
description: REST API design standards for FinGuard. Covers URL conventions, HTTP methods, JSON response format, error handling, pagination, status codes, and OpenAPI workflow via swag annotations. Use when designing new API endpoints, adding routes, defining request/response types, or reviewing API contracts.
---

# REST API Design Standards

## OpenAPI Workflow

The OpenAPI spec is **auto-generated** from Go handler annotations using [swaggo/swag](https://github.com/swaggo/swag). There is no hand-written spec file to maintain.

When adding or changing an endpoint:

1. Write the Go handler with swag annotations (see `go-backend` skill for annotation format).
2. Run `make swagger` to regenerate `docs/swagger/swagger.json` and `swagger.yaml`.
3. Add matching TypeScript types in `web/frontend/src/lib/api.ts`.
4. Verify at `http://localhost:8080/swagger/index.html` during development.

The generated spec lives in `docs/swagger/` and should be committed so CI and other tools can consume it without running swag.

## URL Conventions

```
/api/v1/{resource}                          -- collection
/api/v1/{resource}/{id}                     -- single item
/api/v1/{resource}/{id}/{sub-resource}      -- nested collection
/api/v1/{resource}/{id}/{sub-resource}/{id} -- nested item
```

Rules:
- Path-based versioning: `/api/v1`. Breaking changes require a new version.
- Plural nouns for resources: `/projects`, `/sources`, `/members`.
- Path parameter names use camelCase: `{projectID}`, `{sourceID}`.
- No verbs in URLs. Use HTTP methods to express actions.
- No trailing slashes.

Existing examples:
```
GET    /api/v1/projects
POST   /api/v1/projects
GET    /api/v1/projects/{projectID}
PUT    /api/v1/projects/{projectID}
DELETE /api/v1/projects/{projectID}
POST   /api/v1/projects/{projectID}/sources
GET    /api/v1/projects/{projectID}/sources
GET    /api/v1/projects/{projectID}/sources/{sourceID}
DELETE /api/v1/projects/{projectID}/sources/{sourceID}
GET    /api/v1/projects/{projectID}/costs
```

## HTTP Methods

| Method | Purpose | Request Body | Response |
|--------|---------|-------------|----------|
| GET    | Read / List | None | Resource or list |
| POST   | Create | New resource fields | Created resource |
| PUT    | Full update | Complete resource | Updated resource |
| PATCH  | Partial update | Changed fields only | Updated resource |
| DELETE | Remove | None | Status confirmation |

## JSON Response Format

### Success -- Single Resource

```json
{
  "id": "abc-123",
  "name": "Production",
  "createdAt": "2026-01-15T10:30:00Z"
}
```

### Success -- Collection

```json
{
  "projects": [
    {"id": "abc-123", "name": "Production"}
  ]
}
```

The collection wrapper key matches the resource name (e.g., `"projects"`, `"sources"`, `"members"`).

For paginated responses:

```json
{
  "projects": [...],
  "total": 42
}
```

### Error

```json
{
  "error": "human-readable message"
}
```

Rules:
- All JSON field names are **camelCase**.
- Timestamps are ISO 8601 / RFC 3339 in UTC: `"2026-01-15T10:30:00Z"`.
- Empty collections return `[]`, never `null`.
- Omit optional fields when empty using `omitempty`.

## Status Codes

| Code | When |
|------|------|
| 200 | Successful GET, PUT, PATCH |
| 201 | Successful POST (resource created) |
| 204 | Successful DELETE (no body) -- currently we return `{"status": "deleted"}` with 200; either is acceptable |
| 400 | Malformed request, validation failure |
| 401 | Unauthenticated (no/invalid session) |
| 403 | Authenticated but insufficient role/permissions |
| 404 | Resource not found |
| 409 | Conflict (duplicate name, version mismatch) |
| 500 | Unexpected server error |

## Pagination

For list endpoints that may grow large:

```
GET /api/v1/projects?limit=20&offset=0
```

- `limit` -- max items to return (default: 50, max: 200)
- `offset` -- number of items to skip

Response includes `total` count for UI pagination controls.

## Request Validation

Validate in the handler before calling the store:

```go
if req.Name == "" {
    writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
    return
}
```

Validation rules:
- Required fields checked explicitly.
- Return 400 with a clear error message.
- Validate one field at a time, return on first error (keep it simple for now).

## Shared Types

Go types in `pkg/api/types.go` define the API contract. Frontend mirrors these in `web/frontend/src/lib/api.ts`.

When adding a new endpoint:
1. Define/update Go types in `pkg/api/types.go` (for shared response types) or use anonymous structs in the handler for simple request types.
2. Define the matching TypeScript interface in `web/frontend/src/lib/api.ts`.
3. Keep field names identical (both camelCase in JSON).

## Non-REST Endpoints

- WebSocket: `GET /api/v1/stream` -- event streaming, not REST.
- Health probes: `GET /healthz`, `GET /readyz` -- outside `/api/v1`, no auth.
- Auth: `GET /login`, `GET /callback`, `GET /logout` -- OIDC flow, outside `/api/v1`.
- Swagger UI: `GET /swagger/*` -- auto-generated API docs, outside `/api/v1`, no auth.

## Endpoint Checklist

When adding a new endpoint, verify:

- [ ] Handler has complete swag annotations (`@Summary`, `@Tags`, `@Param`, `@Success`, `@Failure`, `@Router`, `@Security`)
- [ ] `make swagger` passes without errors
- [ ] Route registered in `server.go` under the correct resource group
- [ ] TypeScript types updated in `web/frontend/src/lib/api.ts`
- [ ] URL follows conventions above (plural nouns, no verbs, correct nesting)
- [ ] Response format matches the JSON standards in this doc (camelCase, `[]` not `null`, error shape)
