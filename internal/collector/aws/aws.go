package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/inelson/finguard/internal/collector"
	"github.com/inelson/finguard/internal/models"
)

// AWSCollector queries AWS Cost and Usage Reports via Athena.
// Follows the same pattern as opencost/pkg/cloud/aws/athenaintegration.go.
type AWSCollector struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *AWSCollector {
	return &AWSCollector{logger: logger}
}

func (c *AWSCollector) Type() string {
	return "aws"
}

func (c *AWSCollector) Validate(_ context.Context, config json.RawMessage) error {
	var cfg models.AWSConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid AWS config: %w", err)
	}
	if cfg.AccountID == "" {
		return fmt.Errorf("accountId is required")
	}
	if cfg.RoleARN == "" {
		return fmt.Errorf("roleArn is required")
	}
	return nil
}

// Collect queries AWS CUR data via Athena, similar to OpenCost's AthenaIntegration.
// The query groups by date, resource_id, account, product_code, usage_type, region.
// Supports CUR 1.0 and 2.0 column layouts and dynamic resource tag extraction.
func (c *AWSCollector) Collect(ctx context.Context, source *models.CostSource, window collector.TimeWindow) ([]*models.CostRecord, error) {
	var cfg models.AWSConfig
	if err := json.Unmarshal(source.Config, &cfg); err != nil {
		return nil, fmt.Errorf("parse AWS config: %w", err)
	}

	c.logger.Info("collecting AWS costs via Athena",
		"account", cfg.AccountID,
		"database", cfg.AthenaDatabase,
		"table", cfg.AthenaTable,
		"window", window,
	)

	// TODO: Implement actual Athena query execution using aws-sdk-go-v2/service/athena.
	// Reference: opencost/pkg/cloud/aws/athenaintegration.go
	//
	// The query pattern is:
	//   SELECT line_item_usage_start_date, line_item_resource_id, line_item_usage_account_id,
	//          product_product_name, line_item_usage_type, product_region_code,
	//          SUM(line_item_unblended_cost) as list_cost,
	//          SUM(line_item_net_unblended_cost) as net_cost,
	//          SUM(reservation_effective_cost + savings_plan_savings_plan_effective_cost) as amortized_cost
	//   FROM {database}.{table}
	//   WHERE line_item_usage_start_date >= '{start}' AND line_item_usage_start_date < '{end}'
	//   GROUP BY 1,2,3,4,5,6
	//
	// Supports dynamic tag columns (resource_tags_user_*) for label extraction.
	// Handles CUR 2.0 partition differences (billing_period vs month).

	records := make([]*models.CostRecord, 0)

	c.logger.Warn("AWS Athena collection not yet implemented - returning empty results",
		"account", cfg.AccountID)

	return records, nil
}

func buildAthenaQuery(cfg models.AWSConfig, window collector.TimeWindow) string {
	startDate := window.Start.Format("2006-01-02")
	endDate := window.End.Format("2006-01-02")

	table := cfg.AthenaDatabase + "." + cfg.AthenaTable
	if table == "." {
		table = "cur_database.cur_table"
	}

	return fmt.Sprintf(`SELECT
		DATE(line_item_usage_start_date) as usage_date,
		line_item_resource_id,
		line_item_usage_account_id,
		product_product_name,
		line_item_usage_type,
		product_region_code,
		product_availability_zone,
		SUM(line_item_unblended_cost) as list_cost,
		SUM(COALESCE(line_item_net_unblended_cost, line_item_unblended_cost)) as net_cost,
		SUM(COALESCE(reservation_effective_cost, 0) + COALESCE(savings_plan_savings_plan_effective_cost, 0) + line_item_unblended_cost) as amortized_cost
	FROM %s
	WHERE line_item_usage_start_date >= '%s'
	  AND line_item_usage_start_date < '%s'
	  AND line_item_line_item_type != 'Credit'
	GROUP BY 1,2,3,4,5,6,7`, table, startDate, endDate)
}

func athenaRowToRecord(source *models.CostSource, row map[string]string, usageDate time.Time) *models.CostRecord {
	return &models.CostRecord{
		ProjectID:     source.ProjectID,
		CostSourceID:  source.ID,
		Provider:      "aws",
		ProviderID:    row["line_item_resource_id"],
		AccountID:     row["line_item_usage_account_id"],
		Service:       row["product_product_name"],
		Category:      categorizeAWSService(row["product_product_name"], row["line_item_usage_type"]),
		Region:        row["product_region_code"],
		AvailabilityZone: row["product_availability_zone"],
		StartTime:     usageDate,
		EndTime:       usageDate.Add(24 * time.Hour),
		Currency:      "USD",
	}
}

func categorizeAWSService(product, usageType string) string {
	switch {
	case product == "AmazonEC2" || product == "AmazonEKS" || product == "AWSLambda":
		return "Compute"
	case product == "AmazonS3" || product == "AmazonEBS" || product == "AmazonEFS":
		return "Storage"
	case product == "AmazonVPC" || product == "AmazonCloudFront" || product == "AWSDataTransfer":
		return "Network"
	case product == "AmazonRDS" || product == "AmazonDynamoDB" || product == "AmazonElastiCache":
		return "Database"
	default:
		return "Other"
	}
}
