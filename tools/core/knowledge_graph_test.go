package core

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/swiftdiaries/openharness/tools"
)

// fakeKG is an in-memory KnowledgeGraphStore for tests.
type fakeKG struct {
	pages  map[string]KGPage    // keyed by Name (lowercase)
	byID   map[string]KGPage    // keyed by ID
	blocks map[string][]KGBlock // keyed by PageID
}

func newFakeKG() *fakeKG {
	now := time.Now()
	p := KGPage{
		ID: "p1", PageID: "p1", Title: "stocks_watchlist",
		CreatedAt: now, UpdatedAt: now, Name: "stocks_watchlist",
	}
	return &fakeKG{
		pages: map[string]KGPage{"stocks_watchlist": p},
		byID:  map[string]KGPage{"p1": p},
		blocks: map[string][]KGBlock{
			"p1": {
				{ID: "b1", PageID: "p1", Title: "AAPL 150 shares at $185", Order: 0, CreatedAt: now, UpdatedAt: now},
				{ID: "b2", PageID: "p1", Title: "GOOGL 50 shares at $142", Order: 1, CreatedAt: now, UpdatedAt: now},
				{ID: "b3", PageID: "p1", Title: "Review Q1 performance", Order: 2, CreatedAt: now, UpdatedAt: now},
			},
		},
	}
}

func (f *fakeKG) Search(ctx context.Context, query string, limit int) ([]KGBlock, error) {
	q := strings.ToLower(query)
	var out []KGBlock
	for _, blks := range f.blocks {
		for _, b := range blks {
			if strings.Contains(strings.ToLower(b.Title), q) {
				out = append(out, b)
				if len(out) >= limit {
					return out, nil
				}
			}
		}
	}
	return out, nil
}
func (f *fakeKG) SearchInPage(ctx context.Context, pageID, query string, limit int) ([]KGBlock, error) {
	q := strings.ToLower(query)
	var out []KGBlock
	for _, b := range f.blocks[pageID] {
		if strings.Contains(strings.ToLower(b.Title), q) {
			out = append(out, b)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}
func (f *fakeKG) GetPage(ctx context.Context, name string) (*KGPage, error) {
	if p, ok := f.pages[name]; ok {
		return &p, nil
	}
	return nil, nil
}
func (f *fakeKG) GetPageByID(ctx context.Context, id string) (*KGPage, error) {
	if p, ok := f.byID[id]; ok {
		return &p, nil
	}
	return nil, nil
}
func (f *fakeKG) GetBlockTree(ctx context.Context, pageID string) ([]KGBlock, error) {
	return f.blocks[pageID], nil
}
func (f *fakeKG) GetBlockSnippets(ctx context.Context, pageID string, limit int) ([]string, error) {
	var out []string
	for _, b := range f.blocks[pageID] {
		out = append(out, b.Title)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
func (f *fakeKG) GetRelatedPages(ctx context.Context, pageID string, depth int) ([]KGPage, []KGRef, error) {
	return nil, nil, nil
}
func (f *fakeKG) GetPagesWithSharedTags(ctx context.Context, pageID string, limit int) ([]KGPage, error) {
	return nil, nil
}

func TestKnowledgeGraphDefinitions(t *testing.T) {
	kg := NewKnowledgeGraph(newFakeKG())
	defs := kg.Definitions()
	if len(defs) != 1 {
		t.Fatalf("want 1 definition, got %d", len(defs))
	}
	if defs[0].Name != "knowledge_graph_search" {
		t.Errorf("name = %q, want knowledge_graph_search", defs[0].Name)
	}
	if defs[0].Effects != tools.ToolEffectRead {
		t.Errorf("Effects = %v, want Read", defs[0].Effects)
	}
}

func TestKnowledgeGraphSearch_SearchMode(t *testing.T) {
	kg := NewKnowledgeGraph(newFakeKG())
	args, _ := json.Marshal(map[string]any{"query": "AAPL", "mode": "search"})
	result, err := kg.Execute(context.Background(), "knowledge_graph_search", args)
	if err != nil {
		t.Fatal(err)
	}
	var text string
	json.Unmarshal(result, &text)
	if !strings.Contains(text, "AAPL") {
		t.Fatalf("expected AAPL in result: %s", text)
	}
	if !strings.Contains(text, "stocks_watchlist") {
		t.Fatalf("expected stocks_watchlist in result: %s", text)
	}
}

func TestKnowledgeGraphSearch_PageMode(t *testing.T) {
	kg := NewKnowledgeGraph(newFakeKG())
	args, _ := json.Marshal(map[string]any{"query": "stocks_watchlist", "mode": "page"})
	result, err := kg.Execute(context.Background(), "knowledge_graph_search", args)
	if err != nil {
		t.Fatal(err)
	}
	var text string
	json.Unmarshal(result, &text)
	if !strings.Contains(text, "AAPL") || !strings.Contains(text, "GOOGL") {
		t.Fatalf("expected page content: %s", text)
	}
}
