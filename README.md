# FinGuard

A modular FinOps tool for Kubernetes cost management. FinGuard deploys alongside [OpenCost](https://opencost.io) as an intelligent proxy layer, adding a Go backend plugin system, real-time WebSocket streaming, and a dashboard for cost visibility, budget tracking, and optimization recommendations.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     FinGuard Core                           │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐  │
│  │ HTTP/WS  │  │ OpenCost │  │ Cluster  │  │  Plugin    │  │
│  │ Server   │  │  Proxy   │  │  Cache   │  │  Manager   │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └─────┬──────┘  │
│       │              │             │               │         │
│  ┌────┴──────────────┴─────────────┴───────────────┴──────┐  │
│  │                 Streaming Event Hub                     │  │
│  └────────────────────────┬────────────────────────────────┘  │
└───────────────────────────┼──────────────────────────────────┘
                            │ WebSocket
                    ┌───────┴───────┐
                    │   Dashboard   │
                    └───────────────┘

External:  OpenCost ◄── Prometheus ◄── Kubernetes API
           ▲                              ▲
           └── CSP Billing APIs           └── FinGuard Cluster Cache
```

## Features

- **OpenCost Integration**: Proxies OpenCost's allocation, asset, cloud cost, and custom cost APIs with enrichment
- **Go Backend Plugin System**: Plugins run as separate processes communicating via gRPC (HashiCorp go-plugin pattern), with streaming event support
- **Real-time Streaming**: WebSocket event hub pushes cost alerts, budget breaches, and cluster changes to the frontend
- **Budget Tracking**: Per-namespace budget enforcement with warning and exceeded alerts
- **Idle Resource Detection**: Identifies underutilized workloads with savings recommendations
- **Cluster Context**: Supplemental Kubernetes resource cache for org hierarchy (team labels, cost centers)
- **Helm Deployable**: Production-ready Helm chart with RBAC, health probes, and resource limits

## Quick Start

### Prerequisites

- Go 1.23+
- A Kubernetes cluster (or `kind` for local development)
- OpenCost deployed in the cluster
- Helm 3 (for deployment)

### Build

```bash
make build          # Build the binary
make test           # Run tests
make lint           # Run linter
make docker-build   # Build Docker image
make helm-lint      # Lint Helm chart
```

### Run Locally

```bash
# Set OpenCost URL (if not using default)
export OPENCOST_URL=http://localhost:9003

# Run the server
make run
# or
go run ./cmd/finguard

# Open http://localhost:8080
```

### Deploy with Helm

```bash
helm install finguard deploy/helm/finguard/ \
  --set opencost.url=http://opencost.opencost.svc.cluster.local:9003
```

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /healthz` | Liveness probe |
| `GET /readyz` | Readiness probe |
| `GET /api/v1/health` | Detailed health with service status |
| `GET /api/v1/allocation` | Cost allocation (proxied from OpenCost) |
| `GET /api/v1/assets` | Asset costs (proxied from OpenCost) |
| `GET /api/v1/cloudcost` | Cloud costs (proxied from OpenCost) |
| `GET /api/v1/customcost` | Custom costs (proxied from OpenCost) |
| `GET /api/v1/cluster` | Cluster summary (nodes, namespaces, pods) |
| `GET /api/v1/namespaces` | Namespace list with labels |
| `GET /api/v1/nodes` | Node list with capacity info |
| `GET /api/v1/plugins` | List registered plugins |
| `GET /api/v1/plugins/costbreakdown/recommendations` | Idle resource recommendations |
| `GET /api/v1/plugins/costbreakdown/summary` | Cost breakdown summary |
| `GET /api/v1/plugins/budgets/status` | Budget status per namespace |
| `GET /api/v1/plugins/budgets/config` | Budget configuration |
| `WS /api/v1/stream` | WebSocket event stream |

## Plugin System

FinGuard plugins implement the `plugin.Plugin` interface:

```go
type Plugin interface {
    GetMetadata() (*Metadata, error)
    Initialize(ctx context.Context, req *InitRequest) error
    Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error)
    StreamEvents(ctx context.Context) (<-chan *Event, error)
    Shutdown(ctx context.Context) error
}
```

Plugins can:
- Declare HTTP routes mounted at `/api/v1/plugins/{name}/`
- Publish events to the streaming hub (forwarded to WebSocket clients)
- Query OpenCost data via the provided client
- Access cluster cache for Kubernetes context

### Built-in Plugins

- **costbreakdown**: Detects idle resources, calculates savings recommendations
- **budgets**: Tracks namespace spend against configured budgets, alerts on threshold breaches

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `FINGUARD_ADDR` | `:8080` | HTTP listen address |
| `OPENCOST_URL` | `http://opencost.opencost.svc.cluster.local:9003` | OpenCost API URL |
| `FINGUARD_PLUGIN_DIR` | `/opt/finguard/plugins/bin` | Plugin binary directory |
| `FINGUARD_PLUGIN_CONFIG_DIR` | `/opt/finguard/plugins/config` | Plugin config directory |
| `FINGUARD_LOG_LEVEL` | `info` | Log level |

## Project Structure

```
cmd/finguard/           Server entry point
internal/
  server/               HTTP/WS server, routes, middleware
  stream/               WebSocket event hub (pub/sub)
  opencostproxy/        OpenCost API proxy with enrichment
  clustercache/         Kubernetes resource watcher (client-go informers)
  plugin/               Plugin manager (discovery, lifecycle, event bridge)
  config/               Configuration loading
pkg/
  event/                Shared event types
  api/                  API request/response types
  plugin/               Plugin interface definitions
plugins/
  costbreakdown/        Idle resource detection plugin
  budgets/              Budget tracking plugin
web/                    Embedded frontend dashboard
deploy/helm/finguard/   Helm chart
```

## OSS References

This project uses the following open-source projects as architectural references (included as git submodules):

- **[OpenCost](https://github.com/opencost/opencost)**: Cost engine, CSP billing integrations, plugin system patterns
- **[Headlamp](https://github.com/kubernetes-sigs/headlamp)**: Helm chart structure, frontend serving patterns
- **[Argo CD](https://github.com/argoproj/argo-cd)**: Go project layout, build system patterns

```bash
git clone --recurse-submodules https://github.com/inelson/finguard.git
```

## License

GNU General Public License v3.0
