package opencostproxy

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestProxy_ProxyAllocation(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/allocation" {
			t.Errorf("expected path /allocation, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("window") != "24h" {
			t.Errorf("expected window=24h, got %s", r.URL.Query().Get("window"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": []any{}})
	}))
	defer upstream.Close()

	proxy := New(upstream.URL, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/allocation?window=24h", nil)
	w := httptest.NewRecorder()

	proxy.ProxyAllocation(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("X-Proxied-By") != "finguard" {
		t.Error("missing X-Proxied-By header")
	}
}

func TestProxy_ProxyAssets(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/assets" {
			t.Errorf("expected path /assets, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":[]}`))
	}))
	defer upstream.Close()

	proxy := New(upstream.URL, testLogger())
	req := httptest.NewRequest(http.MethodGet, "/assets", nil)
	w := httptest.NewRecorder()

	proxy.ProxyAssets(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestProxy_UpstreamDown(t *testing.T) {
	proxy := New("http://127.0.0.1:1", testLogger())

	req := httptest.NewRequest(http.MethodGet, "/allocation", nil)
	w := httptest.NewRecorder()

	proxy.ProxyAllocation(w, req)
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestProxy_IsHealthy(t *testing.T) {
	proxy := New("http://invalid", testLogger())
	if proxy.IsHealthy() {
		t.Error("expected unhealthy before any check")
	}
}

func TestProxy_HealthCheckSuccess(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer upstream.Close()

	proxy := New(upstream.URL, testLogger())
	healthy := proxy.checkHealth(httptest.NewRequest(http.MethodGet, "/", nil).Context())
	if !healthy {
		t.Error("expected healthy upstream")
	}
}
