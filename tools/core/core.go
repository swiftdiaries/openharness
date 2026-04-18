// Package core holds built-in tool implementations for openharness: web,
// filesystem, exec, memory, tasks, ask_user, knowledge_graph, and subagent.
// Callers bring their own Registry and call Register(r, cfg) — the
// package does not construct a Registry on their behalf so verticals can
// mix built-ins with their own tools.
package core

import (
	"github.com/swiftdiaries/openharness/tools"
)

// Config captures what the built-in set needs to construct. All fields are
// optional where noted; omitted optional fields cause the corresponding
// tool to be skipped rather than erroring.
type Config struct {
	// WorkspacePath is the root for filesystem + exec + tasks (required
	// for those three tools to function; an empty string still succeeds
	// but scopes tools to the empty workspace).
	WorkspacePath string
	// MemoryPath is the JSON file backing the memory tool. An empty
	// string skips registration of memory_*.
	MemoryPath string
	// KnowledgeStore, if non-nil, enables the knowledge_graph_search tool.
	// nil skips registration — matches ghostfin's "notes optional" path.
	KnowledgeStore KnowledgeGraphStore
	// Subagent enumerates the subagent types advertised to the LLM. An
	// empty map (or zero value) skips registration of the agent tool.
	Subagent SubagentConfig
}

// Register populates r with built-in tools according to cfg. Every
// registered definition must carry a non-Unspecified Effects value;
// Register returns the underlying tools.Registry.Register error if any
// tool fails that contract — the partial registration state is
// whatever r.Register left behind.
//
// Skips:
//   - memory_* if cfg.MemoryPath == ""
//   - knowledge_graph_search if cfg.KnowledgeStore == nil
//   - agent if cfg.Subagent.Subagents is empty
func Register(r *tools.Registry, cfg Config) error {
	if err := r.Register("web", NewWebSearch()); err != nil {
		return err
	}
	if err := r.Register("web", NewWebFetch()); err != nil {
		return err
	}
	if err := r.Register("filesystem", NewFilesystem(cfg.WorkspacePath)); err != nil {
		return err
	}
	if err := r.Register("exec", NewExec(cfg.WorkspacePath)); err != nil {
		return err
	}
	if err := r.Register("interactive", NewAskUser()); err != nil {
		return err
	}
	if err := r.Register("interactive", NewTaskCRUD(cfg.WorkspacePath)); err != nil {
		return err
	}
	if cfg.MemoryPath != "" {
		if err := r.Register("memory", NewMemory(cfg.MemoryPath)); err != nil {
			return err
		}
	}
	if cfg.KnowledgeStore != nil {
		if err := r.Register("knowledge", NewKnowledgeGraph(cfg.KnowledgeStore)); err != nil {
			return err
		}
	}
	if len(cfg.Subagent.Subagents) > 0 {
		if err := r.Register("agent", NewSubagentTool(cfg.Subagent)); err != nil {
			return err
		}
	}
	return nil
}
