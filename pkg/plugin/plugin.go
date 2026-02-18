package plugin

import (
	"context"
	"time"
)

type Metadata struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Type        string   `json:"type"` // "cost", "policy", "security", "governance"
	Topics      []string `json:"topics"`
	Routes      []Route  `json:"routes"`
}

type Route struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

type InitRequest struct {
	Config     []byte `json:"config"`
	OpenCostURL string `json:"opencostUrl"`
}

type ExecuteRequest struct {
	Action string            `json:"action"`
	Params map[string]string `json:"params"`
}

type ExecuteResponse struct {
	Data        []byte `json:"data"`
	ContentType string `json:"contentType"`
	StatusCode  int    `json:"statusCode"`
	Error       string `json:"error,omitempty"`
}

type Event struct {
	Type      string    `json:"type"`
	Topic     string    `json:"topic"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
	Payload   []byte    `json:"payload"`
}

// Plugin is the interface all FinGuard plugins must implement.
// For out-of-process plugins, this is bridged via gRPC using HashiCorp go-plugin.
// For compiled-in plugins, implement this interface directly and register with the manager.
type Plugin interface {
	GetMetadata() (*Metadata, error)
	Initialize(ctx context.Context, req *InitRequest) error
	Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error)
	StreamEvents(ctx context.Context) (<-chan *Event, error)
	Shutdown(ctx context.Context) error
}
