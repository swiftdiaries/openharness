package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultConfig(t *testing.T) {
	// Loading a non-existent path should return defaults without error.
	cfg, err := Load(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Gateway.Port != 18790 {
		t.Errorf("expected gateway port 18790, got %d", cfg.Gateway.Port)
	}
	if cfg.Gateway.Host != "127.0.0.1" {
		t.Errorf("expected gateway host 127.0.0.1, got %s", cfg.Gateway.Host)
	}
	if len(cfg.Providers) != 1 || cfg.Providers[0].Name != "openrouter" {
		t.Errorf("expected single openrouter provider, got %+v", cfg.Providers)
	}
	if !cfg.Providers[0].Enabled {
		t.Error("expected openrouter provider to be enabled")
	}
	if len(cfg.MCPServers) != 1 || cfg.MCPServers[0].Name != "linear" {
		t.Errorf("expected single linear MCP server, got %+v", cfg.MCPServers)
	}
	if cfg.MCPServers[0].Enabled {
		t.Error("expected linear MCP server to be disabled by default")
	}
	if cfg.Cost.DailyLimitUSD != 10.0 {
		t.Errorf("expected daily limit 10.0, got %f", cfg.Cost.DailyLimitUSD)
	}
	if cfg.Cost.MonthlyLimitUSD != 100.0 {
		t.Errorf("expected monthly limit 100.0, got %f", cfg.Cost.MonthlyLimitUSD)
	}
	if cfg.DefaultModel != "google/gemma-4-31b-it:free" {
		t.Errorf("expected default model google/gemma-4-31b-it:free, got %s", cfg.DefaultModel)
	}
	if len(cfg.Agents) != 5 {
		t.Errorf("expected 5 default agents, got %d", len(cfg.Agents))
	}
}

func TestSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := Default()
	cfg.MCPServers = append(cfg.MCPServers, MCPServerConfig{
		Name:      "github",
		Transport: "http",
		URL:       "https://mcp.github.com/mcp",
		Enabled:   true,
	})

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file was created.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.MCPServers) != 2 {
		t.Fatalf("expected 2 MCP servers, got %d", len(loaded.MCPServers))
	}
	github := loaded.MCPServers[1]
	if github.Name != "github" {
		t.Errorf("expected second server name 'github', got %s", github.Name)
	}
	if !github.Enabled {
		t.Error("expected github MCP server to be enabled")
	}
	if github.URL != "https://mcp.github.com/mcp" {
		t.Errorf("unexpected URL: %s", github.URL)
	}

	// Original defaults should still be intact.
	if loaded.DefaultModel != "google/gemma-4-31b-it:free" {
		t.Errorf("default model not persisted, got %s", loaded.DefaultModel)
	}
	if loaded.Gateway.Port != 18790 {
		t.Errorf("gateway port not persisted, got %d", loaded.Gateway.Port)
	}
	if loaded.Cost.MonthlyLimitUSD != 100.0 {
		t.Errorf("monthly limit not persisted, got %f", loaded.Cost.MonthlyLimitUSD)
	}
}

func TestResolveModel_PerAgentOverride(t *testing.T) {
	cfg := Default()
	model := ResolveModel("scribe", cfg)
	if model != "z-ai/glm-4.5-air:free" {
		t.Errorf("expected scribe override, got %s", model)
	}
}

func TestResolveModel_HarnessOverride(t *testing.T) {
	cfg := Default()
	model := ResolveModel("harness", cfg)
	if model != "anthropic/claude-haiku-4.5" {
		t.Errorf("expected harness override, got %s", model)
	}
}

func TestResolveModel_FallbackToDefault(t *testing.T) {
	cfg := Default()
	model := ResolveModel("analyst", cfg)
	if model != "google/gemma-4-31b-it:free" {
		t.Errorf("expected default model, got %s", model)
	}
}

func TestResolveModel_UnknownAgent(t *testing.T) {
	cfg := Default()
	model := ResolveModel("nonexistent", cfg)
	if model != "google/gemma-4-31b-it:free" {
		t.Errorf("expected default model for unknown agent, got %s", model)
	}
}
