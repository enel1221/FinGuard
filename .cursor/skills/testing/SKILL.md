---
name: testing
description: Testing patterns for FinGuard. Covers Go backend tests (httptest, table-driven, store) and React frontend tests (Vitest, Testing Library). Use when writing tests, adding test coverage, creating test helpers, or reviewing test code.
---

# Testing Patterns

## Running All Tests

From the repo root, `make test` runs both backend (Go) and frontend (React) tests:

```bash
make test
```

To run each stack independently:

```bash
# Backend only
go test -race -coverprofile=coverage.out ./...
go test -race ./internal/server/                   # single package
go test -run TestHealthz ./internal/server/         # single test

# Frontend only
cd web/frontend && npm run test
```

---

## Backend (Go) Tests

### File Placement

Test files live alongside the source they test:

```
internal/server/server.go
internal/server/server_test.go      -- same package (white-box)
internal/config/config.go
internal/config/config_test.go
```

### Test Naming

```go
func TestFunctionName(t *testing.T) { ... }
func TestFunctionName_scenario(t *testing.T) { ... }
```

Examples from the codebase:
- `TestHealthz`
- `TestReadyz_NoCacheReady`
- `TestDetailedHealth`
- `TestClusterEndpoint_NoCache`
- `TestLoad_Defaults`
- `TestLoad_EnvOverride`
- `TestEnvIntOr`

### Assertions

Use standard library assertions. No external test frameworks (no testify):

```go
if got != want {
    t.Errorf("expected %v, got %v", want, got)
}

if err != nil {
    t.Fatalf("unexpected error: %v", err)
}
```

- `t.Errorf` for non-fatal failures (test continues).
- `t.Fatalf` / `t.Fatal` for fatal failures (test stops).

### HTTP Handler Tests

Use `httptest.NewRequest` + `httptest.NewRecorder` + the Chi router:

```go
func newTestServer() *Server {
    cfg := &config.Config{
        HTTPAddr:    ":0",
        OpenCostURL: "http://localhost:9003",
    }
    logger := testLogger()
    hub := stream.NewHub(logger)
    proxy := opencostproxy.New(cfg.OpenCostURL, logger)
    return New(cfg, hub, proxy, nil, nil, nil, logger)
}

func testLogger() *slog.Logger {
    return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestMyEndpoint(t *testing.T) {
    srv := newTestServer()
    req := httptest.NewRequest(http.MethodGet, "/api/v1/things", nil)
    w := httptest.NewRecorder()

    srv.router.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", w.Code)
    }

    var body map[string]any
    json.NewDecoder(w.Body).Decode(&body)
    // assert on body contents
}
```

Key points:
- Create a test server with `newTestServer()` (nil dependencies where not needed).
- Route through `srv.router.ServeHTTP()` to test the full middleware chain.
- Decode response body and assert on fields.

### Table-Driven Tests

For functions with multiple input/output cases:

```go
func TestParseThreshold(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    float64
        wantErr bool
    }{
        {"valid percentage", "80%", 0.8, false},
        {"valid decimal", "0.5", 0.5, false},
        {"empty string", "", 0, true},
        {"negative", "-1", 0, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := parseThreshold(tt.input)
            if (err != nil) != tt.wantErr {
                t.Fatalf("error = %v, wantErr = %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

Conventions:
- Test case struct has `name` as the first field.
- Use `t.Run(tt.name, ...)` for subtests.
- Variable name `tt` for the current test case.
- `want` / `wantErr` naming for expected values.

### Environment Variable Tests

When testing config loading, set and unset env vars:

```go
func TestLoad_EnvOverride(t *testing.T) {
    os.Setenv("FINGUARD_ADDR", ":9090")
    defer os.Unsetenv("FINGUARD_ADDR")

    cfg := Load()
    if cfg.HTTPAddr != ":9090" {
        t.Errorf("expected ':9090', got %q", cfg.HTTPAddr)
    }
}
```

### Store Tests

For store tests, use an in-memory SQLite database:

```go
func newTestStore(t *testing.T) store.Store {
    t.Helper()
    s, err := store.New("sqlite:///tmp/test_" + uuid.New().String() + ".db")
    if err != nil {
        t.Fatal(err)
    }
    if err := s.Migrate(migrations.FS); err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() { s.Close() })
    return s
}
```

---

## Frontend (React) Tests

Uses Vitest as the test runner and React Testing Library for component tests. Global test setup lives in `web/frontend/src/test/setup.ts` (imports `@testing-library/jest-dom` matchers).

### File Placement

Tests go alongside components or in a `__tests__/` subfolder:

```
components/Dashboard/Dashboard.tsx
components/Dashboard/Dashboard.test.tsx
```

### Component Test Pattern

```tsx
import { render, screen, waitFor } from '@testing-library/react';
import { vi } from 'vitest';
import Dashboard from './Dashboard';

vi.mock('@/lib/api', () => ({
  api: {
    get: vi.fn(),
  },
}));

import { api } from '@/lib/api';

test('renders dashboard title', async () => {
  (api.get as ReturnType<typeof vi.fn>).mockResolvedValue({
    status: 'ok',
    services: { opencost: 'healthy' },
  });

  render(<Dashboard />);

  await waitFor(() => {
    expect(screen.getByText('Dashboard')).toBeInTheDocument();
  });
});
```

### Testing Principles

- Test user-visible behavior, not implementation details.
- Mock the `api` module, not fetch.
- Use `screen.getByText()`, `screen.getByRole()` over `container.querySelector()`.
- Test loading states, error states, and empty states.
- Wrap components in necessary providers (Redux, Router, Theme) via a test utility:

```tsx
function renderWithProviders(ui: React.ReactElement) {
  return render(
    <Provider store={store}>
      <ThemeProvider theme={theme}>
        <MemoryRouter>
          {ui}
        </MemoryRouter>
      </ThemeProvider>
    </Provider>
  );
}
```

---

## Coverage

- **Backend**: Focus coverage on business logic in `internal/` packages. Handler tests should cover the happy path + key error cases (400, 404, 500). Skip coverage for trivial wiring code (`cmd/finguard/main.go`). Coverage is produced by `go test -coverprofile=coverage.out`.
- **Frontend**: Add `--coverage` to the Vitest command when coverage reporting is needed: `cd web/frontend && npx vitest run --coverage`.
