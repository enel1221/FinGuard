package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/inelson/finguard/internal/collector"
	"github.com/inelson/finguard/internal/models"
)

// GCPCollector queries GCP Cloud Billing data via BigQuery.
// Follows the same pattern as opencost/pkg/cloud/gcp/bigqueryintegration.go.
type GCPCollector struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *GCPCollector {
	return &GCPCollector{logger: logger}
}

func (c *GCPCollector) Type() string {
	return "gcp"
}

func (c *GCPCollector) Validate(_ context.Context, config json.RawMessage) error {
	var cfg models.GCPConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid GCP config: %w", err)
	}
	if cfg.ProjectID == "" {
		return fmt.Errorf("projectId is required")
	}
	return nil
}

// Collect queries GCP billing export dataset in BigQuery.
// Reference: opencost/pkg/cloud/gcp/bigqueryintegration.go
//
// The query pattern:
//   SELECT usage_start_time, billing_account_id, project.id, location.region, location.zone,
//          service.description, sku.description, resource.name,
//          SUM(cost) as cost, SUM(cost_at_list) as list_cost
//   FROM `{project}.{dataset}.gcp_billing_export_resource_v1_*`
//   WHERE _PARTITIONTIME >= '{start}' AND _PARTITIONTIME < '{end}'
//   GROUP BY 1,2,3,4,5,6,7,8
//
// Handles Flexible CUD credits for amortized cost calculations.
// Uses partition filtering (_PARTITIONTIME) to minimize BigQuery costs.
func (c *GCPCollector) Collect(ctx context.Context, source *models.CostSource, window collector.TimeWindow) ([]*models.CostRecord, error) {
	var cfg models.GCPConfig
	if err := json.Unmarshal(source.Config, &cfg); err != nil {
		return nil, fmt.Errorf("parse GCP config: %w", err)
	}

	c.logger.Info("collecting GCP costs via BigQuery",
		"project", cfg.ProjectID,
		"dataset", cfg.BillingDataDataset,
		"window", window,
	)

	// TODO: Implement actual BigQuery query using cloud.google.com/go/bigquery.
	// Reference: opencost/pkg/cloud/gcp/bigqueryintegration.go
	//
	// Query extracts: usage_date, billing_account_id, project, region, zone,
	//   service, SKU, resource, labels, cost, cost_at_list, credits
	//
	// Flexible CUD amortization is calculated by distributing CUD credits
	// proportionally across eligible resources.

	records := make([]*models.CostRecord, 0)

	c.logger.Warn("GCP BigQuery collection not yet implemented - returning empty results",
		"project", cfg.ProjectID)

	return records, nil
}

func buildBigQuerySQL(cfg models.GCPConfig, window collector.TimeWindow) string {
	startDate := window.Start.Format("2006-01-02")
	endDate := window.End.Format("2006-01-02")

	dataset := cfg.BillingDataDataset
	if dataset == "" {
		dataset = "billing_dataset"
	}

	return fmt.Sprintf(`SELECT
		DATE(usage_start_time) as usage_date,
		billing_account_id,
		project.id as project_id,
		project.name as project_name,
		location.region,
		location.zone,
		service.description as service,
		sku.description as sku,
		resource.name as resource_name,
		SUM(cost) as cost,
		SUM(IFNULL((SELECT SUM(amount) FROM UNNEST(credits)), 0)) as credits,
		SUM(cost_at_list) as list_cost
	FROM %s.%s.gcp_billing_export_resource_v1_*
	WHERE _PARTITIONTIME >= '%s'
	  AND _PARTITIONTIME < '%s'
	  AND cost != 0
	GROUP BY 1,2,3,4,5,6,7,8,9`, cfg.ProjectID, dataset, startDate, endDate)
}

func categorizeGCPService(service string) string {
	switch service {
	case "Compute Engine", "Kubernetes Engine", "Cloud Functions", "Cloud Run", "App Engine":
		return "Compute"
	case "Cloud Storage", "Persistent Disk", "Filestore":
		return "Storage"
	case "Cloud NAT", "Cloud Load Balancing", "Cloud CDN", "Cloud Interconnect", "Cloud VPN":
		return "Network"
	case "Cloud SQL", "Cloud Spanner", "Bigtable", "Firestore", "Memorystore":
		return "Database"
	default:
		return "Other"
	}
}

func bigqueryRowToRecord(source *models.CostSource, date time.Time, row map[string]string, cost, listCost float64) *models.CostRecord {
	return &models.CostRecord{
		ProjectID:        source.ProjectID,
		CostSourceID:     source.ID,
		Provider:         "gcp",
		ProviderID:       row["resource_name"],
		AccountID:        row["project_id"],
		AccountName:      row["project_name"],
		InvoiceEntityID:  row["billing_account_id"],
		Service:          row["service"],
		Category:         categorizeGCPService(row["service"]),
		Region:           row["region"],
		AvailabilityZone: row["zone"],
		StartTime:        date,
		EndTime:          date.Add(24 * time.Hour),
		ListCost:         listCost,
		NetCost:          cost,
		AmortizedCost:    cost,
		Currency:         "USD",
	}
}
