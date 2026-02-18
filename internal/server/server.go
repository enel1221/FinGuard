package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	corev1 "k8s.io/api/core/v1"

	"github.com/inelson/finguard/internal/clustercache"
	"github.com/inelson/finguard/internal/config"
	"github.com/inelson/finguard/internal/opencostproxy"
	pluginmgr "github.com/inelson/finguard/internal/plugin"
	"github.com/inelson/finguard/internal/stream"
	"github.com/inelson/finguard/pkg/api"
	"github.com/inelson/finguard/pkg/event"
)

type Server struct {
	cfg       *config.Config
	router    chi.Router
	hub       *stream.Hub
	proxy     *opencostproxy.Proxy
	cache     *clustercache.Cache
	pluginMgr *pluginmgr.Manager
	frontendFS fs.FS
	logger    *slog.Logger
	http      *http.Server
}

func New(cfg *config.Config, hub *stream.Hub, proxy *opencostproxy.Proxy, cc *clustercache.Cache, pm *pluginmgr.Manager, frontendFS fs.FS, logger *slog.Logger) *Server {
	s := &Server{
		cfg:        cfg,
		hub:        hub,
		proxy:      proxy,
		cache:      cc,
		pluginMgr:  pm,
		frontendFS: frontendFS,
		logger:     logger,
	}
	s.router = s.routes()
	s.http = &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return s
}

func (s *Server) Router() chi.Router {
	return s.router
}

func (s *Server) routes() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	r.Get("/healthz", s.handleHealthz)
	r.Get("/readyz", s.handleReadyz)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/stream", s.handleStream)

		// OpenCost proxy endpoints
		r.Get("/allocation", s.proxy.ProxyAllocation)
		r.Get("/assets", s.proxy.ProxyAssets)
		r.Get("/cloudcost", s.proxy.ProxyCloudCost)
		r.Get("/customcost", s.proxy.ProxyCustomCost)

		// Cluster info endpoints
		r.Get("/cluster", s.handleClusterSummary)
		r.Get("/namespaces", s.handleNamespaces)
		r.Get("/nodes", s.handleNodes)
		r.Get("/health", s.handleDetailedHealth)

		// Plugin endpoints
		r.Get("/plugins", s.handleListPlugins)
		if s.pluginMgr != nil {
			s.pluginMgr.MountRoutes(r)
		}
	})

	// Serve embedded frontend SPA
	if s.frontendFS != nil {
		fileServer := http.FileServer(http.FS(s.frontendFS))
		r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
			// Try the exact path; fall back to index.html for SPA routing
			if _, err := fs.Stat(s.frontendFS, req.URL.Path[1:]); err != nil {
				req.URL.Path = "/"
			}
			fileServer.ServeHTTP(w, req)
		})
	}

	return r
}

func (s *Server) Start() error {
	s.logger.Info("starting server", "addr", s.cfg.HTTPAddr)
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down server")
	return s.http.Shutdown(ctx)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	ready := true
	if s.cache != nil && !s.cache.IsReady() {
		ready = false
	}
	if ready {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	} else {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not ready"})
	}
}

func (s *Server) handleDetailedHealth(w http.ResponseWriter, r *http.Request) {
	services := map[string]string{}
	if s.proxy != nil {
		if s.proxy.IsHealthy() {
			services["opencost"] = "healthy"
		} else {
			services["opencost"] = "unreachable"
		}
	}
	if s.cache != nil {
		if s.cache.IsReady() {
			services["cluster_cache"] = "ready"
		} else {
			services["cluster_cache"] = "initializing"
		}
	}
	writeJSON(w, http.StatusOK, api.HealthResponse{
		Status:   "ok",
		Services: services,
	})
}

func (s *Server) handleClusterSummary(w http.ResponseWriter, r *http.Request) {
	if s.cache == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "cluster cache not available"})
		return
	}
	namespaces := s.cache.GetNamespaces()
	nodes := s.cache.GetNodes()
	pods := s.cache.GetPods()

	nsInfos := make([]api.NamespaceInfo, 0, len(namespaces))
	for _, ns := range namespaces {
		nsInfos = append(nsInfos, buildNamespaceInfo(ns))
	}

	nodeInfos := make([]api.NodeInfo, 0, len(nodes))
	for _, n := range nodes {
		nodeInfos = append(nodeInfos, buildNodeInfo(n))
	}

	writeJSON(w, http.StatusOK, api.ClusterSummary{
		NodeCount:      len(nodes),
		PodCount:       len(pods),
		NamespaceCount: len(namespaces),
		Namespaces:     nsInfos,
		Nodes:          nodeInfos,
	})
}

