package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/inelson/finguard/internal/collector"
	"github.com/inelson/finguard/internal/models"
)

// AzureCollector reads CSV billing exports from Azure Blob Storage.
// Follows the same pattern as opencost/pkg/cloud/azure/azurestorageintegration.go.
type AzureCollector struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *AzureCollector {
	return &AzureCollector{logger: logger}
}

func (c *AzureCollector) Type() string {
	return "azure"
}

func (c *AzureCollector) Validate(_ context.Context, config json.RawMessage) error {
	var cfg models.AzureConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid Azure config: %w", err)
	}
	if cfg.SubscriptionID == "" {
		return fmt.Errorf("subscriptionId is required")
	}
	if cfg.StorageAccount == "" {
		return fmt.Errorf("storageAccount is required")
	}
	return nil
}

// Collect reads Azure billing export CSVs from Blob Storage.
// Reference: opencost/pkg/cloud/azure/azurestorageintegration.go
//
// Azure billing exports are CSV files with dynamic headers. The collector:
// 1. Lists blobs in the configured container/path
// 2. Downloads CSV files (supports gzip compression)
// 3. Detects schema (PayAsYouGo, Enterprise, Modern) from headers
// 4. Parses rows extracting: Date, MeterCategory, SubscriptionID, Region, InstanceID, Cost
// 5. Normalizes to CostRecord format
//
// Kubernetes resources are detected via tags: aks-managed-*, kubernetes.io-created-*
func (c *AzureCollector) Collect(ctx context.Context, source *models.CostSource, window collector.TimeWindow) ([]*models.CostRecord, error) {
	var cfg models.AzureConfig
	if err := json.Unmarshal(source.Config, &cfg); err != nil {
		return nil, fmt.Errorf("parse Azure config: %w", err)
	}

	c.logger.Info("collecting Azure costs from billing exports",
		"subscription", cfg.SubscriptionID,
		"storageAccount", cfg.StorageAccount,
		"container", cfg.StorageContainer,
		"window", window,
	)

	// TODO: Implement actual Azure Blob Storage CSV parsing using Azure SDK.
	// Reference: opencost/pkg/cloud/azure/azurestorageintegration.go
	//
	// Key fields to extract from CSV:
	//   - Date / UsageDateTime
	//   - MeterCategory
	//   - SubscriptionId / SubscriptionGuid
	//   - ResourceLocation / ResourceRegion
	//   - InstanceId / ResourceId
	//   - ServiceName / MeterName
	//   - Cost / PreTaxCost / CostInBillingCurrency
	//   - Tags (JSON string or key-value pairs)

	records := make([]*models.CostRecord, 0)

	c.logger.Warn("Azure billing export collection not yet implemented - returning empty results",
		"subscription", cfg.SubscriptionID)

	return records, nil
}

func categorizeAzureService(meterCategory string) string {
	switch meterCategory {
	case "Virtual Machines", "Container Instances", "Azure Kubernetes Service", "Functions":
		return "Compute"
	case "Storage", "Bandwidth":
		return "Storage"
	case "Virtual Network", "Load Balancer", "Application Gateway", "VPN Gateway":
		return "Network"
	case "Azure Cosmos DB", "SQL Database", "Azure Database for PostgreSQL", "Azure Cache for Redis":
		return "Database"
	default:
		return "Other"
	}
}

func azureRowToRecord(source *models.CostSource, date time.Time, meterCategory, subscriptionID, region, instanceID string, cost float64) *models.CostRecord {
	return &models.CostRecord{
		ProjectID:    source.ProjectID,
		CostSourceID: source.ID,
		Provider:     "azure",
		ProviderID:   instanceID,
		AccountID:    subscriptionID,
		Service:      meterCategory,
		Category:     categorizeAzureService(meterCategory),
		Region:       region,
		StartTime:    date,
		EndTime:      date.Add(24 * time.Hour),
		ListCost:     cost,
		NetCost:      cost,
		Currency:     "USD",
	}
}
