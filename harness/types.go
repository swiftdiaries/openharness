package harness

import (
	"encoding/json"
	"time"
)

// Event type constants.
const (
	EventRunStarted   = "run.started"
	EventChunk        = "chunk"
	EventToolCall     = "tool.call"
	EventToolResult   = "tool.result"
	EventRunCompleted = "run.completed"
	EventError        = "error"
	EventAskUser      = "ask.user"
	EventActivity     = "activity"
	EventModeSwitch   = "mode.switch"
	EventCheckpoint   = "checkpoint"
)

// Event is a single entry in the append-only event log.
type Event struct {
	Type      string          `json:"type"`
	Content   string          `json:"content,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Position  int64           `json:"position"`
	Timestamp time.Time       `json:"timestamp"`
}

// RunConfig holds everything needed to dispatch an agent run.
type RunConfig struct {
	RunID          string          `json:"run_id"`
	SessionID      string          `json:"session_id"`
	TenantID       string          `json:"tenant_id"`
	AgentConfig    AgentDefinition `json:"agent_config"`
	HarnessVersion string          `json:"harness_version"`
	Tools          []ToolDefinition `json:"tools"`
	Skills         []Skill         `json:"skills"`
	VaultToken     string          `json:"vault_token,omitempty"`
	ArtifactBucket string          `json:"artifact_bucket,omitempty"`
	Mode           string          `json:"mode,omitempty"`
}

// RunStatus reports the current state of a dispatched run.
type RunStatus struct {
	RunID     string     `json:"run_id"`
	SessionID string     `json:"session_id"`
	Status    string     `json:"status"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Error     string     `json:"error,omitempty"`
}

// EventStream is an iterator over events emitted by an agent run.
type EventStream interface {
	Next() (Event, error)
	Close() error
}

// AgentDefinition is a versioned, storable agent configuration.
type AgentDefinition struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	Name           string          `json:"name"`
	Description    string          `json:"description,omitempty"`
	Model          string          `json:"model"`
	HarnessVersion string          `json:"harness_version,omitempty"`
	SystemPrompt   string          `json:"system_prompt,omitempty"`
	Persona        PersonaFiles    `json:"persona,omitempty"`
	Tools          []string        `json:"tools,omitempty"`
	Skills         []SkillRef      `json:"skills,omitempty"`
	MCPServers     []MCPServerConfig `json:"mcp_servers,omitempty"`
	Autonomy       string          `json:"autonomy,omitempty"`
	Runner         string          `json:"runner,omitempty"`
	FallbackRunner string          `json:"fallback_runner,omitempty"`
	AgentType      string          `json:"agent_type,omitempty"`
	CallableAgents []string        `json:"callable_agents,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Version        int             `json:"version"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	ArchivedAt     *time.Time      `json:"archived_at,omitempty"`
}

// PersonaFiles holds the file-based persona configuration for an agent.
type PersonaFiles struct {
	Soul      string `json:"soul,omitempty"`
	Identity  string `json:"identity,omitempty"`
	Agents    string `json:"agents,omitempty"`
	User      string `json:"user,omitempty"`
	Bootstrap string `json:"bootstrap,omitempty"`
}

// MCPServerConfig describes an MCP server to attach to an agent.
type MCPServerConfig struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// AgentPatch holds partial updates for an agent definition.
// All fields are pointers; nil means "do not update".
type AgentPatch struct {
	Name         *string        `json:"name,omitempty"`
	Description  *string        `json:"description,omitempty"`
	Model        *string        `json:"model,omitempty"`
	HarnessVersion *string      `json:"harness_version,omitempty"`
	SystemPrompt *string        `json:"system_prompt,omitempty"`
	Persona      *PersonaFiles  `json:"persona,omitempty"`
	Tools        *[]string      `json:"tools,omitempty"`
	Skills       *[]SkillRef    `json:"skills,omitempty"`
	Autonomy     *string        `json:"autonomy,omitempty"`
	Metadata     *map[string]string `json:"metadata,omitempty"`
}

// ToolDefinition describes a tool available to agents.
type ToolDefinition struct {
	ID            string          `json:"id"`
	TenantID      string          `json:"tenant_id,omitempty"`
	Name          string          `json:"name"`
	Category      string          `json:"category,omitempty"`
	Description   string          `json:"description"`
	Parameters    json.RawMessage `json:"parameters,omitempty"`
	Endpoint      string          `json:"endpoint,omitempty"`
	CredentialRef string          `json:"credential_ref,omitempty"`
	BuiltIn       bool            `json:"built_in"`
	Verified      bool            `json:"verified"`
	VerifiedAt    *time.Time      `json:"verified_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// VerifyResult reports tool verification outcome.
type VerifyResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// SkillMeta is metadata for a skill (without the prompt/examples payload).
type SkillMeta struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id,omitempty"`
	Name        string    `json:"name"`
	Category    string    `json:"category,omitempty"`
	Description string    `json:"description"`
	Triggers    []string  `json:"triggers,omitempty"`
	ArtifactKey string    `json:"artifact_key,omitempty"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
}

// Skill is a resolved skill with its full prompt and examples.
type Skill struct {
	Meta     SkillMeta `json:"meta"`
	Prompt   string    `json:"prompt"`
	Examples []string  `json:"examples,omitempty"`
}

// SkillRef is a reference to a skill by ID and optional version.
// Version 0 means "use latest".
type SkillRef struct {
	SkillID string `json:"skill_id"`
	Version int    `json:"version,omitempty"`
}

// SessionMeta holds metadata for creating a new session.
type SessionMeta struct {
	AgentID   string            `json:"agent_id"`
	ChannelID string            `json:"channel_id,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Session represents an agent session with its event log cursor.
type Session struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	AgentID     string    `json:"agent_id"`
	Checkpoint  int64     `json:"checkpoint"`
	CreatedAt   time.Time `json:"created_at"`
	LastEventAt time.Time `json:"last_event_at"`
}

// Message is a channel message payload.
type Message struct {
	RunID     string `json:"run_id,omitempty"`
	AgentName string `json:"agent_name,omitempty"`
	Content   string `json:"content"`
	Status    string `json:"status,omitempty"`
	Error     string `json:"error,omitempty"`
}

// InboundResult is returned when a channel message triggers an agent run.
type InboundResult struct {
	SessionID string      `json:"session_id"`
	RunID     string      `json:"run_id"`
	Stream    EventStream `json:"-"`
}

// ChannelConfig describes a channel integration.
type ChannelConfig struct {
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Direction string          `json:"direction"`
	Config    json.RawMessage `json:"config,omitempty"`
}

// ChannelInfo is the stored state of a registered channel.
type ChannelInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Direction  string `json:"direction"`
	WebhookURL string `json:"webhook_url,omitempty"`
	Enabled    bool   `json:"enabled"`
}

// ArtifactMeta describes a stored artifact.
type ArtifactMeta struct {
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

// ListOpts provides pagination and filtering for list operations.
type ListOpts struct {
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Status string `json:"status,omitempty"`
}

// Snapshot captures a point-in-time view of a session for resume.
type Snapshot struct {
	SessionID   string          `json:"session_id"`
	RunID       string          `json:"run_id"`
	Sequence    int             `json:"sequence"`
	Checkpoint  int64           `json:"checkpoint"`
	AgentConfig AgentDefinition `json:"agent_config"`
	Status      string          `json:"status"`
}
