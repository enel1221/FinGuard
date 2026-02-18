package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/inelson/finguard/internal/models"
)

type SQLStore struct {
	db     *rebindDB
	rawDB  *sql.DB
	driver string
}

func New(dsn string) (*SQLStore, error) {
	driver, connStr := parseDSN(dsn)

	rawDB, err := sql.Open(driver, connStr)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := rawDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if driver == "sqlite" {
		if _, err := rawDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
			return nil, fmt.Errorf("set WAL journal mode: %w", err)
		}
		if _, err := rawDB.Exec("PRAGMA foreign_keys=ON"); err != nil {
			return nil, fmt.Errorf("enable foreign keys: %w", err)
		}
		rawDB.SetMaxOpenConns(1)
	}

	db := &rebindDB{db: rawDB, driver: driver}
	return &SQLStore{db: db, rawDB: rawDB, driver: driver}, nil
}

func parseDSN(dsn string) (driver, connStr string) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return "pgx", dsn
	}
	connStr = strings.TrimPrefix(dsn, "sqlite://")
	connStr = strings.TrimPrefix(connStr, "sqlite:///")
	if connStr == "" {
		connStr = "/tmp/finguard.db"
	}
	return "sqlite", connStr
}

func (s *SQLStore) Close() error {
	return s.rawDB.Close()
}

func (s *SQLStore) Migrate(migrationsFS fs.FS) error {
	return runMigrations(s.db, migrationsFS)
}

func newID() string {
	return uuid.New().String()
}

func now() time.Time {
	return time.Now().UTC()
}

// --- Projects ---

