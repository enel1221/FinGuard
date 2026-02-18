package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr        string
	OpenCostURL     string
	PluginDir       string
	PluginConfigDir string
	LogLevel        string
	DevMode         bool
	AuthDisabled    bool
	DatabaseDSN     string

	// OIDC configuration
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string
	OIDCScopes       []string
	SessionSecret    string
}

func Load() *Config {
	return &Config{
		HTTPAddr:         envOr("FINGUARD_ADDR", ":8080"),
		OpenCostURL:      envOr("OPENCOST_URL", "http://opencost.opencost.svc.cluster.local:9003"),
		PluginDir:        envOr("FINGUARD_PLUGIN_DIR", "/opt/finguard/plugins/bin"),
		PluginConfigDir:  envOr("FINGUARD_PLUGIN_CONFIG_DIR", "/opt/finguard/plugins/config"),
		LogLevel:         envOr("FINGUARD_LOG_LEVEL", "info"),
		DevMode:          envBool("FINGUARD_DEV_MODE"),
		AuthDisabled:     envBool("FINGUARD_AUTH_DISABLED"),
		DatabaseDSN:      envOr("FINGUARD_DB_DSN", "sqlite:///tmp/finguard.db"),
		OIDCIssuer:       envOr("FINGUARD_OIDC_ISSUER", ""),
		OIDCClientID:     envOr("FINGUARD_OIDC_CLIENT_ID", "finguard"),
		OIDCClientSecret: envOr("FINGUARD_OIDC_CLIENT_SECRET", ""),
		OIDCRedirectURL:  envOr("FINGUARD_OIDC_REDIRECT_URL", ""),
		OIDCScopes:       envSlice("FINGUARD_OIDC_SCOPES", []string{"openid", "profile", "email", "groups"}),
		SessionSecret:    envOr("FINGUARD_SESSION_SECRET", ""),
	}
}

func envBool(key string) bool {
	v := os.Getenv(key)
	return v == "true" || v == "1" || v == "yes"
}

func envSlice(key string, fallback []string) []string {
	if v := os.Getenv(key); v != "" {
		return strings.Split(v, ",")
	}
	return fallback
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
