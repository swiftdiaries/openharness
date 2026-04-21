package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/swiftdiaries/openharness/tools"
)

// readOnlyCommands are inspection-only commands — safe for Read/Neutral effect.
var readOnlyCommands = map[string]bool{
	"ls": true, "cat": true, "grep": true, "rg": true,
	"git": true, "wc": true, "head": true, "tail": true,
	"date": true, "echo": true, "pwd": true, "find": true,
	"diff": true, "sort": true, "uniq": true, "cut": true,
	"tr": true, "jq": true,
	"which": true, "file": true, "stat": true, "tree": true,
	"du": true, "df": true, "uname": true, "whoami": true,
}

// mutatingCommands modify the workspace — require Mutate effect.
var mutatingCommands = map[string]bool{
	"mkdir": true, "cp": true, "mv": true, "touch": true,
}

// shellWrappers lists commands that can execute arbitrary code.
var shellWrappers = map[string]bool{
	"bash": true, "sh": true, "zsh": true, "fish": true,
	"python": true, "python3": true, "node": true, "ruby": true,
	"perl": true, "php": true, "curl": true, "wget": true,
	"env": true, "xargs": true, "sudo": true,
}

// Exec runs shell commands in a sandboxed workspace.
type Exec struct {
	workDir string
}

// NewExec creates an Exec tool with the given working directory.
func NewExec(workDir string) *Exec {
	return &Exec{workDir: workDir}
}

func (e *Exec) Definitions() []tools.ToolDefinition {
	readParams := json.RawMessage(`{
		"type": "object",
		"properties": {
			"command":    {"type": "string", "description": "Shell command to execute"},
			"timeout_ms": {"type": "integer", "description": "Timeout in milliseconds (default 30000, max 120000)"}
		},
		"required": ["command"]
	}`)
	return []tools.ToolDefinition{
		{
			Name:        "exec_read",
			Description: "Execute a read-only shell command (ls, cat, grep, git-status, etc.) in the workspace. Mutating commands are not allowed here — use exec_mutate.",
			Parameters:  readParams,
			Effects:     tools.ToolEffectNeutral,
		},
		{
			Name:        "exec_mutate",
			Description: "Execute a workspace-mutating shell command (mkdir, cp, mv, touch).",
			Parameters:  readParams,
			Effects:     tools.ToolEffectMutate,
		},
	}
}

// shellMetacharacters rejects any character that has special meaning to
// sh -c. Kept permissive on space/tab so normal args still parse.
var shellMetacharacters = []string{
	";", "|", "&", "`", "$(", "${", ">", "<",
	"\n", "\r", "*", "?", "{", "}", "~",
}

// allowedFor returns whether `command` is allowed under the given
// definition name (exec_read or exec_mutate). Shared metachar and
// deny-pattern checks still apply.
func allowedFor(name, command string) bool {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return false
	}
	// Reject shell metacharacters that enable command chaining/injection.
	for _, meta := range shellMetacharacters {
		if strings.Contains(trimmed, meta) {
			return false
		}
	}
	// Check deny patterns (catches dangerous args to allowed commands).
	if tools.MatchesDenyPattern(trimmed) {
		return false
	}
	parts := strings.Fields(trimmed)
	base := filepath.Base(parts[0])
	if shellWrappers[base] {
		return false
	}
	switch name {
	case "exec_read":
		return readOnlyCommands[base]
	case "exec_mutate":
		return mutatingCommands[base]
	}
	return false
}

// IsAllowed checks if a command is on either allow list. Exported for testing.
// Preserves the pre-split behavior where any safe command — read or mutate —
// returns true.
func IsAllowed(command string) bool {
	return allowedFor("exec_read", command) || allowedFor("exec_mutate", command)
}

func (e *Exec) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	if name != "exec_read" && name != "exec_mutate" {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}

	var params struct {
		Command   string `json:"command"`
		TimeoutMs int    `json:"timeout_ms"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	if !allowedFor(name, params.Command) {
		slog.Warn("security.exec_denied", "name", name, "command", params.Command)
		return nil, fmt.Errorf("command not allowed for %s: %q", name, params.Command)
	}

	timeout := 30 * time.Second
	if params.TimeoutMs > 0 {
		timeout = time.Duration(params.TimeoutMs) * time.Millisecond
	}
	if timeout > 120*time.Second {
		timeout = 120 * time.Second
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "sh", "-c", params.Command)
	cmd.Dir = e.workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("exec: %w", err)
		}
	}

	result := map[string]interface{}{
		"stdout":    SanitizeRead(stdout.String()),
		"stderr":    SanitizeRead(stderr.String()),
		"exit_code": exitCode,
	}
	return json.Marshal(result)
}