func (s *SQLStore) CreateProject(ctx context.Context, p *models.Project) error {
	if p.ID == "" {
		p.ID = newID()
	}
	p.CreatedAt = now()
	p.UpdatedAt = p.CreatedAt
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO projects (id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Description, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (s *SQLStore) GetProject(ctx context.Context, id string) (*models.Project, error) {
	p := &models.Project{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, created_at, updated_at FROM projects WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (s *SQLStore) ListProjects(ctx context.Context) ([]*models.Project, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, description, created_at, updated_at FROM projects ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		p := &models.Project{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (s *SQLStore) UpdateProject(ctx context.Context, p *models.Project) error {
	p.UpdatedAt = now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE projects SET name = ?, description = ?, updated_at = ? WHERE id = ?`,
		p.Name, p.Description, p.UpdatedAt, p.ID,
	)
	return err
}

func (s *SQLStore) DeleteProject(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	return err
}

// --- Cost Sources ---

func (s *SQLStore) CreateCostSource(ctx context.Context, cs *models.CostSource) error {
	if cs.ID == "" {
		cs.ID = newID()
	}
	cs.CreatedAt = now()
	cs.UpdatedAt = cs.CreatedAt
	configJSON := string(cs.Config)
	if configJSON == "" {
		configJSON = "{}"
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO cost_sources (id, project_id, type, name, config_json, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		cs.ID, cs.ProjectID, cs.Type, cs.Name, configJSON, cs.Enabled, cs.CreatedAt, cs.UpdatedAt,
	)
	return err
}

func (s *SQLStore) GetCostSource(ctx context.Context, id string) (*models.CostSource, error) {
	cs := &models.CostSource{}
	var configJSON string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, type, name, config_json, enabled, last_collected_at, created_at, updated_at FROM cost_sources WHERE id = ?`, id,
	).Scan(&cs.ID, &cs.ProjectID, &cs.Type, &cs.Name, &configJSON, &cs.Enabled, &cs.LastCollectedAt, &cs.CreatedAt, &cs.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	cs.Config = json.RawMessage(configJSON)
	return cs, err
}

func (s *SQLStore) ListCostSources(ctx context.Context, projectID string) ([]*models.CostSource, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, type, name, config_json, enabled, last_collected_at, created_at, updated_at FROM cost_sources WHERE project_id = ? ORDER BY name`, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []*models.CostSource
	for rows.Next() {
		cs := &models.CostSource{}
		var configJSON string
		if err := rows.Scan(&cs.ID, &cs.ProjectID, &cs.Type, &cs.Name, &configJSON, &cs.Enabled, &cs.LastCollectedAt, &cs.CreatedAt, &cs.UpdatedAt); err != nil {
			return nil, err
		}
		cs.Config = json.RawMessage(configJSON)
		sources = append(sources, cs)
	}
	return sources, rows.Err()
}

func (s *SQLStore) UpdateCostSource(ctx context.Context, cs *models.CostSource) error {
	cs.UpdatedAt = now()
	configJSON := string(cs.Config)
	if configJSON == "" {
		configJSON = "{}"
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE cost_sources SET name = ?, config_json = ?, enabled = ?, updated_at = ? WHERE id = ?`,
		cs.Name, configJSON, cs.Enabled, cs.UpdatedAt, cs.ID,
	)
	return err
}

func (s *SQLStore) DeleteCostSource(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM cost_sources WHERE id = ?`, id)
	return err
}

func (s *SQLStore) UpdateCostSourceCollectedAt(ctx context.Context, id string, t time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE cost_sources SET last_collected_at = ?, updated_at = ? WHERE id = ?`,
		t, now(), id,
	)
	return err
}

// --- Users ---

func (s *SQLStore) CreateUser(ctx context.Context, u *models.User) error {
	if u.ID == "" {
		u.ID = newID()
	}
	u.CreatedAt = now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, display_name, oidc_subject, created_at) VALUES (?, ?, ?, ?, ?)`,
		u.ID, u.Email, u.DisplayName, nullString(u.OIDCSubject), u.CreatedAt,
	)
	return err
}

func (s *SQLStore) GetUser(ctx context.Context, id string) (*models.User, error) {
	u := &models.User{}
	var oidcSubject sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, display_name, oidc_subject, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.DisplayName, &oidcSubject, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	u.OIDCSubject = oidcSubject.String
	return u, err
}

func (s *SQLStore) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	u := &models.User{}
	var oidcSubject sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, display_name, oidc_subject, created_at FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &u.DisplayName, &oidcSubject, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	u.OIDCSubject = oidcSubject.String
	return u, err
}

func (s *SQLStore) GetUserByOIDCSubject(ctx context.Context, subject string) (*models.User, error) {
	u := &models.User{}
	var oidcSubject sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, display_name, oidc_subject, created_at FROM users WHERE oidc_subject = ?`, subject,
	).Scan(&u.ID, &u.Email, &u.DisplayName, &oidcSubject, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	u.OIDCSubject = oidcSubject.String
	return u, err
}

func (s *SQLStore) ListUsers(ctx context.Context) ([]*models.User, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, email, display_name, oidc_subject, created_at FROM users ORDER BY email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		var oidcSubject sql.NullString
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &oidcSubject, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.OIDCSubject = oidcSubject.String
		users = append(users, u)
	}
	return users, rows.Err()
}

// --- Groups ---

func (s *SQLStore) CreateGroup(ctx context.Context, g *models.Group) error {
	if g.ID == "" {
		g.ID = newID()
	}
	g.CreatedAt = now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO groups (id, name, oidc_claim, created_at) VALUES (?, ?, ?, ?)`,
		g.ID, g.Name, nullString(g.OIDCClaim), g.CreatedAt,
	)
	return err
}

func (s *SQLStore) GetGroup(ctx context.Context, id string) (*models.Group, error) {
	g := &models.Group{}
	var oidcClaim sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, oidc_claim, created_at FROM groups WHERE id = ?`, id,
	).Scan(&g.ID, &g.Name, &oidcClaim, &g.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	g.OIDCClaim = oidcClaim.String
	return g, err
}

func (s *SQLStore) GetGroupByOIDCClaim(ctx context.Context, claim string) (*models.Group, error) {
	g := &models.Group{}
	var oidcClaim sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, oidc_claim, created_at FROM groups WHERE oidc_claim = ?`, claim,
	).Scan(&g.ID, &g.Name, &oidcClaim, &g.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	g.OIDCClaim = oidcClaim.String
	return g, err
}

func (s *SQLStore) ListGroups(ctx context.Context) ([]*models.Group, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, oidc_claim, created_at FROM groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*models.Group
	for rows.Next() {
		g := &models.Group{}
		var oidcClaim sql.NullString
		if err := rows.Scan(&g.ID, &g.Name, &oidcClaim, &g.CreatedAt); err != nil {
			return nil, err
		}
		g.OIDCClaim = oidcClaim.String
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (s *SQLStore) AddGroupMember(ctx context.Context, groupID, userID string) error {
	var query string
	if s.driver == "pgx" {
		query = `INSERT INTO group_members (group_id, user_id) VALUES (?, ?) ON CONFLICT (group_id, user_id) DO NOTHING`
	} else {
		// SQLite doesn't support ON CONFLICT (cols) DO NOTHING on all versions;
		// INSERT OR IGNORE is the canonical SQLite equivalent.
		query = `INSERT OR IGNORE INTO group_members (group_id, user_id) VALUES (?, ?)`
	}
	_, err := s.db.ExecContext(ctx, query, groupID, userID)
	return err
}

func (s *SQLStore) RemoveGroupMember(ctx context.Context, groupID, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM group_members WHERE group_id = ? AND user_id = ?`, groupID, userID)
	return err
}

func (s *SQLStore) ListGroupMembers(ctx context.Context, groupID string) ([]*models.User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT u.id, u.email, u.display_name, u.oidc_subject, u.created_at FROM users u JOIN group_members gm ON u.id = gm.user_id WHERE gm.group_id = ? ORDER BY u.email`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		var oidcSubject sql.NullString
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &oidcSubject, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.OIDCSubject = oidcSubject.String
		users = append(users, u)
	}
	return users, rows.Err()
}

// --- Project Roles ---

func (s *SQLStore) SetProjectRole(ctx context.Context, pr *models.ProjectRole) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO project_roles (project_id, subject_type, subject_id, role) VALUES (?, ?, ?, ?)
		ON CONFLICT(project_id, subject_type, subject_id) DO UPDATE SET role = excluded.role`,
		pr.ProjectID, pr.SubjectType, pr.SubjectID, pr.Role,
	)
	return err
}

