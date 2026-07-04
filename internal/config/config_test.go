package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidate(t *testing.T) {
	base := func() Config {
		return Config{
			Collect: Collect{Framework: "next"},
			Budgets: map[string]Budget{"default": {InitialJS: "35kb"}},
		}
	}

	if err := validate(base()); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*Config)
	}{
		{"missing framework", func(c *Config) { c.Collect.Framework = "" }},
		{"bad framework", func(c *Config) { c.Collect.Framework = "svelte" }},
		{"missing budgets", func(c *Config) { c.Budgets = nil }},
		{"missing default initialJs", func(c *Config) { c.Budgets = map[string]Budget{"default": {}} }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := base()
			c.mutate(&cfg)
			if err := validate(cfg); err == nil {
				t.Errorf("expected validation error for %q, got nil", c.name)
			}
		})
	}
}

func TestHUDIsEnabled(t *testing.T) {
	yes, no := true, false
	if !(HUD{}).IsEnabled() {
		t.Error("HUD with nil Enabled should default to enabled")
	}
	if !(HUD{Enabled: &yes}).IsEnabled() {
		t.Error("HUD{Enabled: true} should be enabled")
	}
	if (HUD{Enabled: &no}).IsEnabled() {
		t.Error("HUD{Enabled: false} should be disabled")
	}
}

func TestLoadRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "tracelet.config.json")
	b, _ := json.Marshal(DefaultConfig())
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load failed on default config: %v", err)
	}
	if cfg.Collect.Framework != "next" {
		t.Errorf("round-trip framework = %q, want next", cfg.Collect.Framework)
	}

	if _, err := Load(filepath.Join(t.TempDir(), "does-not-exist.json")); err == nil {
		t.Error("Load of missing file should error")
	}
}
