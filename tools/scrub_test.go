package tools

import "testing"

func TestScrubCredentials(t *testing.T) {
	tests := []struct {
		name  string
		input string
		clean string // substring that should NOT appear in output
	}{
		{"OpenAI key", "Key is sk-abc123def456ghi789jkl012mno345", "sk-abc123"},
		{"Anthropic key", "Using sk-ant-abcdef1234567890-extra", "sk-ant-"},
		{"GitHub PAT", "Token: ghp_1234567890abcdefghijklmnopqrstuvwxyz", "ghp_"},
		{"AWS key", "AKIAIOSFODNN7EXAMPLE12", "AKIA"},
		{"connection string", "postgres://user:pass@host:5432/db", "postgres://"},
		{"api_key=value", "API_KEY=mysecretvalue123", "mysecretvalue"},
		{"bearer token", "Authorization: Bearer eyJ0eXAiOiJKV1Q", "eyJ0eX"},
		{"clean text", "The fund NAV is $1,234,567", ""},
		{"long hex", "Hash: " + "abcdef1234567890" + "abcdef1234567890" + "abcdef1234567890" + "abcdef1234567890", "abcdef1234567890abcdef1234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScrubCredentials(tt.input)
			if tt.clean != "" && contains(result, tt.clean) {
				t.Errorf("ScrubCredentials(%q) still contains %q:\n%s", tt.input, tt.clean, result)
			}
			if tt.clean == "" && result != tt.input {
				t.Errorf("clean text was modified: %q -> %q", tt.input, result)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
