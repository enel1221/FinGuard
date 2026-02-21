---
name: frontend-api
description: Frontend-backend communication patterns for FinGuard. Covers the api.ts client, type-safe API calls, data fetching patterns, error handling, and keeping TypeScript types in sync with Go. Use when making API calls from React, adding new endpoints to the frontend, or defining request/response types.
---

# Frontend-Backend Communication

## The API Client

All backend communication goes through the centralized client in `web/frontend/src/lib/api.ts`. Never use raw `fetch`.

```typescript
import { api } from '@/lib/api';

// Typed GET
const project = await api.get<Project>(`/projects/${id}`);

// Typed POST
const created = await api.post<Project>('/projects', { name: 'Prod' });

// Typed PUT
const updated = await api.put<Project>(`/projects/${id}`, { name: 'Staging' });

// Typed DELETE
await api.delete<void>(`/projects/${id}`);
```

Rules:
- Always specify the response type parameter: `api.get<T>(path)`.
- Paths are relative to `/api/v1` (the client prepends the base).
- Never construct full URLs. The Vite dev server proxies `/api` to the Go backend.

## Type Definitions

TypeScript interfaces live in `web/frontend/src/lib/api.ts` alongside the client. They must mirror the Go types in `pkg/api/types.go` and `internal/models/`.

### Adding a New Type

When adding a new API endpoint:

1. **Go side**: Define the response type in `pkg/api/types.go` (or use the model from `internal/models/`). JSON tags are camelCase.
2. **TypeScript side**: Add the matching interface in `api.ts`.

```go
// Go (pkg/api/types.go or internal/models/)
type Budget struct {
    ID            string   `json:"id"`
    ProjectID     string   `json:"projectId"`
    MonthlyLimit  float64  `json:"monthlyLimit"`
    WarnThreshold float64  `json:"warnThreshold"`
    CreatedAt     time.Time `json:"createdAt"`
    UpdatedAt     time.Time `json:"updatedAt"`
}
```

```typescript
// TypeScript (api.ts)
export interface Budget {
  id: string;
  projectId: string;
  monthlyLimit: number;
  warnThreshold: number;
  createdAt: string;
  updatedAt: string;
}
```

Field mapping rules:
- `string` -> `string`
- `int`, `float64` -> `number`
- `bool` -> `boolean`
- `time.Time` -> `string` (ISO 8601)
- `json.RawMessage` -> `Record<string, unknown>`
- `map[string]string` -> `Record<string, string>`
- `*T` / `omitempty` -> optional field with `?`
- Go slices -> TypeScript arrays
- Enum-like string types -> TypeScript union: `type RoleType = 'viewer' | 'editor' | 'admin'`

## Data Fetching Pattern

Standard pattern used in components:

```tsx
const [data, setData] = useState<Project | null>(null);
const [loading, setLoading] = useState(true);

useEffect(() => {
  api.get<Project>(`/projects/${projectId}`)
    .then(setData)
    .catch(() => {})
    .finally(() => setLoading(false));
}, [projectId]);
```

For multiple parallel requests:

```tsx
useEffect(() => {
  Promise.all([
    api.get<HealthResponse>('/health').catch(() => null),
    api.get<{ projects: Project[] }>('/projects').catch(() => ({ projects: [] })),
  ]).then(([health, projectsResp]) => {
    setHealth(health);
    setProjects(projectsResp?.projects ?? []);
  }).finally(() => setLoading(false));
}, []);
```

## Error Handling

The API client handles these automatically:
- **401 responses** redirect to `/login` (session expired).
- **Non-OK responses** throw an `Error` with the server's `error` field as the message.

In components, catch errors and show user-friendly feedback:

```tsx
const handleSave = async () => {
  try {
    await api.post<Project>('/projects', { name });
    navigate('/projects');
  } catch (err) {
    setError(err instanceof Error ? err.message : 'Something went wrong');
  }
};
```

For data fetching in `useEffect`, use `.catch(() => {})` or `.catch(() => null)` to prevent unhandled promise rejections, then handle the null/empty state in the UI.

## Collection Responses

The backend wraps collections in a named key. Destructure accordingly:

```typescript
// Backend returns: { "projects": [...] }
const resp = await api.get<{ projects: Project[] }>('/projects');
setProjects(resp.projects);

// Backend returns: { "sources": [...] }
const resp = await api.get<{ sources: CostSource[] }>(`/projects/${id}/sources`);
setSources(resp.sources);
```

## Mutation Pattern

For create/update/delete operations:

```tsx
const handleCreate = async () => {
  try {
    const created = await api.post<Project>('/projects', {
      name: formName,
      description: formDesc,
    });
    // Navigate to the new resource or refresh the list
    navigate(`/projects/${created.id}`);
  } catch (err) {
    setError(err instanceof Error ? err.message : 'Failed to create');
  }
};
```

## WebSocket Events

For real-time data, connect to `/api/v1/stream` via WebSocket. The `stream` package on the backend broadcasts events. The frontend subscribes to topics:

```typescript
const ws = new WebSocket(`ws://${window.location.host}/api/v1/stream`);
ws.send(JSON.stringify({ action: 'subscribe', topics: ['costs', 'cluster'] }));
ws.onmessage = (evt) => {
  const event = JSON.parse(evt.data);
  // handle event
};
```

## Vite Dev Proxy

During development (`npm run dev`), the Vite dev server on port 3000 proxies these paths to the Go backend on port 8080:

- `/api` -> `http://localhost:8080`
- `/login`, `/callback`, `/logout` -> `http://localhost:8080`
- `/healthz` -> `http://localhost:8080`
- `/plugins` -> `http://localhost:8080`

No CORS configuration is needed in development.
