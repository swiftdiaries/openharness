package lite

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/swiftdiaries/openharness/harness"
)

// Compile-time check: LiteToolRegistry implements ToolRegistry.
var _ harness.ToolRegistry = (*LiteToolRegistry)(nil)

// LiteToolRegistry is a standalone in-memory ToolRegistry for the Lite edition.
// It maintains a global catalog of built-in tools and per-tenant overlays.
// Tenant definitions override globals on name collision during Resolve.
type LiteToolRegistry struct {
	mu      sync.RWMutex
	globals map[string]harness.ToolDefinition            // name -> def
	tenants map[string]map[string]harness.ToolDefinition // tenantID -> name -> def
}

// NewLiteToolRegistry creates a new in-memory tool registry with empty maps.
func NewLiteToolRegistry() *LiteToolRegistry {
	return &LiteToolRegistry{
		globals: make(map[string]harness.ToolDefinition),
		tenants: make(map[string]map[string]harness.ToolDefinition),
	}
}

// AddGlobalTool seeds a global tool definition. Called at init time by the
// desktop app to register built-in tools.
func (r *LiteToolRegistry) AddGlobalTool(def harness.ToolDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.globals[def.Name] = def
}

// ListGlobal returns all global tool definitions.
func (r *LiteToolRegistry) ListGlobal(_ context.Context) ([]harness.ToolDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]harness.ToolDefinition, 0, len(r.globals))
	for _, def := range r.globals {
		out = append(out, def)
	}
	return out, nil
}

// ListTenant returns tenant-specific tool definitions for the given tenantID.
func (r *LiteToolRegistry) ListTenant(_ context.Context, tenantID string) ([]harness.ToolDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tm := r.tenants[tenantID]
	out := make([]harness.ToolDefinition, 0, len(tm))
	for _, def := range tm {
		out = append(out, def)
	}
	return out, nil
}

// Resolve merges global and tenant tools for the requested names.
// Tenant definitions override global ones on name collision.
// Returns ErrNotFound if any requested name doesn't exist in either scope.
func (r *LiteToolRegistry) Resolve(_ context.Context, tenantID string, toolNames []string) ([]harness.ToolDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tm := r.tenants[tenantID] // may be nil

	out := make([]harness.ToolDefinition, 0, len(toolNames))
	for _, name := range toolNames {
		// Tenant takes priority over global.
		if tm != nil {
			if def, ok := tm[name]; ok {
				out = append(out, def)
				continue
			}
		}
		if def, ok := r.globals[name]; ok {
			out = append(out, def)
			continue
		}
		return nil, fmt.Errorf("%w: tool %q", harness.ErrNotFound, name)
	}
	return out, nil
}

// RegisterTool adds a tool definition to the tenant map. If the definition's
// ID is empty, one is generated from the current timestamp.
func (r *LiteToolRegistry) RegisterTool(_ context.Context, tenantID string, def harness.ToolDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if def.ID == "" {
		def.ID = fmt.Sprintf("tool_%d", time.Now().UnixNano())
	}
	def.TenantID = tenantID

	if r.tenants[tenantID] == nil {
		r.tenants[tenantID] = make(map[string]harness.ToolDefinition)
	}
	r.tenants[tenantID][def.Name] = def
	return nil
}

// UpdateTool replaces a tool in the tenant map by toolID.
// Returns ErrNotFound if no tool with that ID exists for the tenant.
func (r *LiteToolRegistry) UpdateTool(_ context.Context, tenantID, toolID string, def harness.ToolDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tm := r.tenants[tenantID]
	if tm == nil {
		return fmt.Errorf("%w: tool %q", harness.ErrNotFound, toolID)
	}

	for name, existing := range tm {
		if existing.ID == toolID {
			def.ID = toolID
			def.TenantID = tenantID
			// If the name changed, remove old key and insert new.
			if def.Name != name {
				delete(tm, name)
			}
			tm[def.Name] = def
			return nil
		}
	}
	return fmt.Errorf("%w: tool %q", harness.ErrNotFound, toolID)
}

// DeleteTool removes a tool from the tenant map by toolID.
// Returns ErrNotFound if no tool with that ID exists for the tenant.
func (r *LiteToolRegistry) DeleteTool(_ context.Context, tenantID, toolID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tm := r.tenants[tenantID]
	if tm == nil {
		return fmt.Errorf("%w: tool %q", harness.ErrNotFound, toolID)
	}

	for name, existing := range tm {
		if existing.ID == toolID {
			delete(tm, name)
			return nil
		}
	}
	return fmt.Errorf("%w: tool %q", harness.ErrNotFound, toolID)
}

// VerifyTool performs basic schema validation on a tool definition.
// It checks that name, description, and category are non-empty.
func (r *LiteToolRegistry) VerifyTool(_ context.Context, def harness.ToolDefinition) (harness.VerifyResult, error) {
	var errs []string
	if def.Name == "" {
		errs = append(errs, "name is required")
	}
	if def.Description == "" {
		errs = append(errs, "description is required")
	}
	if def.Category == "" {
		errs = append(errs, "category is required")
	}

	if len(errs) > 0 {
		return harness.VerifyResult{Valid: false, Errors: errs}, nil
	}
	return harness.VerifyResult{Valid: true}, nil
}
