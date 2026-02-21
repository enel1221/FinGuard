package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/inelson/finguard/internal/auth"
	"github.com/inelson/finguard/internal/clustercache"
	"github.com/inelson/finguard/internal/collector"
	collectoraws "github.com/inelson/finguard/internal/collector/aws"
	collectorazure "github.com/inelson/finguard/internal/collector/azure"
	collectorgcp "github.com/inelson/finguard/internal/collector/gcp"
	collectork8s "github.com/inelson/finguard/internal/collector/kubernetes"
	"github.com/inelson/finguard/internal/config"
	"github.com/inelson/finguard/internal/models"
	"github.com/inelson/finguard/internal/opencostproxy"
	pluginmgr "github.com/inelson/finguard/internal/plugin"
	"github.com/inelson/finguard/internal/server"
	"github.com/inelson/finguard/internal/store"
	"github.com/inelson/finguard/internal/stream"
	"github.com/inelson/finguard/migrations"
	"github.com/inelson/finguard/plugins/budgets"
	"github.com/inelson/finguard/plugins/costbreakdown"
	"github.com/inelson/finguard/web"

	_ "github.com/inelson/finguard/docs/swagger"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

// @title           FinGuard API
// @version         1.0
// @description     FinGuard cloud cost management platform API.
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey  SessionAuth
// @in                          cookie
// @name                        finguard_session
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting finguard", "version", version, "commit", commit, "build_time", buildTime)

	cfg := config.Load()
	hub := stream.NewHub(logger)

	db, err := store.New(cfg.DatabaseDSN)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Migrate(migrations.FS); err != nil {
		logger.Error("failed to run database migrations", "error", err)
		os.Exit(1)
	}
	logger.Info("database ready", "dsn", cfg.DatabaseDSN)

	var proxy *opencostproxy.Proxy
	if cfg.DevMode {
		logger.Info("dev mode enabled, using mock OpenCost data")
		proxy = opencostproxy.NewMock(logger)
	} else {
		proxy = opencostproxy.New(cfg.OpenCostURL, logger)
	}

	var cc *clustercache.Cache
	cc, err = clustercache.New(logger)
	if err != nil {
		logger.Warn("cluster cache unavailable, running without k8s integration", "error", err)
		cc = nil
	}

	pm := pluginmgr.NewManager(hub, logger)

	cbPlugin := costbreakdown.New(logger)
	if err := pm.Register(cbPlugin); err != nil {
		logger.Error("failed to register costbreakdown plugin", "error", err)
	}

	budgetPlugin := budgets.New(logger, nil)
	if err := pm.Register(budgetPlugin); err != nil {
		logger.Error("failed to register budgets plugin", "error", err)
	}

	frontendFS, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		logger.Error("failed to load embedded frontend", "error", err)
		frontendFS = nil
	}

	authMgr, err := auth.NewManager(cfg, db, logger)
	if err != nil {
		logger.Error("failed to initialize auth manager", "error", err)
		os.Exit(1)
	}

	// Cost collector registry and scheduler
	collectorRegistry := collector.NewRegistry()
	collectorRegistry.Register(models.CostSourceAWS, collectoraws.New(logger))
	collectorRegistry.Register(models.CostSourceAzure, collectorazure.New(logger))
	collectorRegistry.Register(models.CostSourceGCP, collectorgcp.New(logger))
	collectorRegistry.Register(models.CostSourceKubernetes, collectork8s.New(logger))

	collectorScheduler := collector.NewScheduler(collectorRegistry, db, hub, collector.DefaultSchedulerConfig(), logger)

	srv := server.New(cfg, hub, proxy, cc, pm, db, authMgr, frontendFS, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.DevMode {
		go proxy.StartHealthCheckMock(ctx, 30*time.Second)
	} else {
		go proxy.StartHealthCheck(ctx, 30*time.Second)
	}

	if cc != nil {
		go func() {
			if err := cc.Start(ctx); err != nil {
				logger.Error("cluster cache failed to start", "error", err)
			}
		}()
	}

	go collectorScheduler.Start(ctx)

	if err := pm.InitializeAll(ctx, cfg.OpenCostURL); err != nil {
		logger.Error("failed to initialize plugins", "error", err)
	}

	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	logger.Info("finguard started", "addr", cfg.HTTPAddr)
	<-ctx.Done()
	logger.Info("received shutdown signal")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pm.ShutdownAll(shutdownCtx)

	if cc != nil {
		cc.Stop()
	}

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", "error", err)
	}

	logger.Info("finguard stopped")
}
