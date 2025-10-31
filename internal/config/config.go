package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type Budget struct {
	InitialJS    string `json:"initialJs,omitempty"`
	ThirdPartyJS string `json:"thirdPartyJs,omitempty"`
}

type Probe struct {
	Profile string `json:"profile,omitempty"`
	Runs    int    `json:"runs,omitempty"`
}

type Collect struct {
	Framework string `json:"framework"`
	StatsFile string `json:"statsFile,omitempty"`
}

type Config struct {
	Extends []string          `json:"extends,omitempty"`
	Budgets map[string]Budget `json:"budgets,omitempty"`
	Rules   map[string]string `json:"rules,omitempty"`
	Probe   Probe             `json:"probe,omitempty"`
	Collect Collect           `json:"collect"`
}

func DefaultConfig() Config {
	return Config{
		Extends: []string{"recommended"},
		Budgets: map[string]Budget{
			"default": {InitialJS: "35kb", ThirdPartyJS: "50kb"},
			"/":       {InitialJS: "10kb"},
		},
		Rules: map[string]string{
			"route-initial-js":  "error",
			"unoptimized-image": "warn",
			"font-display":      "info",
		},
		Probe:   Probe{Profile: "desktop", Runs: 1},
		Collect: Collect{Framework: "next", StatsFile: ".tracelet/stats.json"},
	}
}

func Load(path string) (Config, error) {
	if path == "" {
		path = "tracelet.config.json"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file %s: %w. Run 'tracelet init' to create one", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("invalid JSON in config file %s: %w", path, err)
	}
	if err := validate(cfg); err != nil {
		return Config{}, fmt.Errorf("config validation failed: %w", err)
	}
	return cfg, nil
}

func validate(cfg Config) error {
	if cfg.Collect.Framework == "" {
		return errors.New("collect.framework is required (must be 'next' or 'vite')")
	}
	if cfg.Collect.Framework != "next" && cfg.Collect.Framework != "vite" {
		return fmt.Errorf("collect.framework must be 'next' or 'vite', got '%s'", cfg.Collect.Framework)
	}
	if cfg.Budgets == nil {
		return errors.New("budgets section is required")
	}
	if cfg.Budgets["default"].InitialJS == "" {
		return errors.New("budgets.default.initialJs is required (e.g., '35kb')")
	}
	return nil
}
