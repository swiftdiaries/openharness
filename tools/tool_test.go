package tools

import (
	"encoding/json"
	"testing"
)

func TestToolEffectZeroValueIsUnspecified(t *testing.T) {
	var e ToolEffect
	if e != ToolEffectUnspecified {
		t.Fatalf("zero value of ToolEffect should be ToolEffectUnspecified, got %d", int(e))
	}
}

func TestToolDefinitionEffectsNotSerialized(t *testing.T) {
	def := ToolDefinition{
		Name:        "example",
		Description: "x",
		Parameters:  json.RawMessage(`{}`),
		Effects:     ToolEffectMutate,
	}
	data, err := json.Marshal(def)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if containsAny(s, []string{"Effects", "effects"}) {
		t.Fatalf("Effects must not appear in JSON output, got: %s", s)
	}
}

func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		for i := 0; i+len(n) <= len(s); i++ {
			if s[i:i+len(n)] == n {
				return true
			}
		}
	}
	return false
}
