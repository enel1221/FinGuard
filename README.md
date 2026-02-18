# FinGuard

A modular FinOps platform for multi-cloud cost management. FinGuard supports AWS, Azure, GCP, and Kubernetes cost tracking with project-based organization, OIDC authentication via Dex, real-time WebSocket streaming, and a plugin system for extensibility.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        FinGuard Core                            │
│                                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────────┐  │
│  │ HTTP/WS  │  │   Cost   │  │  Cluster │  │    Plugin      │  │
│  │ Server   │  │Collectors│  │  Cache   │  │    Manager     │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └──────┬─────────┘  │
│       │              │             │                │            │
│  ┌────┴──────────────┴─────────────┴────────────────┴────────┐  │
│  │              Streaming Event Hub + Store (SQLite/PG)       │  │
│  └────────────────────────────┬───────────────────────────────┘  │
└───────────────────────────────┼─────────────────────────────────┘
                                │ WebSocket
                        ┌───────┴───────┐
                        │   React SPA   │
                        └───────────────┘

External:  OpenCost ◄── Prometheus ◄── Kubernetes API
           AWS Cost Explorer (Athena/CUR)
           Azure Cost Management (Blob Storage)
           GCP Cloud Billing (BigQuery)
```

## Features

- **Multi-Cloud Cost Tracking**: AWS, Azure, GCP, and Kubernetes cost collection out of the box
- **Project-Based Organization**: Kion-inspired project structure grouping cost sources, members, and budgets
- **OIDC Authentication**: Dex-based SSO supporting GitHub, Google, Okta, Azure AD, LDAP, SAML, and more
- **Role-Based Access Control**: Per-project roles (admin, editor, viewer) with group-based assignment
- **Go Backend Plugin System**: Extensible plugin architecture with gRPC support for out-of-process plugins
- **Real-time Streaming**: WebSocket event hub pushes cost alerts, budget breaches, and cluster changes
- **Budget Tracking**: Per-project and per-source budget enforcement with alerts
- **Idle Resource Detection**: Identifies underutilized workloads with savings recommendations
- **Helm Deployable**: Production-ready Helm chart with RBAC, OIDC, health probes, and persistence

## Quick Start

### Prerequisites

- Go 1.23+
- Node.js 20+ (for frontend development)
- A Kubernetes cluster (or `kind` for local development)
- Helm 3 (for deployment)

### Build

```bash
make build          # Build the Go binary
make frontend       # Build the React frontend
make test           # Run tests
make lint           # Run linter
make docker-build   # Build Docker image
make helm-lint      # Lint Helm chart
```

### Run Locally (Dev Mode)

```bash
# Dev mode uses mock OpenCost data and disables auth
make dev

# Open http://localhost:8080
```

### Run with Real Data

```bash
export OPENCOST_URL=http://localhost:9003
export FINGUARD_DB_DSN=sqlite:///tmp/finguard.db
make run

# Open http://localhost:8080
```

### Deploy with Helm

```bash
helm install finguard deploy/helm/finguard/ \
  --set opencost.url=http://opencost.opencost.svc.cluster.local:9003 \
  --set auth.oidc.issuerURL=https://dex.example.com \
  --set auth.oidc.clientID=finguard \
  --set auth.oidc.redirectURL=https://finguard.example.com/callback
```

## Authentication

FinGuard uses [Dex](https://dexidp.io/) as an OIDC identity broker, following the same pattern as ArgoCD. Dex handles the fan-out to any upstream identity provider, so FinGuard only implements a single OIDC client flow.

```
Browser → FinGuard → Dex → Upstream IdP (GitHub, Google, Okta, etc.)
```

### Helm OIDC Configuration

```yaml
auth:
  enabled: true
  oidc:
    issuerURL: "https://dex.example.com"
    clientID: "finguard"
    clientSecret: "your-client-secret"
    redirectURL: "https://finguard.example.com/callback"
    scopes: [openid, profile, email, groups]
    sessionSecret: "32-byte-random-secret-here"
```

### Deploying Dex as a Sidecar

Set `dex.enabled: true` in your Helm values to deploy Dex alongside FinGuard:

```yaml
dex:
  enabled: true
  config:
    issuer: "https://dex.example.com"
    connectors:
      - type: github
        id: github
        name: GitHub
        config:
          clientID: "$GITHUB_CLIENT_ID"
          clientSecret: "$GITHUB_CLIENT_SECRET"
          redirectURI: "https://dex.example.com/callback"
    staticClients:
      - id: finguard
        name: FinGuard
        redirectURIs: ["https://finguard.example.com/callback"]
        secret: "your-client-secret"
```

### Example Dex Connectors

**GitHub:**
```yaml
connectors:
  - type: github
    id: github
    name: GitHub
    config:
      clientID: "$GITHUB_CLIENT_ID"
      clientSecret: "$GITHUB_CLIENT_SECRET"
      redirectURI: "https://dex.example.com/callback"
      orgs:
        - name: your-org
```

**Google:**
```yaml
connectors:
  - type: oidc
    id: google
    name: Google
    config:
      issuer: "https://accounts.google.com"
      clientID: "$GOOGLE_CLIENT_ID"
      clientSecret: "$GOOGLE_CLIENT_SECRET"
      redirectURI: "https://dex.example.com/callback"