func (s *SQLStore) RemoveProjectRole(ctx context.Context, projectID string, subjectType models.SubjectType, subjectID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM project_roles WHERE project_id = ? AND subject_type = ? AND subject_id = ?`,
		projectID, subjectType, subjectID,
	)
	return err
}

func (s *SQLStore) ListProjectRoles(ctx context.Context, projectID string) ([]*models.ProjectRole, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT project_id, subject_type, subject_id, role FROM project_roles WHERE project_id = ?`, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*models.ProjectRole
	for rows.Next() {
		pr := &models.ProjectRole{}
		if err := rows.Scan(&pr.ProjectID, &pr.SubjectType, &pr.SubjectID, &pr.Role); err != nil {
			return nil, err
		}
		roles = append(roles, pr)
	}
	return roles, rows.Err()
}

func (s *SQLStore) GetUserProjectRole(ctx context.Context, projectID, userID string) (*models.ProjectRole, error) {
	pr := &models.ProjectRole{}
	err := s.db.QueryRowContext(ctx,
		`SELECT project_id, subject_type, subject_id, role FROM project_roles WHERE project_id = ? AND subject_type = 'user' AND subject_id = ?`,
		projectID, userID,
	).Scan(&pr.ProjectID, &pr.SubjectType, &pr.SubjectID, &pr.Role)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return pr, err
}

func (s *SQLStore) ListUserProjects(ctx context.Context, userID string) ([]*models.Project, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT p.id, p.name, p.description, p.created_at, p.updated_at
		FROM projects p
		LEFT JOIN project_roles pr ON p.id = pr.project_id
		WHERE (pr.subject_type = 'user' AND pr.subject_id = ?)
		   OR pr.subject_id IN (SELECT group_id FROM group_members WHERE user_id = ?)
		ORDER BY p.name`, userID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		p := &models.Project{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// --- Budgets ---

func (s *SQLStore) CreateBudget(ctx context.Context, b *models.Budget) error {
	if b.ID == "" {
		b.ID = newID()
	}
	b.CreatedAt = now()
	b.UpdatedAt = b.CreatedAt
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO budgets (id, project_id, cost_source_id, monthly_limit, warn_threshold, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		b.ID, b.ProjectID, b.CostSourceID, b.MonthlyLimit, b.WarnThreshold, b.CreatedAt, b.UpdatedAt,
	)
	return err
}

func (s *SQLStore) GetBudget(ctx context.Context, id string) (*models.Budget, error) {
	b := &models.Budget{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, cost_source_id, monthly_limit, warn_threshold, created_at, updated_at FROM budgets WHERE id = ?`, id,
	).Scan(&b.ID, &b.ProjectID, &b.CostSourceID, &b.MonthlyLimit, &b.WarnThreshold, &b.CreatedAt, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return b, err
}

func (s *SQLStore) ListBudgets(ctx context.Context, projectID string) ([]*models.Budget, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, cost_source_id, monthly_limit, warn_threshold, created_at, updated_at FROM budgets WHERE project_id = ?`, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgets []*models.Budget
	for rows.Next() {
		b := &models.Budget{}
		if err := rows.Scan(&b.ID, &b.ProjectID, &b.CostSourceID, &b.MonthlyLimit, &b.WarnThreshold, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		budgets = append(budgets, b)
	}
	return budgets, rows.Err()
}

func (s *SQLStore) UpdateBudget(ctx context.Context, b *models.Budget) error {
	b.UpdatedAt = now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE budgets SET monthly_limit = ?, warn_threshold = ?, updated_at = ? WHERE id = ?`,
		b.MonthlyLimit, b.WarnThreshold, b.UpdatedAt, b.ID,
	)
	return err
}

