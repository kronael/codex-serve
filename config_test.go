package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test loading config with default values
func TestLoadConfig_Defaults(t *testing.T) {
	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Address != "localhost:8080" {
		t.Errorf("expected default address localhost:8080, got %s", cfg.Address)
	}
	if cfg.Path != "codex" {
		t.Errorf("expected default path codex, got %s", cfg.Path)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", cfg.Timeout)
	}
	if cfg.DefaultModel != "claude-3-5-sonnet-20241022" {
		t.Errorf("expected default model claude-3-5-sonnet-20241022, got %s", cfg.DefaultModel)
	}
}

// Test loading config from environment variables
func TestLoadConfig_EnvVars(t *testing.T) {
	os.Setenv("CODEX_ADDRESS", "0.0.0.0:9000")
	os.Setenv("CODEX_PATH", "/usr/bin/codex")
	os.Setenv("CODEX_TIMEOUT", "60s")
	os.Setenv("CODEX_JWT_SECRET", "test-secret")
	os.Setenv("CODEX_DEFAULT_MODEL", "claude-opus-4-5-20251101")
	defer func() {
		os.Unsetenv("CODEX_ADDRESS")
		os.Unsetenv("CODEX_PATH")
		os.Unsetenv("CODEX_TIMEOUT")
		os.Unsetenv("CODEX_JWT_SECRET")
		os.Unsetenv("CODEX_DEFAULT_MODEL")
	}()

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Address != "0.0.0.0:9000" {
		t.Errorf("expected address 0.0.0.0:9000, got %s", cfg.Address)
	}
	if cfg.Path != "/usr/bin/codex" {
		t.Errorf("expected path /usr/bin/codex, got %s", cfg.Path)
	}
	if cfg.Timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", cfg.Timeout)
	}
	if cfg.JWTSecret != "test-secret" {
		t.Errorf("expected jwt secret test-secret, got %s", cfg.JWTSecret)
	}
	if cfg.DefaultModel != "claude-opus-4-5-20251101" {
		t.Errorf("expected model claude-opus-4-5-20251101, got %s", cfg.DefaultModel)
	}
}

// Test loading config from TOML file
func TestLoadConfig_TOMLFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
address = "127.0.0.1:8888"
path = "claude"
timeout = "45s"
jwt_secret = "file-secret"
default_model = "claude-3-5-sonnet-20241022"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Address != "127.0.0.1:8888" {
		t.Errorf("expected address 127.0.0.1:8888, got %s", cfg.Address)
	}
	if cfg.Path != "claude" {
		t.Errorf("expected path claude, got %s", cfg.Path)
	}
	if cfg.Timeout != 45*time.Second {
		t.Errorf("expected timeout 45s, got %v", cfg.Timeout)
	}
	if cfg.JWTSecret != "file-secret" {
		t.Errorf("expected jwt secret file-secret, got %s", cfg.JWTSecret)
	}
}

// Test that environment variables override TOML file
func TestLoadConfig_Precedence(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
address = "127.0.0.1:8888"
path = "claude"
default_model = "claude-3-5-sonnet-20241022"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	os.Setenv("CODEX_ADDRESS", "0.0.0.0:7777")
	os.Setenv("CODEX_DEFAULT_MODEL", "claude-opus-4-5-20251101")
	defer func() {
		os.Unsetenv("CODEX_ADDRESS")
		os.Unsetenv("CODEX_DEFAULT_MODEL")
	}()

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Address != "0.0.0.0:7777" {
		t.Errorf("expected env var to override, got %s", cfg.Address)
	}
	if cfg.Path != "claude" {
		t.Errorf("expected path from file, got %s", cfg.Path)
	}
	if cfg.DefaultModel != "claude-opus-4-5-20251101" {
		t.Errorf("expected env var to override model, got %s", cfg.DefaultModel)
	}
}
