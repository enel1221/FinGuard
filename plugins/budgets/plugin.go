package budgets

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	pluginpkg "github.com/inelson/finguard/pkg/plugin"
)

type Budget struct {
	Namespace      string  `json:"namespace"`
	MonthlyBudget  float64 `json:"monthlyBudget"`
	WarningPercent float64 `json:"warningPercent"` // e.g. 0.8 = warn at 80%
}

type BudgetStatus struct {
	Namespace     string  `json:"namespace"`
	MonthlyBudget float64 `json:"monthlyBudget"`
	CurrentSpend  float64 `json:"currentSpend"`
	Utilization   float64 `json:"utilization"`
	Status        string  `json:"status"` // "ok", "warning", "exceeded"
	ProjectedEnd  float64 `json:"projectedEndOfMonth"`
}

type Plugin struct {
	opencostURL string
	httpClient  *http.Client
	events      chan *pluginpkg.Event
	logger      *slog.Logger
	mu          sync.RWMutex
	budgets     []Budget
	statuses    []BudgetStatus
}

func New(logger *slog.Logger, budgets []Budget) *Plugin {
	if budgets == nil {
		budgets = defaultBudgets()
	}
	return &Plugin{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		events:     make(chan *pluginpkg.Event, 100),
		logger:     logger,
		budgets:    budgets,
	}
}

func defaultBudgets() []Budget {
	return []Budget{
		{Namespace: "default", MonthlyBudget: 100, WarningPercent: 0.8},
		{Namespace: "kube-system", MonthlyBudget: 200, WarningPercent: 0.8},
	}
}

func (p *Plugin) GetMetadata() (*pluginpkg.Metadata, error) {
	return &pluginpkg.Metadata{
		Name:        "budgets",
		Version:     "0.1.0",
		Description: "Budget tracking and enforcement per namespace with streaming alerts",
		Type:        "policy",
		Topics:      []string{"budget.warning", "budget.exceeded"},
		Routes: []pluginpkg.Route{
			{Method: "GET", Path: "/status", Description: "Budget status per namespace"},
			{Method: "GET", Path: "/config", Description: "Current budget configuration"},
		},
	}, nil
}

func (p *Plugin) Initialize(ctx context.Context, req *pluginpkg.InitRequest) error {
	p.opencostURL = req.OpenCostURL

	if len(req.Config) > 0 {
		var cfg struct {
			Budgets []Budget `json:"budgets"`
		}
		if err := json.Unmarshal(req.Config, &cfg); err == nil && len(cfg.Budgets) > 0 {
			p.budgets = cfg.Budgets
		}
	}

	go p.pollLoop(ctx)
	return nil
}

func (p *Plugin) Execute(ctx context.Context, req *pluginpkg.ExecuteRequest) (*pluginpkg.ExecuteResponse, error) {
	switch req.Action {
	case "/status":
		return p.handleStatus()
	case "/config":
		return p.handleConfig()
	default:
		return &pluginpkg.ExecuteResponse{
			StatusCode: http.StatusNotFound,
			Error:      fmt.Sprintf("unknown action: %s", req.Action),
		}, nil
	}
}

func (p *Plugin) StreamEvents(ctx context.Context) (<-chan *pluginpkg.Event, error) {
	return p.events, nil
}

func (p *Plugin) Shutdown(ctx context.Context) error {
	close(p.events)
	return nil
}

func (p *Plugin) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	p.checkBudgets(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.checkBudgets(ctx)
		}
	}
}

func (p *Plugin) checkBudgets(ctx context.Context) {
	costs, err := p.fetchNamespaceCosts(ctx)
	if err != nil {
		p.logger.Error("failed to fetch namespace costs for budget check", "error", err)
		return
	}

	now := time.Now()
	daysInMonth := float64(daysInCurrentMonth(now))
	dayOfMonth := float64(now.Day())
	monthProgress := dayOfMonth / daysInMonth

	var statuses []BudgetStatus
	for _, b := range p.budgets {
		spend, ok := costs[b.Namespace]
		if !ok {
			continue
		}

		utilization := 0.0
		if b.MonthlyBudget > 0 {
			utilization = spend / b.MonthlyBudget
		}

		projected := 0.0
		if monthProgress > 0 {
			projected = spend / monthProgress
		}

		status := "ok"
		if utilization >= 1.0 {
			status = "exceeded"
		} else if utilization >= b.WarningPercent {
			status = "warning"
		}

		statuses = append(statuses, BudgetStatus{
			Namespace:     b.Namespace,
			MonthlyBudget: b.MonthlyBudget,
			CurrentSpend:  spend,
			Utilization:   utilization,
			Status:        status,
			ProjectedEnd:  projected,
		})

		if status == "warning" || status == "exceeded" {
			topic := "budget.warning"
			if status == "exceeded" {
				topic = "budget.exceeded"
			}
			payload, _ := json.Marshal(map[string]any{
				"namespace":     b.Namespace,
				"budget":        b.MonthlyBudget,
				"currentSpend":  spend,
				"utilization":   utilization,
				"projected":     projected,
				"status":        status,
			})
			select {
			case p.events <- &pluginpkg.Event{
				Type:      status,
				Topic:     topic,
				Timestamp: time.Now().UTC(),
				Source:    "budgets",
				Payload:   payload,
			}:
			default:
			}
		}
	}

	p.mu.Lock()
	p.statuses = statuses
	p.mu.Unlock()
}

func (p *Plugin) fetchNamespaceCosts(ctx context.Context) (map[string]float64, error) {
	url := p.opencostURL + "/allocation?window=month&aggregate=namespace&accumulate=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code int                        `json:"code"`
		Data []map[string]map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse allocation response: %w", err)
	}

	costs := make(map[string]float64)
	for _, window := range result.Data {
		for ns, alloc := range window {
			if total, ok := alloc["totalCost"].(float64); ok {
				costs[ns] += total
			}
		}
	}
	return costs, nil
}

func (p *Plugin) handleStatus() (*pluginpkg.ExecuteResponse, error) {
	p.mu.RLock()
	statuses := p.statuses
	p.mu.RUnlock()

	if statuses == nil {
		statuses = []BudgetStatus{}
	}

	data, err := json.Marshal(map[string]any{
		"budgets": statuses,
		"count":   len(statuses),
	})
	if err != nil {
		return nil, err
	}
	return &pluginpkg.ExecuteResponse{
		Data:        data,
		ContentType: "application/json",
		StatusCode:  http.StatusOK,
	}, nil
}

func (p *Plugin) handleConfig() (*pluginpkg.ExecuteResponse, error) {
	data, err := json.Marshal(map[string]any{
		"budgets": p.budgets,
	})
	if err != nil {
		return nil, err
	}
	return &pluginpkg.ExecuteResponse{
		Data:        data,
		ContentType: "application/json",
		StatusCode:  http.StatusOK,
	}, nil
}

func daysInCurrentMonth(t time.Time) int {
	y, m, _ := t.Date()
	return time.Date(y, m+1, 0, 0, 0, 0, 0, t.Location()).Day()
}