func (s *SQLStore) DeleteBudget(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM budgets WHERE id = ?`, id)
	return err
}

// --- Cost Records ---

func (s *SQLStore) InsertCostRecords(ctx context.Context, records []*models.CostRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO cost_records (id, project_id, cost_source_id, provider, provider_id, account_id, account_name, invoice_entity_id, service, category, region, availability_zone, start_time, end_time, list_cost, net_cost, amortized_cost, amortized_net_cost, currency, labels_json, kubernetes_percent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range records {
		if r.ID == "" {
			r.ID = newID()
		}
		var labelsJSON []byte
		if r.Labels == nil {
			labelsJSON = []byte("{}")
		} else {
			labelsJSON, _ = json.Marshal(r.Labels)
		}
		// Prepared statements already have placeholders rebound, so use raw ExecContext
		_, err := stmt.ExecContext(ctx,
			r.ID, r.ProjectID, r.CostSourceID, r.Provider, r.ProviderID, r.AccountID, r.AccountName, r.InvoiceEntityID,
			r.Service, r.Category, r.Region, r.AvailabilityZone, r.StartTime, r.EndTime,
			r.ListCost, r.NetCost, r.AmortizedCost, r.AmortizedNetCost, r.Currency, string(labelsJSON), r.KubernetesPercent,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLStore) QueryCostRecords(ctx context.Context, q CostQuery) ([]*models.CostRecord, error) {
	where, args := buildCostWhere(q)
	query := `SELECT id, project_id, cost_source_id, provider, provider_id, account_id, account_name, invoice_entity_id, service, category, region, availability_zone, start_time, end_time, list_cost, net_cost, amortized_cost, amortized_net_cost, currency, labels_json, kubernetes_percent FROM cost_records` + where + ` ORDER BY start_time DESC`

	if q.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", q.Limit)
	}
	if q.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", q.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*models.CostRecord
	for rows.Next() {
		r := &models.CostRecord{}
		if err := rows.Scan(
			&r.ID, &r.ProjectID, &r.CostSourceID, &r.Provider, &r.ProviderID, &r.AccountID, &r.AccountName, &r.InvoiceEntityID,
			&r.Service, &r.Category, &r.Region, &r.AvailabilityZone, &r.StartTime, &r.EndTime,
			&r.ListCost, &r.NetCost, &r.AmortizedCost, &r.AmortizedNetCost, &r.Currency, &r.LabelsJSON, &r.KubernetesPercent,
		); err != nil {
			return nil, err
		}
		if r.LabelsJSON != "" && r.LabelsJSON != "{}" {
			if err := json.Unmarshal([]byte(r.LabelsJSON), &r.Labels); err != nil {
				return nil, fmt.Errorf("unmarshal labels for record %s: %w", r.ID, err)
			}
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func (s *SQLStore) AggregateCosts(ctx context.Context, q CostQuery) (*CostSummary, error) {
	where, args := buildCostWhere(q)
	query := `SELECT COALESCE(SUM(list_cost),0), COALESCE(SUM(net_cost),0), COALESCE(SUM(amortized_cost),0), COALESCE(SUM(amortized_net_cost),0), COUNT(*) FROM cost_records` + where

	summary := &CostSummary{}
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&summary.TotalListCost, &summary.TotalNetCost, &summary.TotalAmortized, &summary.TotalAmortizedNet, &summary.RecordCount,
	)
	return summary, err
}

func buildCostWhere(q CostQuery) (string, []any) {
	var conditions []string
	var args []any

	if q.ProjectID != "" {
		conditions = append(conditions, "project_id = ?")
		args = append(args, q.ProjectID)
	}
	if q.CostSourceID != "" {
		conditions = append(conditions, "cost_source_id = ?")
		args = append(args, q.CostSourceID)
	}
	if q.Provider != "" {
		conditions = append(conditions, "provider = ?")
		args = append(args, q.Provider)
	}
	if q.Service != "" {
		conditions = append(conditions, "service = ?")
		args = append(args, q.Service)
	}
	if !q.StartTime.IsZero() {
		conditions = append(conditions, "start_time >= ?")
		args = append(args, q.StartTime)
	}
	if !q.EndTime.IsZero() {
		conditions = append(conditions, "end_time <= ?")
		args = append(args, q.EndTime)
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
