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

	"github.com/inelson/finguard/internal/clustercache"
	"github.com/inelson/finguard/internal/config"
	"github.com/inelson/finguard/internal/opencostproxy"
	pluginmgr "github.com/inelson/finguard/internal/plugin"
	"github.com/inelson/finguard/internal/server"
	"github.com/inelson/finguard/internal/stream"
	"github.com/inelson/finguard/plugins/budgets"
	"github.com/inelson/finguard/plugins/costbreakdown"
	"github.com/inelson/finguard/web"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting finguard", "version", version, "commit", commit, "build_time", buildTime)

	cfg := config.Load()
	hub := stream.NewHub(logger)
	proxy := opencostproxy.New(cfg.OpenCostURL, logger)

	var cc *clustercache.Cache
	cc, err := clustercache.New(logger)
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

	srv := server.New(cfg, hub, proxy, cc, pm, frontendFS, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go proxy.StartHealthCheck(ctx, 30*time.Second)

	if cc != nil {
		go func() {
			if err := cc.Start(ctx); err != nil {
				logger.Error("cluster cache failed to start", "error", err)
			}
		}()
	}

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
