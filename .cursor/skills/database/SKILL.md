---
name: database
description: Database patterns for FinGuard. Covers the Store interface, SQLStore implementation, migration conventions, query patterns, transaction handling, and multi-database portability (PostgreSQL and SQLite). Use when writing database queries, adding migrations, modifying the store, or creating new database tables.
---

# Database Patterns

## Architecture

```
internal/store/
  store.go     -- Store interface (the contract)
  sql.go       -- SQLStore implementation (database/sql)
  rebind.go    -- Placeholder rebinding (? -> $1 for Postgres)
  migrate.go   -- Migration runner
  drivers.go   -- Driver registration

migrations/
  embed.go            -- go:embed for migration files
  000001_init.up.sql
  000001_init.down.sql
```

The `Store` interface in `store.go` defines all data operations. `SQLStore` in `sql.go` implements it. Code outside `internal/store/` only depends on the interface.

## Store Interface

Group methods by domain with comments. Every method takes `context.Context` as the first parameter.

```go
type Store interface {
    Close() error
    Migrate(migrationsFS fs.FS) error

    // Things
    CreateThing(ctx context.Context, t *models.Thing) error
    GetThing(ctx context.Context, id string) (*models.Thing, error)
    ListThings(ctx context.Context) ([]*models.Thing, error)
    UpdateThing(ctx context.Context, t *models.Thing) error
    DeleteThing(ctx context.Context, id string) error
}
```

Conventions:
- `Create` returns error; the model is mutated in place (ID, timestamps set).
- `Get` returns `(*T, nil)` when found, `(nil, nil)` when not found, `(nil, err)` on failure.
- `List` returns `([]*T, nil)` -- may be nil slice (handler converts to `[]`).
- `Update` returns error; caller must set changed fields before calling.
- `Delete` returns error; no-op if already deleted.

## SQLStore Implementation

### Create Pattern

```go
func (s *SQLStore) CreateThing(ctx context.Context, t *models.Thing) error {
    if t.ID == "" {
        t.ID = newID()    // uuid.New().String()
    }
    t.CreatedAt = now()   // time.Now().UTC()
    t.UpdatedAt = t.CreatedAt
    _, err := s.db.ExecContext(ctx,
        `INSERT INTO things (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)`,
        t.ID, t.Name, t.CreatedAt, t.UpdatedAt,
    )
    return err
}
```

### Get Pattern (single row)

```go
func (s *SQLStore) GetThing(ctx context.Context, id string) (*models.Thing, error) {
    t := &models.Thing{}
    err := s.db.QueryRowContext(ctx,
        `SELECT id, name, created_at, updated_at FROM things WHERE id = ?`, id,
    ).Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    return t, err
}
```

### List Pattern (multiple rows)

```go
func (s *SQLStore) ListThings(ctx context.Context) ([]*models.Thing, error) {
    rows, err := s.db.QueryContext(ctx,
        `SELECT id, name, created_at, updated_at FROM things ORDER BY name`,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var things []*models.Thing
    for rows.Next() {
        t := &models.Thing{}
        if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt); err != nil {
            return nil, err
        }
        things = append(things, t)
    }
    return things, rows.Err()
}
```

### Update Pattern

```go
func (s *SQLStore) UpdateThing(ctx context.Context, t *models.Thing) error {
    t.UpdatedAt = now()
    _, err := s.db.ExecContext(ctx,
        `UPDATE things SET name = ?, updated_at = ? WHERE id = ?`,
        t.Name, t.UpdatedAt, t.ID,
    )
    return err
}
```

### Delete Pattern

```go
func (s *SQLStore) DeleteThing(ctx context.Context, id string) error {
    _, err := s.db.ExecContext(ctx, `DELETE FROM things WHERE id = ?`, id)
    return err
}
```

## Transactions

Use transactions for multi-step operations:

```go
func (s *SQLStore) InsertBatch(ctx context.Context, records []*models.Record) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    stmt, err := tx.PrepareContext(ctx, `INSERT INTO records (...) VALUES (?, ?, ?)`)
    if err != nil {
        return err
    }
    defer stmt.Close()

    for _, r := range records {
        if _, err := stmt.ExecContext(ctx, r.A, r.B, r.C); err != nil {
            return err
        }
    }
    return tx.Commit()
}
```

## Migrations

Files in `migrations/` are embedded via `go:embed` and run at startup.

### Naming

```
{number}_{description}.up.sql
{number}_{description}.down.sql
```

Number is zero-padded to 6 digits: `000001`, `000002`, etc. Always provide both up and down.

### Writing Migrations

```sql
-- 000002_add_alerts.up.sql
CREATE TABLE IF NOT EXISTS alerts (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    threshold   REAL NOT NULL DEFAULT 0,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_alerts_project ON alerts(project_id);
```

```sql
-- 000002_add_alerts.down.sql
DROP TABLE IF EXISTS alerts;
```

### SQL Conventions

- Primary keys: `TEXT` type, UUID values generated in Go via `uuid.New().String()`.
- Foreign keys: Always specify `ON DELETE CASCADE` or `ON DELETE SET NULL`.
- Timestamps: `TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP`. Go stores UTC.
- Boolean: `BOOLEAN NOT NULL DEFAULT TRUE`.
- JSON data: `TEXT NOT NULL DEFAULT '{}'` (stored as string, parsed in Go).
- Column naming: `snake_case`.
- Always add indexes on foreign key columns and frequently queried columns.
- Use `IF NOT EXISTS` / `IF EXISTS` for idempotent migrations.

## Multi-Database Portability

FinGuard supports both PostgreSQL and SQLite.

- Write queries with `?` placeholders. The `rebindDB` wrapper converts `?` to `$1, $2, ...` for PostgreSQL.
- Avoid database-specific syntax. When unavoidable, branch on `s.driver`:

```go
if s.driver == "pgx" {
    query = `INSERT INTO ... ON CONFLICT (col) DO NOTHING`
} else {
    query = `INSERT OR IGNORE INTO ...`
}
```

- Test migrations against both databases when possible.
- SQLite runs with `PRAGMA journal_mode=WAL` and `PRAGMA foreign_keys=ON`.

## Nullable Columns

For nullable string columns, use `sql.NullString` in Scan, then unwrap:

```go
var oidcSubject sql.NullString
err := row.Scan(&u.ID, &u.Email, &oidcSubject)
u.OIDCSubject = oidcSubject.String
```

For nullable time columns, use `*time.Time` in the model struct directly.

## Helper Functions

```go
func newID() string       { return uuid.New().String() }
func now() time.Time      { return time.Now().UTC() }
func nullString(s string) sql.NullString { ... }
```

Use these consistently rather than inlining UUID/time generation.

## Testing Requirements

New store methods are not complete until they have passing tests. Follow the **testing** skill for patterns (in-memory SQLite test store, table-driven tests).

When adding or modifying a store method or migration:

1. Write tests using `newTestStore(t)` covering the happy path and error/edge cases (not found, duplicates, cascading deletes).
2. Run `make test` and confirm all tests pass (backend + frontend).
3. Do not consider the feature implemented until `make test` exits cleanly.
