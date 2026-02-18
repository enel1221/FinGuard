package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	cfg := Load()

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("expected default addr ':8080', got %q", cfg.HTTPAddr)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default log level 'info', got %q", cfg.LogLevel)
	}
	if cfg.OpenCostURL == "" {
		t.Error("expected non-empty OpenCost URL")
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	os.Setenv("FINGUARD_ADDR", ":9090")
	os.Setenv("FINGUARD_LOG_LEVEL", "debug")
	defer os.Unsetenv("FINGUARD_ADDR")
	defer os.Unsetenv("FINGUARD_LOG_LEVEL")

	cfg := Load()

	if cfg.HTTPAddr != ":9090" {
		t.Errorf("expected addr ':9090', got %q", cfg.HTTPAddr)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected log level 'debug', got %q", cfg.LogLevel)
	}
}

func TestEnvIntOr(t *testing.T) {
	if v := envIntOr("NONEXISTENT_VAR", 42); v != 42 {
		t.Errorf("expected 42, got %d", v)
	}

	os.Setenv("TEST_INT", "100")
	defer os.Unsetenv("TEST_INT")
	if v := envIntOr("TEST_INT", 42); v != 100 {
		t.Errorf("expected 100, got %d", v)
	}

	os.Setenv("TEST_BAD_INT", "abc")
	defer os.Unsetenv("TEST_BAD_INT")
	if v := envIntOr("TEST_BAD_INT", 42); v != 42 {
		t.Errorf("expected 42 for bad int, got %d", v)
	}
}
