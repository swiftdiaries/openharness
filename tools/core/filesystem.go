package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/swiftdiaries/openharness/tools"
)

// Filesystem provides file operations scoped to a workspace directory.
type Filesystem struct {
	workspace  string
	agentType  string // "open" | "predefined" | "" (no guard)
	selfEvolve bool
}

// predefinedImmutableFiles are context files that predefined agents may never modify.
var predefinedImmutableFiles = map[string]bool{
	"IDENTITY.md": true,
	"AGENTS.md":   true,
}

// SetWriteGuard configures context file write protection.
func (f *Filesystem) SetWriteGuard(agentType string, selfEvolve bool) {
	f.agentType = agentType
	f.selfEvolve = selfEvolve
}

// NewFilesystem creates a Filesystem tool scoped to the given workspace directory.
// Canonicalizes the workspace path at construction to avoid repeated resolution.
func NewFilesystem(workspace string) *Filesystem {
	resolved, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		resolved = filepath.Clean(workspace)
	}
	return &Filesystem{workspace: resolved}
}

// DefaultWorkspace returns ~/.ghostfin/workspace/.
func DefaultWorkspace() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ghostfin", "workspace")
}

// ValidatePath checks that the resolved path is within the workspace. Exported for testing.
func (f *Filesystem) ValidatePath(p string) (string, error) {
	if filepath.IsAbs(p) {
		return "", fmt.Errorf("path %q escapes workspace", p)
	}
	joined := filepath.Join(f.workspace, p)

	resolved, err := filepath.EvalSymlinks(joined)
	if err != nil {
		parent := filepath.Dir(joined)
		canonicalParent, perr := filepath.EvalSymlinks(parent)
		if perr != nil {
			return "", fmt.Errorf("path %q escapes workspace", p)
		}
		resolved = filepath.Join(canonicalParent, filepath.Base(joined))
	}
	resolved = filepath.Clean(resolved)

	wsPrefix := f.workspace + string(filepath.Separator)
	if !strings.HasPrefix(resolved, wsPrefix) && resolved != f.workspace {
		return "", fmt.Errorf("path %q escapes workspace", p)
	}
	return resolved, nil
}

func (f *Filesystem) Definitions() []tools.ToolDefinition {
	return []tools.ToolDefinition{
		{
			Name:        "read_file",
			Description: "Read the contents of a file in the workspace.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string", "description": "Relative path within workspace"}
				},
				"required": ["path"]
			}`),
			Effects: tools.ToolEffectRead,
		},
		{
			Name:        "write_file",
			Description: "Write content to a file in the workspace.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string", "description": "Relative path within workspace"},
					"content": {"type": "string", "description": "File content to write"}
				},
				"required": ["path", "content"]
			}`),
			Effects: tools.ToolEffectMutate,
		},
		{
			Name:        "list_files",
			Description: "List files and directories at a path in the workspace.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string", "description": "Relative path within workspace (default: root)"}
				}
			}`),
			Effects: tools.ToolEffectRead,
		},
		{
			Name:        "edit_file",
			Description: "Replace a substring in a file within the workspace.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string", "description": "Relative path within workspace"},
					"old_text": {"type": "string", "description": "Text to find"},
					"new_text": {"type": "string", "description": "Replacement text"}
				},
				"required": ["path", "old_text", "new_text"]
			}`),
			Effects: tools.ToolEffectMutate,
		},
	}
}

func (f *Filesystem) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	switch name {
	case "read_file":
		return f.readFile(args)
	case "write_file":
		return f.writeFile(args)
	case "list_files":
		return f.listFiles(args)
	case "edit_file":
		return f.editFile(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (f *Filesystem) readFile(args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}
	resolved, err := f.ValidatePath(params.Path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]string{"content": SanitizeRead(string(data))})
}

func (f *Filesystem) writeFile(args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}
	resolved, err := f.ValidatePath(params.Path)
	if err != nil {
		return nil, err
	}
	// Context file write guard
	if f.agentType == "predefined" {
		base := filepath.Base(resolved)
		if predefinedImmutableFiles[base] {
			return nil, fmt.Errorf("cannot modify %s: immutable for predefined agents", base)
		}
		if base == "SOUL.md" && !f.selfEvolve {
			return nil, fmt.Errorf("cannot modify SOUL.md: self-evolution is disabled for this agent")
		}
	}
	os.MkdirAll(filepath.Dir(resolved), 0755)
	if err := os.WriteFile(resolved, []byte(params.Content), 0644); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]string{"status": "success"})
}

func (f *Filesystem) listFiles(args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}
	if params.Path == "" {
		params.Path = "."
	}
	resolved, err := f.ValidatePath(params.Path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(resolved)
	if err != nil {
		return nil, err
	}

	type FileInfo struct {
		Name  string `json:"name"`
		IsDir bool   `json:"is_dir"`
		Size  int64  `json:"size"`
	}
	var files []FileInfo
	for _, e := range entries {
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		files = append(files, FileInfo{Name: e.Name(), IsDir: e.IsDir(), Size: size})
	}
	return json.Marshal(files)
}

func (f *Filesystem) editFile(args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		Path    string `json:"path"`
		OldText string `json:"old_text"`
		NewText string `json:"new_text"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}
	resolved, err := f.ValidatePath(params.Path)
	if err != nil {
		return nil, err
	}
	// Context file write guard
	if f.agentType == "predefined" {
		base := filepath.Base(resolved)
		if predefinedImmutableFiles[base] {
			return nil, fmt.Errorf("cannot modify %s: immutable for predefined agents", base)
		}
		if base == "SOUL.md" && !f.selfEvolve {
			return nil, fmt.Errorf("cannot modify SOUL.md: self-evolution is disabled for this agent")
		}
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, err
	}
	content := string(data)
	if !strings.Contains(content, params.OldText) {
		return nil, fmt.Errorf("old_text not found in file")
	}
	content = strings.Replace(content, params.OldText, params.NewText, 1)
	if err := os.WriteFile(resolved, []byte(content), 0644); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]string{"status": "success"})
}
