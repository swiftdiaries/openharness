package config

import (
	"os"
	"path/filepath"
)

// AppConfig holds the full application configuration.
type AppConfig struct {
	DefaultModel string            `json:"default_model"`
	Gateway      GatewayConfig     `json:"gateway"`
	Providers  []ProviderConfig  `json:"providers"`
	MCPServers []MCPServerConfig `json:"mcp_servers"`
	Agents     []AgentOverride   `json:"agents"`
	Channels   []ChannelConfig   `json:"channels"`
	Cost       CostConfig         `json:"cost"`
	Compaction CompactionSettings `json:"compaction"`
	Workspace  WorkspaceConfig    `json:"workspace"`

	// Enterprise fields
	Edition       string `json:"edition"`         // "lite" or "enterprise"
	MgmtServerURL string `json:"mgmt_server_url"` // e.g. "http://mgmt.ghostfin.io:8080"
	FirmID        string `json:"firm_id"`
	UserID        string `json:"user_id"`
	PGDSN         string `json:"pg_dsn"`           // PostgreSQL connection string
}

// WorkspaceConfig configures the local workspace directory for user-editable files.
// Journal markdown files are stored at {Path}/journal/.
type WorkspaceConfig struct {
	Path string `json:"path"` // default: ~/.ghostfin/workspace/
}

// GatewayConfig configures the local gateway server.
type GatewayConfig struct {
	Port int    `json:"port"`
	Host string `json:"host"`
}

// ProviderConfig represents an LLM provider.
type ProviderConfig struct {
	Name         string `json:"name"`
	DisplayName  string `json:"display_name"`
	ProviderType string `json:"provider_type"`
	APIBase      string `json:"api_base"`
	APIKey       string `json:"api_key"`
	Enabled      bool   `json:"enabled"`
}

// MCPServerConfig represents an MCP server connection.
type MCPServerConfig struct {
	Name       string            `json:"name"`
	Transport  string            `json:"transport"`            // "stdio", "sse", "streamable-http"
	Command    string            `json:"command,omitempty"`
	Args       []string          `json:"args,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	URL        string            `json:"url,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	APIKey     string            `json:"api_key,omitempty"`
	Enabled    bool              `json:"enabled"`
	ToolPrefix string            `json:"tool_prefix,omitempty"`
	TimeoutSec        int               `json:"timeout_sec,omitempty"`
	OAuthClientID     string            `json:"oauth_client_id,omitempty"`
	OAuthClientSecret string            `json:"oauth_client_secret,omitempty"`
	OAuthScopes       []string          `json:"oauth_scopes,omitempty"`
}

// AgentOverride allows per-agent model/behavior overrides.
type AgentOverride struct {
	Name     string `json:"name"`
	Model    string `json:"model"`
	Autonomy string `json:"autonomy"`
	Enabled  bool   `json:"enabled"`
}

// ChannelConfig represents a notification/integration channel.
type ChannelConfig struct {
	Name        string `json:"name"`
	ChannelType string `json:"channel_type"`
	Enabled     bool   `json:"enabled"`
}

// CostConfig sets spending limits.
type CostConfig struct {
	DailyLimitUSD   float64 `json:"daily_limit_usd"`
	MonthlyLimitUSD float64 `json:"monthly_limit_usd"`
}

// CompactionSettings configures auto-compaction behavior for agent loops.
type CompactionSettings struct {
	MaxContextShare  float64 `json:"max_context_share"`  // 0.0-1.0, default 0.5
	KeepLastMessages int     `json:"keep_last_messages"` // default 4
}

// Default returns an AppConfig populated with sensible defaults.
// DefaultWorkspacePath returns the default workspace path (~/.ghostfin/workspace/).
func DefaultWorkspacePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ghostfin", "workspace")
}

func Default() AppConfig {
	return AppConfig{
		DefaultModel: "google/gemma-4-31b-it:free",
		Gateway: GatewayConfig{
			Port: 18790,
			Host: "127.0.0.1",
		},
		Providers: []ProviderConfig{
			{
				Name:         "openrouter",
				DisplayName:  "OpenRouter",
				ProviderType: "openai_compat",
				APIBase:      "https://openrouter.ai/api/v1",
				APIKey:       "$OPENROUTER_API_KEY",
				Enabled:      true,
			},
		},
		MCPServers: []MCPServerConfig{
			{
				Name:      "linear",
				Transport: "streamable-http",
				URL:       "https://mcp.linear.app/mcp",
				Enabled:   false,
			},
		},
		Agents: []AgentOverride{
			{Name: "harness", Model: "anthropic/claude-haiku-4.5", Autonomy: "auto", Enabled: true},
			{Name: "analyst", Model: "", Autonomy: "approval", Enabled: true},
			{Name: "scribe", Model: "z-ai/glm-4.5-air:free", Autonomy: "approval", Enabled: true},
			{Name: "executive-assistant", Model: "", Autonomy: "auto", Enabled: true},
			{Name: "fund-analyst", Model: "", Autonomy: "approval", Enabled: true},
		},
		Channels: []ChannelConfig{},
		Cost: CostConfig{
			DailyLimitUSD:   10.0,
			MonthlyLimitUSD: 100.0,
		},
		Compaction: CompactionSettings{
			MaxContextShare:  0.5,
			KeepLastMessages: 4,
		},
		Workspace: WorkspaceConfig{
			Path: DefaultWorkspacePath(),
		},
	}
}

// ResolveModel returns the model for an agent: per-agent override if set, else DefaultModel.
func ResolveModel(agentName string, cfg AppConfig) string {
	for _, a := range cfg.Agents {
		if a.Name == agentName && a.Model != "" {
			return a.Model
		}
	}
	return cfg.DefaultModel
}
