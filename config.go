package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Address      string        `toml:"address"`
	Path         string        `toml:"path"`
	Timeout      time.Duration `toml:"timeout"`
	JWTSecret    string        `toml:"jwt_secret"`
	DefaultModel string        `toml:"default_model"`
}

func LoadConfig(configPath string) (*Config, error) {
	cfg := &Config{
		Address:      "localhost:8080",
		Path:         "codex",
		Timeout:      30 * time.Second,
		JWTSecret:    "",
		DefaultModel: "claude-3-5-sonnet-20241022",
	}

	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			if _, err := toml.DecodeFile(configPath, cfg); err != nil {
				return nil, fmt.Errorf("failed to decode config: %w", err)
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to stat config: %w", err)
		}
	} else {
		localPathCodex := "./.codex-serve"
		localPathClaude := "./.claude-serve"
		globalPathCodex := filepath.Join(os.Getenv("HOME"), ".codex-serve", "config")
		globalPathClaude := filepath.Join(os.Getenv("HOME"), ".claude-serve", "config")

		if _, err := os.Stat(localPathCodex); err == nil {
			if _, err := toml.DecodeFile(localPathCodex, cfg); err != nil {
				return nil, fmt.Errorf("failed to decode local config: %w", err)
			}
		} else if _, err := os.Stat(localPathClaude); err == nil {
			if _, err := toml.DecodeFile(localPathClaude, cfg); err != nil {
				return nil, fmt.Errorf("failed to decode local config: %w", err)
			}
		} else if _, err := os.Stat(globalPathCodex); err == nil {
			if _, err := toml.DecodeFile(globalPathCodex, cfg); err != nil {
				return nil, fmt.Errorf("failed to decode global config: %w", err)
			}
		} else if _, err := os.Stat(globalPathClaude); err == nil {
			if _, err := toml.DecodeFile(globalPathClaude, cfg); err != nil {
				return nil, fmt.Errorf("failed to decode global config: %w", err)
			}
		}
	}

	if addr := os.Getenv("CODEX_ADDRESS"); addr != "" {
		cfg.Address = addr
	}
	if path := os.Getenv("CODEX_PATH"); path != "" {
		cfg.Path = path
	}
	if timeout := os.Getenv("CODEX_TIMEOUT"); timeout != "" {
		d, err := time.ParseDuration(timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid CODEX_TIMEOUT: %w", err)
		}
		cfg.Timeout = d
	}
	if secret := os.Getenv("CODEX_JWT_SECRET"); secret != "" {
		cfg.JWTSecret = secret
	}
	if model := os.Getenv("CODEX_DEFAULT_MODEL"); model != "" {
		cfg.DefaultModel = model
	}

	return cfg, nil
}
