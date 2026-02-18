package opencostproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Proxy struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
	mu         sync.RWMutex
	healthy    bool
}

func New(baseURL string, logger *slog.Logger) *Proxy {
	return &Proxy{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (p *Proxy) IsHealthy() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.healthy
}

func (p *Proxy) StartHealthCheck(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			healthy := p.checkHealth(ctx)
			p.mu.Lock()
			p.healthy = healthy
			p.mu.Unlock()
		}
	}
}

func (p *Proxy) checkHealth(ctx context.Context) bool {
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, p.baseURL+"/healthz", nil)
	if err != nil {
		return false
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.logger.Debug("opencost health check failed", "error", err)
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ProxyAllocation proxies GET /allocation to OpenCost.
func (p *Proxy) ProxyAllocation(w http.ResponseWriter, r *http.Request) {
	p.proxyRequest(w, r, "/allocation")
}

// ProxyAssets proxies GET /assets to OpenCost.
func (p *Proxy) ProxyAssets(w http.ResponseWriter, r *http.Request) {
	p.proxyRequest(w, r, "/assets")
}

// ProxyCloudCost proxies GET /cloudCost to OpenCost.
func (p *Proxy) ProxyCloudCost(w http.ResponseWriter, r *http.Request) {
	p.proxyRequest(w, r, "/cloudCost")
}

// ProxyCustomCost proxies GET /customCost/total to OpenCost.
func (p *Proxy) ProxyCustomCost(w http.ResponseWriter, r *http.Request) {
	p.proxyRequest(w, r, "/customCost/total")
}

func (p *Proxy) proxyRequest(w http.ResponseWriter, r *http.Request, path string) {
	target, err := url.Parse(p.baseURL + path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid upstream URL")
		return
	}
	target.RawQuery = r.URL.RawQuery

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, target.String(), r.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create proxy request")
		return
	}
	proxyReq.Header = r.Header.Clone()

	resp, err := p.httpClient.Do(proxyReq)
	if err != nil {
		p.logger.Error("opencost proxy request failed", "path", path, "error", err)
		writeError(w, http.StatusBadGateway, fmt.Sprintf("opencost unreachable: %v", err))
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.Header().Set("X-Proxied-By", "finguard")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
