package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Address   string        `toml:"address"`
	Path      string        `toml:"path"`
	Timeout   time.Duration `toml:"timeout"`
	JWTSecret string        `toml:"jwt_secret"`
}

func LoadConfig(configPath string) (*Config, error) {
	cfg := &Config{
		Address:   "localhost:8080",
		Path:      "codex",
		Timeout:   30 * time.Second,
		JWTSecret: "",
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
		localPath := "./.claude-serve"
		globalPath := filepath.Join(os.Getenv("HOME"), ".claude-serve", "config")

		if _, err := os.Stat(localPath); err == nil {
			if _, err := toml.DecodeFile(localPath, cfg); err != nil {
				return nil, fmt.Errorf("failed to decode local config: %w", err)
			}
		} else if _, err := os.Stat(globalPath); err == nil {
			if _, err := toml.DecodeFile(globalPath, cfg); err != nil {
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

	return cfg, nil
}
