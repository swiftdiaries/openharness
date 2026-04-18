package tools

import (
	"fmt"
	"sort"
)

// ToolCategory groups tools under a named category. Retained from ghostfin
// for Plan 4 (MCP) compatibility; Plan 3 callers do not use it yet.
type ToolCategory struct {
	Name  string           `json:"name"`
	Count int              `json:"count"`
	Tools []ToolDefinition `json:"tools"`
}

// Registry holds registered tools and their category/effects mappings.
// Registration is Effects-aware: every definition must carry a
// non-Unspecified Effects value or Register returns an error. Effects are
// precomputed into an O(1) map because Plan 5's loop-detection path
// calls EffectsFor on every tool invocation.
type Registry struct {
	tools             map[string]Tool
	categories        map[string]string     // def name -> category
	effects           map[string]ToolEffect // def name -> effects
	deferredActivator func(name string) bool
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		tools:      make(map[string]Tool),
		categories: make(map[string]string),
		effects:    make(map[string]ToolEffect),
	}
}

// Register adds a tool under the given category. Every definition returned
// by t.Definitions() must carry a non-Unspecified Effects value, otherwise
// Register returns an error naming the offending definition and nothing is
// added to the registry. If a later definition fails validation, earlier
// definitions in the same call are rolled back.
func (r *Registry) Register(category string, t Tool) error {
	defs := t.Definitions()
	for _, def := range defs {
		if def.Effects == ToolEffectUnspecified {
			return fmt.Errorf("tools: definition %q has Effects: ToolEffectUnspecified; every registered definition must classify its effects", def.Name)
		}
	}
	for _, def := range defs {
		r.tools[def.Name] = t
		r.categories[def.Name] = category
		r.effects[def.Name] = def.Effects
	}
	return nil
}

// Get returns the Tool that handles the given tool name, or nil.
func (r *Registry) Get(name string) Tool {
	return r.tools[name]
}

// EffectsFor returns the precomputed ToolEffect for a definition name.
// Unknown names return ToolEffectUnspecified with no error — the caller
// decides how to handle unclassified tools.
func (r *Registry) EffectsFor(name string) ToolEffect {
	return r.effects[name]
}

// Unregister removes a tool by name from the registry.
func (r *Registry) Unregister(name string) {
	delete(r.categories, name)
	delete(r.tools, name)
	delete(r.effects, name)
}

// Categories returns all categories with their tool counts and definitions.
func (r *Registry) Categories() []ToolCategory {
	catTools := make(map[string][]ToolDefinition)
	for name, cat := range r.categories {
		t := r.tools[name]
		for _, def := range t.Definitions() {
			if def.Name == name {
				catTools[cat] = append(catTools[cat], def)
			}
		}
	}

	var result []ToolCategory
	for cat, defs := range catTools {
		result = append(result, ToolCategory{Name: cat, Count: len(defs), Tools: defs})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// ToolsByCategory returns all Tool implementations in the given category.
func (r *Registry) ToolsByCategory(cat string) []Tool {
	seen := make(map[Tool]bool)
	var result []Tool
	for name, c := range r.categories {
		if c == cat {
			t := r.tools[name]
			if !seen[t] {
				seen[t] = true
				result = append(result, t)
			}
		}
	}
	return result
}

// SetDeferredActivator sets a callback that can lazily activate deferred tools by name.
func (r *Registry) SetDeferredActivator(fn func(name string) bool) {
	r.deferredActivator = fn
}

// TryActivateDeferred attempts to activate a deferred tool by name.
// Returns the tool if activation succeeded, nil otherwise.
func (r *Registry) TryActivateDeferred(name string) Tool {
	if r.deferredActivator != nil && r.deferredActivator(name) {
		return r.tools[name]
	}
	return nil
}

// ToolsByNames returns the Tools wrapping the named definitions, deduplicated.
func (r *Registry) ToolsByNames(names ...string) []Tool {
	seen := make(map[Tool]bool)
	var result []Tool
	for _, name := range names {
		if t, ok := r.tools[name]; ok && !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}
	return result
}
