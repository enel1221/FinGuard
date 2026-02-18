package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/inelson/finguard/internal/collector"
	"github.com/inelson/finguard/internal/models"
)

// KubernetesCollector wraps OpenCost's allocation API and normalizes the response into CostRecords.
type KubernetesCollector struct {
	httpClient *http.Client
	logger     *slog.Logger
}

func New(logger *slog.Logger) *KubernetesCollector {
	return &KubernetesCollector{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
	}
}

func (c *KubernetesCollector) Type() string {
	return "kubernetes"
}

func (c *KubernetesCollector) Validate(_ context.Context, config json.RawMessage) error {
	var cfg models.KubernetesConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid Kubernetes config: %w", err)
	}
	if cfg.ClusterName == "" {
		return fmt.Errorf("clusterName is required")
	}
	if cfg.OpenCostURL == "" {
		return fmt.Errorf("opencostUrl is required")
	}
	return nil
}

// Collect queries the OpenCost allocation API and normalizes the response.
func (c *KubernetesCollector) Collect(ctx context.Context, source *models.CostSource, window collector.TimeWindow) ([]*models.CostRecord, error) {
	var cfg models.KubernetesConfig
	if err := json.Unmarshal(source.Config, &cfg); err != nil {
		return nil, fmt.Errorf("parse Kubernetes config: %w", err)
	}

	c.logger.Info("collecting Kubernetes costs from OpenCost",
		"cluster", cfg.ClusterName,
		"opencostUrl", cfg.OpenCostURL,
		"window", window,
	)

	allocationURL := fmt.Sprintf("%s/allocation?window=%s,%s&aggregate=namespace&accumulate=true",
		cfg.OpenCostURL,
		window.Start.Format(time.RFC3339),
		window.End.Format(time.RFC3339),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, allocationURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("opencost allocation request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("opencost returned %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Code int                        `json:"code"`
		Data []map[string]allocationData `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode allocation response: %w", err)
	}

	var records []*models.CostRecord
	for _, dataSet := range apiResp.Data {
		for namespace, alloc := range dataSet {
			record := &models.CostRecord{
				ProjectID:    source.ProjectID,
				CostSourceID: source.ID,
				Provider:     "kubernetes",
				ProviderID:   cfg.ClusterName + "/" + namespace,
				AccountID:    cfg.ClusterName,
				AccountName:  cfg.ClusterName,
				Service:      "Kubernetes/" + namespace,
				Category:     "Compute",
				Region:       "",
				StartTime:    window.Start,
				EndTime:      window.End,
				ListCost:     alloc.TotalCost,
				NetCost:      alloc.TotalCost,
				AmortizedCost: alloc.TotalCost,
				Currency:     "USD",
				Labels: map[string]string{
					"namespace": namespace,
					"cluster":   cfg.ClusterName,
				},
				KubernetesPercent: 1.0,
			}
			records = append(records, record)
		}
	}

	return records, nil
}

type allocationData struct {
	Name      string  `json:"name"`
	Start     string  `json:"start"`
	End       string  `json:"end"`
	CPUCost   float64 `json:"cpuCost"`
	RAMCost   float64 `json:"ramCost"`
	GPUCost   float64 `json:"gpuCost"`
	TotalCost float64 `json:"totalCost"`
}
