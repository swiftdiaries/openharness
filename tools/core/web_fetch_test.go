package core

import (
	"testing"

	"github.com/swiftdiaries/openharness/tools"
)

func TestWebFetchDefinitions(t *testing.T) {
	w := NewWebFetch()
	defs := w.Definitions()
	if len(defs) == 0 {
		t.Fatal("want at least 1 definition, got 0")
	}
	for _, d := range defs {
		if d.Effects != tools.ToolEffectRead {
			t.Errorf("%s: Effects = %v, want Read", d.Name, d.Effects)
		}
	}
}
