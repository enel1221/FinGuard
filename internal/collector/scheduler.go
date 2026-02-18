package collector

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/inelson/finguard/internal/models"
	"github.com/inelson/finguard/internal/store"
	"github.com/inelson/finguard/internal/stream"
	"github.com/inelson/finguard/pkg/event"
)

type SchedulerConfig struct {
	CSPInterval        time.Duration
	KubernetesInterval time.Duration
}

func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		CSPInterval:        1 * time.Hour,
		KubernetesInterval: 5 * time.Minute,
	}
}

type Scheduler struct {
	registry *Registry
	store    store.Store
	hub      *stream.Hub
	config   SchedulerConfig
	logger   *slog.Logger
	mu       sync.Mutex
	running  bool
}

func NewScheduler(registry *Registry, st store.Store, hub *stream.Hub, cfg SchedulerConfig, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		registry: registry,
		store:    st,
		hub:      hub,
		config:   cfg,
		logger:   logger,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info("cost collector scheduler started")

	cspTicker := time.NewTicker(s.config.CSPInterval)
	k8sTicker := time.NewTicker(s.config.KubernetesInterval)
	defer cspTicker.Stop()
	defer k8sTicker.Stop()

	// Run an initial collection immediately
	go s.collectAll(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("cost collector scheduler stopped")
			return
		case <-cspTicker.C:
			go s.collectByTypes(ctx, models.CostSourceAWS, models.CostSourceAzure, models.CostSourceGCP)
		case <-k8sTicker.C:
			go s.collectByTypes(ctx, models.CostSourceKubernetes)
		}
	}
}

func (s *Scheduler) collectAll(ctx context.Context) {
	projects, err := s.store.ListProjects(ctx)
	if err != nil {
		s.logger.Error("scheduler: failed to list projects", "error", err)
		return
	}

	for _, project := range projects {
		s.collectForProject(ctx, project.ID)
	}
}

func (s *Scheduler) collectByTypes(ctx context.Context, types ...models.CostSourceType) {
	projects, err := s.store.ListProjects(ctx)
	if err != nil {
		s.logger.Error("scheduler: failed to list projects", "error", err)
		return
	}

	typeSet := make(map[models.CostSourceType]bool)
	for _, t := range types {
		typeSet[t] = true
	}

	for _, project := range projects {
		sources, err := s.store.ListCostSources(ctx, project.ID)
		if err != nil {
			s.logger.Error("scheduler: failed to list sources", "project", project.ID, "error", err)
			continue
		}
		for _, source := range sources {
			if !source.Enabled || !typeSet[source.Type] {
				continue
			}
			s.collectSource(ctx, source)
		}
	}
}

func (s *Scheduler) collectForProject(ctx context.Context, projectID string) {
	sources, err := s.store.ListCostSources(ctx, projectID)
	if err != nil {
		s.logger.Error("scheduler: failed to list sources", "project", projectID, "error", err)
		return
	}

	for _, source := range sources {
		if !source.Enabled {
			continue
		}
		s.collectSource(ctx, source)
	}
}

func (s *Scheduler) collectSource(ctx context.Context, source *models.CostSource) {
	collector, ok := s.registry.Get(source.Type)
	if !ok {
		s.logger.Warn("scheduler: no collector for source type", "type", source.Type, "source", source.Name)
		return
	}

	window := TimeWindow{
		Start: time.Now().UTC().Add(-24 * time.Hour),
		End:   time.Now().UTC(),
	}
	if source.LastCollectedAt != nil {
		window.Start = *source.LastCollectedAt
	}

	s.logger.Info("collecting costs", "source", source.Name, "type", source.Type, "window", window)

	records, err := collector.Collect(ctx, source, window)
	if err != nil {
		s.logger.Error("collection failed", "source", source.Name, "type", source.Type, "error", err)
		s.publishEvent("collection.failed", source.Name, map[string]string{
			"sourceId": source.ID,
			"error":    err.Error(),
		})
		return
	}

	if len(records) > 0 {
		if err := s.store.InsertCostRecords(ctx, records); err != nil {
			s.logger.Error("failed to insert cost records", "source", source.Name, "count", len(records), "error", err)
			return
		}
	}

	if err := s.store.UpdateCostSourceCollectedAt(ctx, source.ID, time.Now().UTC()); err != nil {
		s.logger.Error("failed to update collected_at", "source", source.Name, "error", err)
	}

	s.logger.Info("collection complete", "source", source.Name, "records", len(records))
	s.publishEvent("collection.complete", source.Name, map[string]string{
		"sourceId": source.ID,
		"records":  strconv.Itoa(len(records)),
	})
}

func (s *Scheduler) publishEvent(eventType, sourceName string, payload map[string]string) {
	if s.hub == nil {
		return
	}
	e, err := event.New(eventType, "cost.collection", sourceName, payload)
	if err != nil {
		return
	}
	s.hub.Publish(e)
}
