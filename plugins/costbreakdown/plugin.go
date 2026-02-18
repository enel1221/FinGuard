package costbreakdown

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

type Plugin struct {
	opencostURL string
	httpClient  *http.Client
	events      chan *pluginpkg.Event
	logger      *slog.Logger
	mu          sync.RWMutex
	recommendations []Recommendation
}

type Recommendation struct {
	Namespace    string  `json:"namespace"`
	Pod          string  `json:"pod"`
	Container    string  `json:"container"`
	ResourceType string  `json:"resourceType"`
	Requested    float64 `json:"requested"`
	Used         float64 `json:"used"`
	Savings      float64 `json:"estimatedSavings"`
	Severity     string  `json:"severity"`
}

func New(logger *slog.Logger) *Plugin {
	return &Plugin{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		events:     make(chan *pluginpkg.Event, 100),
		logger:     logger,
	}
}

func (p *Plugin) GetMetadata() (*pluginpkg.Metadata, error) {
	return &pluginpkg.Metadata{
		Name:        "costbreakdown",
		Version:     "0.1.0",
		Description: "Enriched cost breakdown with idle resource detection and savings recommendations",
		Type:        "cost",
		Topics:      []string{"cost.idle.detected"},
		Routes: []pluginpkg.Route{
			{Method: "GET", Path: "/recommendations", Description: "List idle resource recommendations"},
			{Method: "GET", Path: "/summary", Description: "Cost breakdown summary"},
		},
	}, nil
}

func (p *Plugin) Initialize(ctx context.Context, req *pluginpkg.InitRequest) error {
	p.opencostURL = req.OpenCostURL
	go p.pollLoop(ctx)
	return nil
}

func (p *Plugin) Execute(ctx context.Context, req *pluginpkg.ExecuteRequest) (*pluginpkg.ExecuteResponse, error) {
	switch req.Action {
	case "/recommendations":
		return p.handleRecommendations()
	case "/summary":
		return p.handleSummary(ctx)
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
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	p.detectIdle(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.detectIdle(ctx)
		}
	}
}

func (p *Plugin) detectIdle(ctx context.Context) {
	allocations, err := p.fetchAllocations(ctx)
	if err != nil {
		p.logger.Error("failed to fetch allocations", "error", err)
		return
	}

	var recs []Recommendation
	for _, alloc := range allocations {
		cpuReq, _ := alloc["cpuCoreRequestAverage"].(float64)
		cpuUsed, _ := alloc["cpuCoreUsageAverage"].(float64)
		name, _ := alloc["name"].(string)

		if cpuReq > 0 && cpuUsed > 0 && cpuUsed/cpuReq < 0.2 {
			totalCost, _ := alloc["totalCost"].(float64)
			savings := totalCost * (1 - cpuUsed/cpuReq) * 0.8
			severity := "low"
			if savings > 10 {
				severity = "medium"
			}
			if savings > 50 {
				severity = "high"
			}
			recs = append(recs, Recommendation{
				Namespace:    name,
				ResourceType: "cpu",
				Requested:    cpuReq,
				Used:         cpuUsed,
				Savings:      savings,
				Severity:     severity,
			})
		}

		ramReq, _ := alloc["ramByteRequestAverage"].(float64)
		ramUsed, _ := alloc["ramByteUsageAverage"].(float64)

		if ramReq > 0 && ramUsed > 0 && ramUsed/ramReq < 0.2 {
			totalCost, _ := alloc["totalCost"].(float64)
			savings := totalCost * (1 - ramUsed/ramReq) * 0.2
			severity := "low"
			if savings > 10 {
				severity = "medium"
			}
			if savings > 50 {
				severity = "high"
			}
			recs = append(recs, Recommendation{
				Namespace:    name,
				ResourceType: "memory",
				Requested:    ramReq,
				Used:         ramUsed,
				Savings:      savings,
				Severity:     severity,
			})
		}
	}

	p.mu.Lock()
	p.recommendations = recs
	p.mu.Unlock()

	if len(recs) > 0 {
		payload, _ := json.Marshal(map[string]any{
			"count":           len(recs),
			"recommendations": recs,
		})
		select {
		case p.events <- &pluginpkg.Event{
			Type:      "idle_detected",
			Topic:     "cost.idle.detected",
			Timestamp: time.Now().UTC(),
			Source:    "costbreakdown",
			Payload:   payload,
		}:
		default:
		}
	}
}

func (p *Plugin) fetchAllocations(ctx context.Context) ([]map[string]any, error) {
	url := p.opencostURL + "/allocation?window=24h&aggregate=namespace&accumulate=true"
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

	var allocations []map[string]any
	for _, window := range result.Data {
		for _, alloc := range window {
			allocations = append(allocations, alloc)
		}
	}
	return allocations, nil
}

func (p *Plugin) handleRecommendations() (*pluginpkg.ExecuteResponse, error) {
	p.mu.RLock()
	recs := p.recommendations
	p.mu.RUnlock()

	if recs == nil {
		recs = []Recommendation{}
	}

	data, err := json.Marshal(map[string]any{
		"recommendations": recs,
		"count":           len(recs),
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

func (p *Plugin) handleSummary(ctx context.Context) (*pluginpkg.ExecuteResponse, error) {
	url := p.opencostURL + "/allocation?window=24h&aggregate=namespace&accumulate=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return &pluginpkg.ExecuteResponse{Error: err.Error(), StatusCode: http.StatusInternalServerError}, nil
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return &pluginpkg.ExecuteResponse{Error: err.Error(), StatusCode: http.StatusBadGateway}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &pluginpkg.ExecuteResponse{Error: err.Error(), StatusCode: http.StatusInternalServerError}, nil
	}

	return &pluginpkg.ExecuteResponse{
		Data:        body,
		ContentType: "application/json",
		StatusCode:  resp.StatusCode,
	}, nil
}
