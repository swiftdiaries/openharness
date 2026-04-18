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

// allowedCommands lists commands that are permitted to run.
var allowedCommands = map[string]bool{
	"ls": true, "cat": true, "grep": true, "rg": true,
	"git": true, "wc": true, "head": true, "tail": true,
	"date": true, "echo": true, "pwd": true, "find": true,
	"diff": true, "sort": true, "uniq": true, "cut": true,
	"tr": true, "jq": true,
	"which": true, "file": true, "stat": true, "tree": true,
	"du": true, "df": true, "uname": true, "whoami": true,
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
	return []tools.ToolDefinition{
		{
			Name:        "exec",
			Description: "Execute a shell command in the workspace. Dangerous commands are blocked.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"command": {"type": "string", "description": "Shell command to execute"},
					"timeout_ms": {"type": "integer", "description": "Timeout in milliseconds (default 30000, max 120000)"}
				},
				"required": ["command"]
			}`),
			Effects: tools.ToolEffectNeutral,
		},
	}
}

// shellMetacharacters contains patterns that allow shell injection when
// commands are passed to sh -c. We reject these before checking the allowlist.
var shellMetacharacters = []string{";", "|", "&&", "||", "`", "$(", ">", "<"}

// IsAllowed checks if a command is on the allow list. Exported for testing.
func IsAllowed(command string) bool {
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
	return allowedCommands[base]
}

func (e *Exec) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	if name != "exec" {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}

	var params struct {
		Command   string `json:"command"`
		TimeoutMs int    `json:"timeout_ms"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	if !IsAllowed(params.Command) {
		slog.Warn("security.exec_denied", "command", params.Command)
		return nil, fmt.Errorf("command not allowed: %q", params.Command)
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
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"exit_code": exitCode,
	}
	return json.Marshal(result)
}
