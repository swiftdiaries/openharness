package core

import (
	"strings"
	"testing"
)

func TestSanitizeRead_ScrubsCredentials(t *testing.T) {
	in := "Before sk-1234567890abcdefghij After"
	out := SanitizeRead(in)
	if strings.Contains(out, "sk-1234567890abcdefghij") {
		t.Fatalf("expected credential scrubbed, got %q", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Fatalf("expected [REDACTED] marker, got %q", out)
	}
}

func TestSanitizeRead_PreservesNormalText(t *testing.T) {
	in := "hello world"
	if SanitizeRead(in) != "hello world" {
		t.Fatal("plain text should pass through")
	}
}

func TestSanitizeExternal_WrapsAndScrubs(t *testing.T) {
	in := "Before sk-1234567890abcdefghij After"
	out := SanitizeExternal(in)
	if strings.Contains(out, "sk-1234567890abcdefghij") {
		t.Fatal("credential should be scrubbed inside wrapper")
	}
	if !strings.Contains(out, "EXTERNAL_UNTRUSTED_CONTENT") {
		t.Fatal("expected external-content marker")
	}
}
