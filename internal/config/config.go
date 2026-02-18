package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr       string
	OpenCostURL    string
	PluginDir      string
	PluginConfigDir string
	LogLevel       string
}

func Load() *Config {
	return &Config{
		HTTPAddr:        envOr("FINGUARD_ADDR", ":8080"),
		OpenCostURL:     envOr("OPENCOST_URL", "http://opencost.opencost.svc.cluster.local:9003"),
		PluginDir:       envOr("FINGUARD_PLUGIN_DIR", "/opt/finguard/plugins/bin"),
		PluginConfigDir: envOr("FINGUARD_PLUGIN_CONFIG_DIR", "/opt/finguard/plugins/config"),
		LogLevel:        envOr("FINGUARD_LOG_LEVEL", "info"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
