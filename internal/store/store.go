package store

import (
	"context"
	"io/fs"
	"time"

	"github.com/inelson/finguard/internal/models"
)

type Store interface {
	Close() error
	Migrate(migrationsFS fs.FS) error

	// Projects
	CreateProject(ctx context.Context, p *models.Project) error
	GetProject(ctx context.Context, id string) (*models.Project, error)
	ListProjects(ctx context.Context) ([]*models.Project, error)
	UpdateProject(ctx context.Context, p *models.Project) error
	DeleteProject(ctx context.Context, id string) error

	// Cost Sources
	CreateCostSource(ctx context.Context, cs *models.CostSource) error
	GetCostSource(ctx context.Context, id string) (*models.CostSource, error)
	ListCostSources(ctx context.Context, projectID string) ([]*models.CostSource, error)
	UpdateCostSource(ctx context.Context, cs *models.CostSource) error
	DeleteCostSource(ctx context.Context, id string) error
	UpdateCostSourceCollectedAt(ctx context.Context, id string, t time.Time) error

	// Users
	CreateUser(ctx context.Context, u *models.User) error
	GetUser(ctx context.Context, id string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByOIDCSubject(ctx context.Context, subject string) (*models.User, error)
	ListUsers(ctx context.Context) ([]*models.User, error)

	// Groups
	CreateGroup(ctx context.Context, g *models.Group) error
	GetGroup(ctx context.Context, id string) (*models.Group, error)
	GetGroupByOIDCClaim(ctx context.Context, claim string) (*models.Group, error)
	ListGroups(ctx context.Context) ([]*models.Group, error)
	AddGroupMember(ctx context.Context, groupID, userID string) error
	RemoveGroupMember(ctx context.Context, groupID, userID string) error
	ListGroupMembers(ctx context.Context, groupID string) ([]*models.User, error)

	// Project Roles
	SetProjectRole(ctx context.Context, pr *models.ProjectRole) error
	RemoveProjectRole(ctx context.Context, projectID string, subjectType models.SubjectType, subjectID string) error
	ListProjectRoles(ctx context.Context, projectID string) ([]*models.ProjectRole, error)
	GetUserProjectRole(ctx context.Context, projectID, userID string) (*models.ProjectRole, error)
	ListUserProjects(ctx context.Context, userID string) ([]*models.Project, error)

	// Budgets
	CreateBudget(ctx context.Context, b *models.Budget) error
	GetBudget(ctx context.Context, id string) (*models.Budget, error)
	ListBudgets(ctx context.Context, projectID string) ([]*models.Budget, error)
	UpdateBudget(ctx context.Context, b *models.Budget) error
	DeleteBudget(ctx context.Context, id string) error

	// Cost Records
	InsertCostRecords(ctx context.Context, records []*models.CostRecord) error
	QueryCostRecords(ctx context.Context, q CostQuery) ([]*models.CostRecord, error)
	AggregateCosts(ctx context.Context, q CostQuery) (*CostSummary, error)
}

type CostQuery struct {
	ProjectID    string
	CostSourceID string
	Provider     string
	Service      string
	StartTime    time.Time
	EndTime      time.Time
	GroupBy      string
	Limit        int
	Offset       int
}

type CostSummary struct {
	TotalListCost     float64 `json:"totalListCost"`
	TotalNetCost      float64 `json:"totalNetCost"`
	TotalAmortized    float64 `json:"totalAmortized"`
	TotalAmortizedNet float64 `json:"totalAmortizedNet"`
	RecordCount       int     `json:"recordCount"`
}
