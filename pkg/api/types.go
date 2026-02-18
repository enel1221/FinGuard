package api

type HealthResponse struct {
	Status   string            `json:"status"`
	Version  string            `json:"version,omitempty"`
	Services map[string]string `json:"services,omitempty"`
}

type NamespaceInfo struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CostCenter  string            `json:"costCenter,omitempty"`
	Team        string            `json:"team,omitempty"`
}

type NodeInfo struct {
	Name             string            `json:"name"`
	Labels           map[string]string `json:"labels,omitempty"`
	InstanceType     string            `json:"instanceType,omitempty"`
	Region           string            `json:"region,omitempty"`
	Zone             string            `json:"zone,omitempty"`
	CapacityCPU      string            `json:"capacityCPU"`
	CapacityMemory   string            `json:"capacityMemory"`
	AllocatableCPU   string            `json:"allocatableCPU"`
	AllocatableMemory string           `json:"allocatableMemory"`
}

type ClusterSummary struct {
	NodeCount      int              `json:"nodeCount"`
	PodCount       int              `json:"podCount"`
	NamespaceCount int              `json:"namespaceCount"`
	Namespaces     []NamespaceInfo  `json:"namespaces"`
	Nodes          []NodeInfo       `json:"nodes"`
}
