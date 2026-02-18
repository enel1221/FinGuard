package store

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
)

// rebindDB wraps a *sql.DB and rewrites ? placeholders to $N for PostgreSQL.
// SQLite uses ? natively and passes through unchanged.
type rebindDB struct {
	db     *sql.DB
	driver string
}

func (r *rebindDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return r.db.ExecContext(ctx, r.rebind(query), args...)
}

func (r *rebindDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return r.db.QueryContext(ctx, r.rebind(query), args...)
}

func (r *rebindDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return r.db.QueryRowContext(ctx, r.rebind(query), args...)
}

func (r *rebindDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return r.db.PrepareContext(ctx, r.rebind(query))
}

func (r *rebindDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*rebindTx, error) {
	tx, err := r.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &rebindTx{tx: tx, driver: r.driver}, nil
}

func (r *rebindDB) Exec(query string, args ...any) (sql.Result, error) {
	return r.db.Exec(r.rebind(query), args...)
}

func (r *rebindDB) QueryRow(query string, args ...any) *sql.Row {
	return r.db.QueryRow(r.rebind(query), args...)
}

func (r *rebindDB) Ping() error {
	return r.db.Ping()
}

func (r *rebindDB) Close() error {
	return r.db.Close()
}

func (r *rebindDB) rebind(query string) string {
	if r.driver != "pgx" {
		return query
	}
	return rebindQuery(query)
}

// rebindTx wraps a *sql.Tx with the same placeholder rewriting.
type rebindTx struct {
	tx     *sql.Tx
	driver string
}

func (t *rebindTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.tx.ExecContext(ctx, t.rebind(query), args...)
}

func (t *rebindTx) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return t.tx.PrepareContext(ctx, t.rebind(query))
}

func (t *rebindTx) Commit() error {
	return t.tx.Commit()
}

func (t *rebindTx) Rollback() error {
	return t.tx.Rollback()
}

func (t *rebindTx) rebind(query string) string {
	if t.driver != "pgx" {
		return query
	}
	return rebindQuery(query)
}

// rebindQuery replaces ? placeholders with $1, $2, $3, etc.
// Skips ? inside single-quoted string literals.
func rebindQuery(query string) string {
	if !strings.Contains(query, "?") {
		return query
	}

	var b strings.Builder
	b.Grow(len(query) + 16)
	n := 0
	inQuote := false

	for i := 0; i < len(query); i++ {
		ch := query[i]
		if ch == '\'' {
			inQuote = !inQuote
			b.WriteByte(ch)
		} else if ch == '?' && !inQuote {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
		} else {
			b.WriteByte(ch)
		}
	}
	return b.String()
}
