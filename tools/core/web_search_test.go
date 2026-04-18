package core

import (
	"testing"

	"github.com/swiftdiaries/openharness/tools"
)

func TestWebSearchDefinitions(t *testing.T) {
	w := NewWebSearch()
	defs := w.Definitions()
	if len(defs) != 1 {
		t.Fatalf("want 1 definition, got %d", len(defs))
	}
	if defs[0].Name != "web_search" {
		t.Errorf("name = %q, want web_search", defs[0].Name)
	}
	if defs[0].Effects != tools.ToolEffectRead {
		t.Errorf("Effects = %v, want Read", defs[0].Effects)
	}
}
