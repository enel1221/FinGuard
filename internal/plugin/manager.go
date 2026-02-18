package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/inelson/finguard/internal/stream"
	"github.com/inelson/finguard/pkg/event"
	pluginpkg "github.com/inelson/finguard/pkg/plugin"
)

type Manager struct {
	mu      sync.RWMutex
	plugins map[string]*registeredPlugin
	hub     *stream.Hub
	logger  *slog.Logger
}

type registeredPlugin struct {
	instance pluginpkg.Plugin
	meta     *pluginpkg.Metadata
	cancel   context.CancelFunc
}

func NewManager(hub *stream.Hub, logger *slog.Logger) *Manager {
	return &Manager{
		plugins: make(map[string]*registeredPlugin),
		hub:     hub,
		logger:  logger,
	}
}

// Register adds a compiled-in plugin to the manager.
func (m *Manager) Register(p pluginpkg.Plugin) error {
	meta, err := p.GetMetadata()
	if err != nil {
		return fmt.Errorf("failed to get plugin metadata: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[meta.Name]; exists {
		return fmt.Errorf("plugin %q already registered", meta.Name)
	}

	m.plugins[meta.Name] = &registeredPlugin{
		instance: p,
		meta:     meta,
	}

	m.logger.Info("plugin registered", "name", meta.Name, "version", meta.Version, "type", meta.Type)
	return nil
}

// InitializeAll initializes all registered plugins.
func (m *Manager) InitializeAll(ctx context.Context, opencostURL string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, rp := range m.plugins {
		req := &pluginpkg.InitRequest{
			OpenCostURL: opencostURL,
		}
		if err := rp.instance.Initialize(ctx, req); err != nil {
			m.logger.Error("failed to initialize plugin", "name", name, "error", err)
			continue
		}
		m.logger.Info("plugin initialized", "name", name)

		pctx, cancel := context.WithCancel(ctx)
		rp.cancel = cancel
		go m.bridgeEvents(pctx, name, rp.instance)
	}
	return nil
}

func (m *Manager) bridgeEvents(ctx context.Context, name string, p pluginpkg.Plugin) {
	events, err := p.StreamEvents(ctx)
	if err != nil {
		m.logger.Error("failed to start event stream", "plugin", name, "error", err)
		return
	}
	if events == nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-events:
			if !ok {
				return
			}
			streamEvt := &event.Event{
				Type:      evt.Type,
				Topic:     evt.Topic,
				Timestamp: evt.Timestamp,
				Source:    evt.Source,
				Payload:   evt.Payload,
			}
			m.hub.Publish(streamEvt)
		}
	}
}

// MountRoutes mounts plugin-declared HTTP routes onto the given router under /api/v1/plugins/{name}/.
func (m *Manager) MountRoutes(r chi.Router) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, rp := range m.plugins {
		pluginName := name
		pluginInstance := rp.instance
		routes := rp.meta.Routes

		r.Route("/plugins/"+pluginName, func(sub chi.Router) {
			for _, route := range routes {
				action := route.Path
				switch route.Method {
				case "GET":
					sub.Get(route.Path, makePluginHandler(pluginInstance, action))
				case "POST":
					sub.Post(route.Path, makePluginHandler(pluginInstance, action))
				}
			}
		})

		m.logger.Info("plugin routes mounted", "plugin", pluginName, "routes", len(routes))
	}
}

func makePluginHandler(p pluginpkg.Plugin, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := make(map[string]string)
		for k, v := range r.URL.Query() {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}

		resp, err := p.Execute(r.Context(), &pluginpkg.ExecuteRequest{
			Action: action,
			Params: params,
		})
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		if resp.Error != "" {
			w.Header().Set("Content-Type", "application/json")
			code := resp.StatusCode
			if code == 0 {
				code = http.StatusInternalServerError
			}
			w.WriteHeader(code)
			json.NewEncoder(w).Encode(map[string]string{"error": resp.Error})
			return
		}

		ct := resp.ContentType
		if ct == "" {
			ct = "application/json"
		}
		w.Header().Set("Content-Type", ct)
		code := resp.StatusCode
		if code == 0 {
			code = http.StatusOK
		}
		w.WriteHeader(code)
		w.Write(resp.Data)
	}
}

// ShutdownAll shuts down all plugins.
func (m *Manager) ShutdownAll(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, rp := range m.plugins {
		if rp.cancel != nil {
			rp.cancel()
		}
		if err := rp.instance.Shutdown(ctx); err != nil {
			m.logger.Error("plugin shutdown failed", "name", name, "error", err)
		} else {
			m.logger.Info("plugin shut down", "name", name)
		}
	}
}

// ListPlugins returns metadata for all registered plugins.
func (m *Manager) ListPlugins() []*pluginpkg.Metadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*pluginpkg.Metadata, 0, len(m.plugins))
	for _, rp := range m.plugins {
		result = append(result, rp.meta)
	}
	return result
}