```

**Azure AD:**
```yaml
connectors:
  - type: microsoft
    id: azure
    name: Azure AD
    config:
      clientID: "$AZURE_CLIENT_ID"
      clientSecret: "$AZURE_CLIENT_SECRET"
      redirectURI: "https://dex.example.com/callback"
      tenant: "your-tenant-id"
```

**Okta:**
```yaml
connectors:
  - type: oidc
    id: okta
    name: Okta
    config:
      issuer: "https://your-org.okta.com"
      clientID: "$OKTA_CLIENT_ID"
      clientSecret: "$OKTA_CLIENT_SECRET"
      redirectURI: "https://dex.example.com/callback"
```

### Local Development with Static User

Use `docker-compose.dev.yml` which includes a pre-configured Dex with a static user:

```bash
docker compose -f docker-compose.dev.yml up
# Login with: admin@finguard.local / password
```

Or disable auth entirely for local development:
```bash
make dev  # sets FINGUARD_AUTH_DISABLED=true
```

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /healthz` | Liveness probe |
| `GET /readyz` | Readiness probe |
| `GET /login` | Initiate OIDC login |
| `GET /callback` | OIDC callback |
| `GET /logout` | Clear session |
| `GET /api/v1/me` | Current user info |
| `GET /api/v1/health` | Detailed health with service status |
| `POST /api/v1/projects` | Create project |
| `GET /api/v1/projects` | List projects |
| `GET /api/v1/projects/{id}` | Get project |
| `PUT /api/v1/projects/{id}` | Update project |
| `DELETE /api/v1/projects/{id}` | Delete project |
| `POST /api/v1/projects/{id}/sources` | Add cost source |
| `GET /api/v1/projects/{id}/sources` | List cost sources |
| `DELETE /api/v1/projects/{id}/sources/{sid}` | Remove cost source |
| `GET /api/v1/projects/{id}/costs` | Aggregated project costs |
| `POST /api/v1/projects/{id}/members` | Add project member |
| `GET /api/v1/projects/{id}/members` | List project members |
| `DELETE /api/v1/projects/{id}/members/{sid}` | Remove member |
| `GET /api/v1/allocation` | Cost allocation (OpenCost proxy) |
| `GET /api/v1/assets` | Asset costs (OpenCost proxy) |
| `GET /api/v1/cloudcost` | Cloud costs (OpenCost proxy) |
| `GET /api/v1/customcost` | Custom costs (OpenCost proxy) |
| `GET /api/v1/cluster` | Cluster summary |
| `GET /api/v1/plugins` | List plugins |
| `WS /api/v1/stream` | WebSocket event stream |

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `FINGUARD_ADDR` | `:8080` | HTTP listen address |
| `FINGUARD_DB_DSN` | `sqlite:///tmp/finguard.db` | Database DSN (sqlite:// or postgres://) |
| `FINGUARD_DEV_MODE` | `false` | Enable mock data for development |
| `FINGUARD_AUTH_DISABLED` | `false` | Disable authentication |
| `FINGUARD_OIDC_ISSUER` | | OIDC issuer URL (Dex) |
| `FINGUARD_OIDC_CLIENT_ID` | `finguard` | OIDC client ID |
| `FINGUARD_OIDC_CLIENT_SECRET` | | OIDC client secret |
| `FINGUARD_OIDC_REDIRECT_URL` | | OIDC redirect URL |
| `FINGUARD_OIDC_SCOPES` | `openid,profile,email,groups` | OIDC scopes |
| `FINGUARD_SESSION_SECRET` | | Session cookie encryption key |
| `OPENCOST_URL` | `http://opencost...svc:9003` | OpenCost API URL |
| `FINGUARD_PLUGIN_DIR` | `/opt/finguard/plugins/bin` | Plugin binary directory |
| `FINGUARD_PLUGIN_CONFIG_DIR` | `/opt/finguard/plugins/config` | Plugin config directory |
| `FINGUARD_LOG_LEVEL` | `info` | Log level |

## Project Structure

```
cmd/finguard/              Server entry point
internal/
  auth/                    OIDC authentication, sessions, RBAC
  server/                  HTTP/WS server, routes, middleware
  store/                   Database layer (SQLite/PostgreSQL)
  stream/                  WebSocket event hub (pub/sub)
  opencostproxy/           OpenCost API proxy (+ mock for dev)
  clustercache/            Kubernetes resource watcher
  plugin/                  Plugin manager
  config/                  Configuration loading
  models/                  Domain models (Project, CostSource, User, etc.)
  collector/               CSP cost collectors (AWS, Azure, GCP, K8s)
pkg/
  event/                   Shared event types
  api/                     API request/response types
  plugin/                  Plugin interface definitions
plugins/
  costbreakdown/           Idle resource detection plugin
  budgets/                 Budget tracking plugin
migrations/                SQL migration files (auto-applied on startup)
web/
  frontend/                React SPA (Vite + MUI)
  dist/                    Built frontend (Go-embedded)
deploy/helm/finguard/      Helm chart
configs/                   Default configuration files
```

## OSS References

This project uses the following open-source projects as architectural references (included as git submodules):

- **[OpenCost](https://github.com/opencost/opencost)**: CSP billing integrations (Athena, Azure Storage, BigQuery), cost model
- **[Headlamp](https://github.com/kubernetes-sigs/headlamp)**: Frontend plugin system, theming architecture
- **[Argo CD](https://github.com/argoproj/argo-cd)**: Dex OIDC pattern, Go project layout

```bash
git clone --recurse-submodules https://github.com/inelson/finguard.git
```

## License

GNU General Public License v3.0