func (s *Server) handleNamespaces(w http.ResponseWriter, r *http.Request) {
	if s.cache == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "cluster cache not available"})
		return
	}
	namespaces := s.cache.GetNamespaces()
	infos := make([]api.NamespaceInfo, 0, len(namespaces))
	for _, ns := range namespaces {
		infos = append(infos, buildNamespaceInfo(ns))
	}
	writeJSON(w, http.StatusOK, infos)
}

func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	if s.cache == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "cluster cache not available"})
		return
	}
	nodes := s.cache.GetNodes()
	infos := make([]api.NodeInfo, 0, len(nodes))
	for _, n := range nodes {
		infos = append(infos, buildNodeInfo(n))
	}
	writeJSON(w, http.StatusOK, infos)
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		s.logger.Error("websocket accept failed", "error", err)
		return
	}

	client := s.hub.Register(r.Context(), conn)
	defer s.hub.Unregister(client)

	welcome, _ := event.New("connected", event.TopicSystem, "finguard", map[string]string{
		"message": "connected to finguard event stream",
	})
	if data, err := json.Marshal(welcome); err == nil {
		client.Conn().Write(client.Context(), websocket.MessageText, data)
	}

	go s.readPump(client)
	s.writePump(client)
}

func (s *Server) readPump(c *stream.Client) {
	defer c.Conn().CloseNow()
	for {
		_, data, err := c.Conn().Read(c.Context())
		if err != nil {
			return
		}
		var msg struct {
			Action string   `json:"action"`
			Topics []string `json:"topics"`
		}
		if json.Unmarshal(data, &msg) == nil && msg.Action == "subscribe" {
			c.Subscribe(msg.Topics...)
			s.logger.Debug("client subscribed", "topics", msg.Topics)
		}
	}
}

func (s *Server) writePump(c *stream.Client) {
	for {
		select {
		case msg, ok := <-c.Send():
			if !ok {
				return
			}
			if err := c.Conn().Write(c.Context(), websocket.MessageText, msg); err != nil {
				return
			}
		case <-c.Context().Done():
			return
		}
	}
}

func (s *Server) handleListPlugins(w http.ResponseWriter, r *http.Request) {
	if s.pluginMgr == nil {
		writeJSON(w, http.StatusOK, map[string]any{"plugins": []any{}})
		return
	}
	plugins := s.pluginMgr.ListPlugins()
	writeJSON(w, http.StatusOK, map[string]any{"plugins": plugins})
}

func buildNamespaceInfo(ns *corev1.Namespace) api.NamespaceInfo {
	return api.NamespaceInfo{
		Name:        ns.Name,
		Labels:      ns.Labels,
		Annotations: ns.Annotations,
		CostCenter:  labelValue(ns.Labels, "cost-center"),
		Team:        labelValue(ns.Labels, "team"),
	}
}

func buildNodeInfo(n *corev1.Node) api.NodeInfo {
	return api.NodeInfo{
		Name:              n.Name,
		Labels:            n.Labels,
		InstanceType:      labelValue(n.Labels, "node.kubernetes.io/instance-type"),
		Region:            labelValue(n.Labels, "topology.kubernetes.io/region"),
		Zone:              labelValue(n.Labels, "topology.kubernetes.io/zone"),
		CapacityCPU:       n.Status.Capacity.Cpu().String(),
		CapacityMemory:    n.Status.Capacity.Memory().String(),
		AllocatableCPU:    n.Status.Allocatable.Cpu().String(),
		AllocatableMemory: n.Status.Allocatable.Memory().String(),
	}
}

func labelValue(labels map[string]string, key string) string {
	if labels == nil {
		return ""
	}
	return labels[key]
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write json response", "error", err)
	}
}
