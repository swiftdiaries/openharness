package harness

import (
	"context"
	"encoding/json"
	"io"
	"time"
)

// HarnessRunner abstracts how agent runs are dispatched.
// Implementations include embedded goroutine (Lite), Kubernetes job, Lambda, etc.
type HarnessRunner interface {
	Dispatch(ctx context.Context, cfg RunConfig) (EventStream, error)
	Resume(ctx context.Context, sessionID string) (EventStream, error)
	Interrupt(ctx context.Context, runID string, event Event) error
	Cancel(ctx context.Context, runID string) error
	Status(ctx context.Context, runID string) (RunStatus, error)
}

// SessionStore is the durable, append-only event log for agent sessions.
type SessionStore interface {
	CreateSession(ctx context.Context, tenantID string, meta SessionMeta) (string, error)
	AppendEvent(ctx context.Context, sessionID string, event Event) error
	GetEvents(ctx context.Context, sessionID string, fromPos int64) ([]Event, error)
	GetSession(ctx context.Context, sessionID string) (Session, error)
	SetCheckpoint(ctx context.Context, sessionID string, pos int64) error
	ListSessions(ctx context.Context, tenantID string, opts ListOpts) ([]Session, error)
}

// SecretStore abstracts credential access.
type SecretStore interface {
	GetCredentials(ctx context.Context, tenantID, name string) (map[string]string, error)
	MintScopedToken(ctx context.Context, tenantID string, ttl time.Duration) (string, error)
	StoreCredentials(ctx context.Context, tenantID, name string, creds map[string]string) error
}

// ArtifactStore handles durable object storage for agent run outputs.
type ArtifactStore interface {
	Put(ctx context.Context, tenantID, runID, name string, data io.Reader) (string, error)
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	List(ctx context.Context, tenantID, runID string) ([]ArtifactMeta, error)
	Presign(ctx context.Context, key string, ttl time.Duration) (string, error)
}

// ToolRegistry manages tools with a global catalog + per-tenant overlay.
type ToolRegistry interface {
	ListGlobal(ctx context.Context) ([]ToolDefinition, error)
	ListTenant(ctx context.Context, tenantID string) ([]ToolDefinition, error)
	Resolve(ctx context.Context, tenantID string, toolNames []string) ([]ToolDefinition, error)
	RegisterTool(ctx context.Context, tenantID string, def ToolDefinition) error
	UpdateTool(ctx context.Context, tenantID, toolID string, def ToolDefinition) error
	DeleteTool(ctx context.Context, tenantID, toolID string) error
	VerifyTool(ctx context.Context, def ToolDefinition) (VerifyResult, error)
}

// ChannelRouter handles bidirectional message routing.
type ChannelRouter interface {
	HandleInbound(ctx context.Context, channelType string, payload json.RawMessage) (InboundResult, error)
	SendOutbound(ctx context.Context, tenantID, channelName string, msg Message) error
	RegisterWebhook(ctx context.Context, tenantID, channelType string, cfg ChannelConfig) (string, error)
	UnregisterWebhook(ctx context.Context, tenantID, channelType string) error
	ListChannels(ctx context.Context, tenantID string) ([]ChannelInfo, error)
}

// AgentStore manages versioned agent definitions.
type AgentStore interface {
	Create(ctx context.Context, tenantID string, def AgentDefinition) (AgentDefinition, error)
	Get(ctx context.Context, agentID string) (AgentDefinition, error)
	Update(ctx context.Context, agentID string, version int, patch AgentPatch) (AgentDefinition, error)
	Archive(ctx context.Context, agentID string) (AgentDefinition, error)
	List(ctx context.Context, tenantID string, opts ListOpts) ([]AgentDefinition, error)
	ListVersions(ctx context.Context, agentID string) ([]AgentDefinition, error)
	Clone(ctx context.Context, agentID string, newName string) (AgentDefinition, error)
}

// SkillStore manages skill metadata and artifacts.
type SkillStore interface {
	ListPlatform(ctx context.Context) ([]SkillMeta, error)
	ListTenant(ctx context.Context, tenantID string) ([]SkillMeta, error)
	Resolve(ctx context.Context, refs []SkillRef) ([]Skill, error)
	Upload(ctx context.Context, tenantID string, meta SkillMeta, bundle io.Reader) (SkillMeta, error)
	Delete(ctx context.Context, tenantID, skillID string) error
}
