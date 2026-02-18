package collector

import (
	"context"
	"encoding/json"
	"time"

	"github.com/inelson/finguard/internal/models"
)

type TimeWindow struct {
	Start time.Time
	End   time.Time
}

// Collector is the interface all CSP cost collectors implement.
// Modeled after OpenCost's cloud cost integration patterns.
type Collector interface {
	Type() string
	Collect(ctx context.Context, source *models.CostSource, window TimeWindow) ([]*models.CostRecord, error)
	Validate(ctx context.Context, config json.RawMessage) error
}

// Registry maps cost source types to their collector implementations.
type Registry struct {
	collectors map[models.CostSourceType]Collector
}

func NewRegistry() *Registry {
	return &Registry{
		collectors: make(map[models.CostSourceType]Collector),
	}
}

func (r *Registry) Register(sourceType models.CostSourceType, c Collector) {
	r.collectors[sourceType] = c
}

func (r *Registry) Get(sourceType models.CostSourceType) (Collector, bool) {
	c, ok := r.collectors[sourceType]
	return c, ok
}

func (r *Registry) Types() []models.CostSourceType {
	types := make([]models.CostSourceType, 0, len(r.collectors))
	for t := range r.collectors {
		types = append(types, t)
	}
	return types
}
