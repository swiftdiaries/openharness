package tools

import "regexp"

// shellDenyPatterns catches dangerous arguments to otherwise-allowed commands.
// This is Layer 3b of the defense-in-depth security model: even if a command
// passes the allowlist, these patterns block destructive argument combinations.
var shellDenyPatterns = []*regexp.Regexp{
	// Destructive file operations
	regexp.MustCompile(`(?i)\brm\b.*\s+-[a-zA-Z]*r[a-zA-Z]*f`),
	regexp.MustCompile(`(?i)\brm\b.*\s+-[a-zA-Z]*f[a-zA-Z]*r`),
	regexp.MustCompile(`(?i)\brmdir\b.*\s+/s`),

	// Destructive disk operations
	regexp.MustCompile(`(?i)\bmkfs\b`),
	regexp.MustCompile(`(?i)\bdd\b.*\bif=`),

	// System commands
	regexp.MustCompile(`(?i)\b(shutdown|reboot|poweroff|halt|init\s+[06])\b`),

	// Fork bombs
	regexp.MustCompile(`:\(\)\s*\{`),
	regexp.MustCompile(`\.\(\)\s*\{`),

	// Git destructive operations
	regexp.MustCompile(`(?i)\bgit\b.*\b(push\s+--force|reset\s+--hard|clean\s+-[a-zA-Z]*f)`),
}

// matchesDenyPattern returns true if the command matches any shell deny pattern.
func matchesDenyPattern(command string) bool {
	for _, pat := range shellDenyPatterns {
		if pat.MatchString(command) {
			return true
		}
	}
	return false
}
