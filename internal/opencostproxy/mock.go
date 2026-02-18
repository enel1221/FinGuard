package opencostproxy

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"time"
)

type MockProxy struct {
	logger *slog.Logger
}

func NewMock(logger *slog.Logger) *Proxy {
	logger.Info("starting in dev mode with mock OpenCost data")
	mock := &MockProxy{logger: logger}
	p := &Proxy{
		baseURL: "http://mock-opencost:9003",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger:  logger,
		healthy: true,
	}
	p.mock = mock
	return p
}

func (m *MockProxy) handleAllocation(w http.ResponseWriter, _ *http.Request) {
	now := time.Now().UTC()
	start := now.Add(-24 * time.Hour)

	namespaces := []string{"default", "kube-system", "monitoring", "production", "staging"}
	allocations := map[string]any{}

	for _, ns := range namespaces {
		cpuCost := roundTo(rand.Float64()*50+5, 2)
		ramCost := roundTo(rand.Float64()*30+3, 2)
		gpuCost := 0.0
		if ns == "production" {
			gpuCost = roundTo(rand.Float64()*100, 2)
		}
		totalCost := roundTo(cpuCost+ramCost+gpuCost, 2)

		allocations[ns] = map[string]any{
			"name":     ns,
			"start":    start.Format(time.RFC3339),
			"end":      now.Format(time.RFC3339),
			"cpuCost":  cpuCost,
			"ramCost":  ramCost,
			"gpuCost":  gpuCost,
			"totalCost": totalCost,
			"cpuCoreRequestAverage": roundTo(rand.Float64()*4+0.5, 3),
			"ramByteRequestAverage": roundTo(rand.Float64()*4e9+5e8, 0),
			"cpuCoreUsageAverage":   roundTo(rand.Float64()*2+0.1, 3),
			"ramByteUsageAverage":   roundTo(rand.Float64()*2e9+1e8, 0),
		}
	}

	mockWriteJSON(w, http.StatusOK, map[string]any{
		"code": 200,
		"data": []any{allocations},
	})
}

func (m *MockProxy) handleAssets(w http.ResponseWriter, _ *http.Request) {
	now := time.Now().UTC()
	start := now.Add(-24 * time.Hour)

	nodes := []map[string]any{
		{
			"type":       "Node",
			"name":       "ip-10-0-1-101.ec2.internal",
			"start":      start.Format(time.RFC3339),
			"end":        now.Format(time.RFC3339),
			"nodeType":   "m5.xlarge",
			"cpuCost":    roundTo(rand.Float64()*20+10, 2),
			"ramCost":    roundTo(rand.Float64()*10+5, 2),
			"totalCost":  roundTo(rand.Float64()*35+15, 2),
			"provider":   "AWS",
			"providerID": "i-0abc123def456",
		},
		{
			"type":       "Node",
			"name":       "ip-10-0-2-202.ec2.internal",
			"start":      start.Format(time.RFC3339),
			"end":        now.Format(time.RFC3339),
			"nodeType":   "m5.2xlarge",
			"cpuCost":    roundTo(rand.Float64()*40+20, 2),
			"ramCost":    roundTo(rand.Float64()*20+10, 2),
			"totalCost":  roundTo(rand.Float64()*70+30, 2),
			"provider":   "AWS",
			"providerID": "i-0def789abc012",
		},
	}

	mockWriteJSON(w, http.StatusOK, map[string]any{
		"code": 200,
		"data": []any{nodes},
	})
}

func (m *MockProxy) handleCloudCost(w http.ResponseWriter, _ *http.Request) {
	now := time.Now().UTC()
	start := now.Add(-24 * time.Hour)
	dateKey := start.Format("2006-01-02T15:04:05Z")

	services := []struct {
		provider, service, account string
		cost                       float64
	}{
		{"AWS", "AmazonEC2", "123456789012", roundTo(rand.Float64()*500+100, 2)},
		{"AWS", "AmazonS3", "123456789012", roundTo(rand.Float64()*50+10, 2)},
		{"AWS", "AmazonRDS", "123456789012", roundTo(rand.Float64()*200+50, 2)},
		{"AWS", "AmazonEKS", "123456789012", roundTo(rand.Float64()*100+20, 2)},
		{"Azure", "Virtual Machines", "sub-abc-123", roundTo(rand.Float64()*300+80, 2)},
		{"Azure", "Storage", "sub-abc-123", roundTo(rand.Float64()*40+8, 2)},
		{"GCP", "Compute Engine", "my-gcp-project", roundTo(rand.Float64()*250+60, 2)},
		{"GCP", "Cloud Storage", "my-gcp-project", roundTo(rand.Float64()*30+5, 2)},
	}

	cloudCosts := map[string]any{}
	for _, svc := range services {
		key := svc.provider + "/" + svc.account + "/" + svc.service
		cloudCosts[key] = map[string]any{
			"properties": map[string]any{
				"provider":    svc.provider,
				"accountID":   svc.account,
				"service":     svc.service,
				"category":    "Compute",
				"invoiceEntityID": svc.account,
			},
			"listCost":        map[string]any{"cost": svc.cost},
			"netCost":         map[string]any{"cost": roundTo(svc.cost*0.9, 2)},
			"amortizedCost":   map[string]any{"cost": roundTo(svc.cost*0.85, 2)},
			"invoicedCost":    map[string]any{"cost": roundTo(svc.cost*0.88, 2)},
		}
	}

	mockWriteJSON(w, http.StatusOK, map[string]any{
		"code": 200,
		"data": map[string]any{
			"sets": []any{
				map[string]any{
					"cloudCosts": map[string]any{dateKey: cloudCosts},
				},
			},
		},
	})
}

func (m *MockProxy) handleCustomCost(w http.ResponseWriter, _ *http.Request) {
	mockWriteJSON(w, http.StatusOK, map[string]any{
		"code": 200,
		"data": map[string]any{
			"sets": []any{},
		},
	})
}

func roundTo(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return math.Round(f*shift) / shift
}

func mockWriteJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// StartHealthCheckMock is a no-op for mock mode since it's always healthy.
func (p *Proxy) StartHealthCheckMock(_ context.Context, _ time.Duration) {}
