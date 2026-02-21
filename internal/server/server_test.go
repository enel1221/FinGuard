package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/inelson/finguard/internal/config"
	"github.com/inelson/finguard/internal/opencostproxy"
	"github.com/inelson/finguard/internal/stream"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestServer() *Server {
	cfg := &config.Config{
		HTTPAddr:    ":0",
		OpenCostURL: "http://localhost:9003",
	}
	logger := testLogger()
	hub := stream.NewHub(logger)
	proxy := opencostproxy.New(cfg.OpenCostURL, logger)
	return New(cfg, hub, proxy, nil, nil, nil, nil, nil, logger)
}

func TestHealthz(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", body["status"])
	}
}

func TestReadyz_NoCacheReady(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	// No cache means ready (cache is nil, so considered ready)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestDetailedHealth(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	services, ok := body["services"].(map[string]any)
	if !ok {
		t.Fatal("expected services map in health response")
	}
	if services["opencost"] == nil {
		t.Error("expected opencost service status")
	}
}

func TestClusterEndpoint_NoCache(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cluster", nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when no cache, got %d", w.Code)
	}
}

func TestLabelValue_NilMap(t *testing.T) {
	if v := labelValue(nil, "any-key"); v != "" {
		t.Errorf("expected empty string for nil map, got %q", v)
	}
}

func TestLabelValue_MissingKey(t *testing.T) {
	labels := map[string]string{"a": "1"}
	if v := labelValue(labels, "missing"); v != "" {
		t.Errorf("expected empty string, got %q", v)
	}
}

func TestLabelValue_Present(t *testing.T) {
	labels := map[string]string{"team": "platform"}
	if v := labelValue(labels, "team"); v != "platform" {
		t.Errorf("expected 'platform', got %q", v)
	}
}

func TestPluginsEndpoint_NoManager(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins", nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	plugins, ok := body["plugins"].([]any)
	if !ok {
		t.Fatal("expected plugins array")
	}
	if len(plugins) != 0 {
		t.Errorf("expected empty plugins list, got %d", len(plugins))
	}
}
