package tools

import "testing"

func TestShellDenyPatterns(t *testing.T) {
	tests := []struct {
		name    string
		command string
		denied  bool
	}{
		{"safe ls", "ls -la", false},
		{"safe git status", "git status", false},
		{"rm -rf", "rm -rf /", true},
		{"rm -rf home", "rm -rf ~", true},
		{"dd if", "dd if=/dev/zero of=/dev/sda", true},
		{"mkfs", "mkfs.ext4 /dev/sda", true},
		{"shutdown", "shutdown -h now", true},
		{"fork bomb", ":(){ :|:& };:", true},
		{"git push force", "git push --force origin main", true},
		{"cp overwrite system", "cp /dev/null /etc/passwd", false},
		{"safe git log", "git log --oneline", false},
		{"safe find", "find . -name '*.go'", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			denied := MatchesDenyPattern(tt.command)
			if denied != tt.denied {
				t.Errorf("MatchesDenyPattern(%q) = %v, want %v", tt.command, denied, tt.denied)
			}
		})
	}
}
