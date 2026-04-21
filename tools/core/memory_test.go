package core

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/swiftdiaries/openharness/agent"
	"github.com/swiftdiaries/openharness/tools"
)

// Compile-time assertion that *Memory satisfies agent.MemoryStore.
// If the interface grows or Search's signature drifts, this line
// won't compile and the test file will fail to build.
var _ agent.MemoryStore = (*Memory)(nil)

func newMemory(t *testing.T) *Memory {
	t.Helper()
	path := filepath.Join(t.TempDir(), "memory.json")
	return NewMemory(path)
}

func TestMemoryStoreAndSearch_RoundTrip(t *testing.T) {
	m := newMemory(t)
	ctx := context.Background()
	// Store via Execute so we exercise the JSON schema path.
	args, _ := json.Marshal(map[string]string{
		"key":   "fav_color",
		"value": "blue",
	})
	if _, err := m.Execute(ctx, "memory_store", args); err != nil {
		t.Fatal(err)
	}
	// New Memory on same file → Search still finds it.
	m2 := NewMemory(m.path)
	results, err := m2.Search(ctx, "blue", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if results[0].Key != "fav_color" || results[0].Value != "blue" {
		t.Errorf("got %+v", results[0])
	}
	if results[0].UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set by memory_store")
	}
}

func TestMemoryOnDiskFormat(t *testing.T) {
	m := newMemory(t)
	ctx := context.Background()
	args, _ := json.Marshal(map[string]string{"key": "k", "value": "v"})
	if _, err := m.Execute(ctx, "memory_store", args); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(m.path)
	if err != nil {
		t.Fatal(err)
	}
	// On-disk format must be agent.MemoryEntry — Key/Value/UpdatedAt only.
	var decoded map[string]struct {
		Key       string `json:"Key"`
		Value     string `json:"Value"`
		UpdatedAt string `json:"UpdatedAt"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("on-disk file not decodable with agent.MemoryEntry shape: %v\nraw: %s", err, raw)
	}
	e, ok := decoded["k"]
	if !ok || e.Key != "k" || e.Value != "v" || e.UpdatedAt == "" {
		t.Errorf("decoded entry: %+v", e)
	}
}

func TestMemorySearchSubstringSemantics(t *testing.T) {
	m := newMemory(t)
	ctx := context.Background()

	for _, kv := range [][2]string{
		{"fruit_apple", "red and round"},
		{"fruit_banana", "yellow and curved"},
		{"veg_carrot", "orange and long"},
		{"color_red", "rhymes with bed"},
	} {
		args, _ := json.Marshal(map[string]string{"key": kv[0], "value": kv[1]})
		if _, err := m.Execute(ctx, "memory_store", args); err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		query string
		want  []string // keys expected in the result set (order-independent)
	}{
		{"red", []string{"fruit_apple", "color_red"}},
		{"APPLE", []string{"fruit_apple"}},
		{"yellow", []string{"fruit_banana"}},
		{"fruit", []string{"fruit_apple", "fruit_banana"}},
		{"nomatch", nil},
	}
	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			got, err := m.Search(ctx, c.query, 10)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(c.want) {
				t.Fatalf("query %q: got %d results, want %d: %+v", c.query, len(got), len(c.want), got)
			}
			seen := map[string]bool{}
			for _, e := range got {
				seen[e.Key] = true
			}
			for _, k := range c.want {
				if !seen[k] {
					t.Errorf("query %q: expected key %q in results", c.query, k)
				}
			}
		})
	}
}

func TestMemorySearchHonorsLimit(t *testing.T) {
	m := newMemory(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		args, _ := json.Marshal(map[string]string{
			"key":   "k" + string(rune('0'+i)),
			"value": "hello world",
		})
		m.Execute(ctx, "memory_store", args)
	}
	got, err := m.Search(ctx, "hello", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Errorf("want 3 results, got %d", len(got))
	}
}

func TestMemoryExecuteRejectsCategoryParameter(t *testing.T) {
	m := newMemory(t)
	defs := m.Definitions()
	for _, d := range defs {
		if d.Name == "memory_store" {
			var schema struct {
				Properties map[string]any `json:"properties"`
			}
			if err := json.Unmarshal(d.Parameters, &schema); err != nil {
				t.Fatal(err)
			}
			if _, has := schema.Properties["category"]; has {
				t.Error("memory_store schema must not include 'category'")
			}
		}
	}
}

func TestMemoryDefinitionsEffects(t *testing.T) {
	m := newMemory(t)
	want := map[string]tools.ToolEffect{
		"memory_store":  tools.ToolEffectMutate,
		"memory_search": tools.ToolEffectRead,
		"memory_get":    tools.ToolEffectRead,
	}
	for _, d := range m.Definitions() {
		if got := d.Effects; got != want[d.Name] {
			t.Errorf("%s: Effects = %v, want %v", d.Name, got, want[d.Name])
		}
	}
}

func TestMemory_FileMode0600(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "memory.json")
	m := NewMemory(path)
	args, _ := json.Marshal(map[string]string{"key": "k", "value": "v"})
	if _, err := m.Execute(context.Background(), "memory_store", args); err != nil {
		t.Fatalf("store: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600, got %o", info.Mode().Perm())
	}
}
