package models

import (
	"encoding/json"
	"time"
)

type Project struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

type CostSourceType string

const (
	CostSourceAWS        CostSourceType = "aws_account"
	CostSourceAzure      CostSourceType = "azure_subscription"
	CostSourceGCP        CostSourceType = "gcp_project"
	CostSourceKubernetes CostSourceType = "kubernetes"
	CostSourcePlugin     CostSourceType = "plugin"
)

type CostSource struct {
	ID              string          `json:"id" db:"id"`
	ProjectID       string          `json:"projectId" db:"project_id"`
	Type            CostSourceType  `json:"type" db:"type"`
	Name            string          `json:"name" db:"name"`
	Config          json.RawMessage `json:"config" db:"config_json" swaggertype:"object"`
	Enabled         bool            `json:"enabled" db:"enabled"`
	LastCollectedAt *time.Time      `json:"lastCollectedAt,omitempty" db:"last_collected_at"`
	CreatedAt       time.Time       `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time       `json:"updatedAt" db:"updated_at"`
}

type AWSConfig struct {
	AccountID       string `json:"accountId"`
	RoleARN         string `json:"roleArn"`
	ExternalID      string `json:"externalId,omitempty"`
	Region          string `json:"region"`
	AthenaBucket    string `json:"athenaBucket,omitempty"`
	AthenaRegion    string `json:"athenaRegion,omitempty"`
	AthenaDatabase  string `json:"athenaDatabase,omitempty"`
	AthenaTable     string `json:"athenaTable,omitempty"`
	AthenaWorkgroup string `json:"athenaWorkgroup,omitempty"`
	CURVersion      string `json:"curVersion,omitempty"`
}

type AzureConfig struct {
	SubscriptionID   string `json:"subscriptionId"`
	TenantID         string `json:"tenantId"`
	ClientID         string `json:"clientId"`
	ClientSecret     string `json:"clientSecret,omitempty"`
	StorageAccount   string `json:"storageAccount,omitempty"`
	StorageAccessKey string `json:"storageAccessKey,omitempty"`
	StorageContainer string `json:"storageContainer,omitempty"`
	ContainerPath    string `json:"containerPath,omitempty"`
	AzureCloud       string `json:"azureCloud,omitempty"`
}

type GCPConfig struct {
	ProjectID         string `json:"projectId"`
	BillingAccountID  string `json:"billingAccountId,omitempty"`
	BillingDataDataset string `json:"billingDataDataset,omitempty"`
	ServiceAccountKey string `json:"serviceAccountKey,omitempty"`
}

type KubernetesConfig struct {
	ClusterName  string `json:"clusterName"`
	OpenCostURL  string `json:"opencostUrl"`
	KubeconfigRef string `json:"kubeconfigRef,omitempty"`
}

type PluginConfig struct {
	PluginName string          `json:"pluginName"`
	Config     json.RawMessage `json:"config,omitempty" swaggertype:"object"`
}

type User struct {
	ID          string    `json:"id" db:"id"`
	Email       string    `json:"email" db:"email"`
	DisplayName string    `json:"displayName" db:"display_name"`
	OIDCSubject string    `json:"oidcSubject,omitempty" db:"oidc_subject"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
}

type Group struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	OIDCClaim string    `json:"oidcClaim,omitempty" db:"oidc_claim"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

type Role string

const (
	RolePlatformAdmin Role = "platform-admin"
	RoleAdmin         Role = "admin"
	RoleEditor        Role = "editor"
	RoleViewer        Role = "viewer"
)

type SubjectType string

const (
	SubjectUser  SubjectType = "user"
	SubjectGroup SubjectType = "group"
)

type ProjectRole struct {
	ProjectID   string      `json:"projectId" db:"project_id"`
	SubjectType SubjectType `json:"subjectType" db:"subject_type"`
	SubjectID   string      `json:"subjectId" db:"subject_id"`
	Role        Role        `json:"role" db:"role"`
}

type Budget struct {
	ID            string   `json:"id" db:"id"`
	ProjectID     string   `json:"projectId" db:"project_id"`
	CostSourceID  *string  `json:"costSourceId,omitempty" db:"cost_source_id"`
	MonthlyLimit  float64  `json:"monthlyLimit" db:"monthly_limit"`
	WarnThreshold float64  `json:"warnThreshold" db:"warn_threshold"`
	CreatedAt     time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time `json:"updatedAt" db:"updated_at"`
}

type CostRecord struct {
	ID                string            `json:"id" db:"id"`
	ProjectID         string            `json:"projectId" db:"project_id"`
	CostSourceID      string            `json:"costSourceId" db:"cost_source_id"`
	Provider          string            `json:"provider" db:"provider"`
	ProviderID        string            `json:"providerId,omitempty" db:"provider_id"`
	AccountID         string            `json:"accountId,omitempty" db:"account_id"`
	AccountName       string            `json:"accountName,omitempty" db:"account_name"`
	InvoiceEntityID   string            `json:"invoiceEntityId,omitempty" db:"invoice_entity_id"`
	Service           string            `json:"service" db:"service"`
	Category          string            `json:"category" db:"category"`
	Region            string            `json:"region,omitempty" db:"region"`
	AvailabilityZone  string            `json:"availabilityZone,omitempty" db:"availability_zone"`
	StartTime         time.Time         `json:"startTime" db:"start_time"`
	EndTime           time.Time         `json:"endTime" db:"end_time"`
	ListCost          float64           `json:"listCost" db:"list_cost"`
	NetCost           float64           `json:"netCost" db:"net_cost"`
	AmortizedCost     float64           `json:"amortizedCost" db:"amortized_cost"`
	AmortizedNetCost  float64           `json:"amortizedNetCost" db:"amortized_net_cost"`
	Currency          string            `json:"currency" db:"currency"`
	Labels            map[string]string `json:"labels,omitempty"`
	LabelsJSON        string            `json:"-" db:"labels_json"`
	KubernetesPercent float64           `json:"kubernetesPercent,omitempty" db:"kubernetes_percent"`
}
