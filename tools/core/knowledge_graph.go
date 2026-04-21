package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/swiftdiaries/openharness/tools"
)

// KnowledgeGraphStore is the rich interface the knowledge_graph tool calls
// against. It is intentionally broader than agent.KnowledgeStore (which
// ships a two-method Search/Get surface for Plan 5's retrieval layer).
// Different consumers want different contracts — keeping them separate
// avoids forcing every future vertical to implement 8 methods when the
// retrieval layer only needs 2.
type KnowledgeGraphStore interface {
	Search(ctx context.Context, query string, limit int) ([]KGBlock, error)
	SearchInPage(ctx context.Context, pageID, query string, limit int) ([]KGBlock, error)
	GetPage(ctx context.Context, name string) (*KGPage, error)
	GetPageByID(ctx context.Context, id string) (*KGPage, error)
	GetBlockTree(ctx context.Context, pageID string) ([]KGBlock, error)
	GetBlockSnippets(ctx context.Context, pageID string, limit int) ([]string, error)
	GetRelatedPages(ctx context.Context, pageID string, depth int) ([]KGPage, []KGRef, error)
	GetPagesWithSharedTags(ctx context.Context, pageID string, limit int) ([]KGPage, error)
}

// KnowledgeGraph is the built-in tool that exposes a user's markdown/block
// knowledge store to the agent as a single knowledge_graph_search tool with
// four modes: search (full-text), traverse (connected pages via refs),
// page (fetch content), related (shared-tag pages).
type KnowledgeGraph struct {
	store KnowledgeGraphStore
}

// NewKnowledgeGraph creates a KnowledgeGraph tool backed by the given store.
func NewKnowledgeGraph(store KnowledgeGraphStore) *KnowledgeGraph {
	return &KnowledgeGraph{store: store}
}

type kgArgs struct {
	Query      string `json:"query"`
	Page       string `json:"page,omitempty"`
	Mode       string `json:"mode,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
}

func (kg *KnowledgeGraph) Definitions() []tools.ToolDefinition {
	return []tools.ToolDefinition{
		{
			Name:        "knowledge_graph_search",
			Description: "Search the user's notes knowledge graph. Modes: 'search' for full-text keyword search, 'traverse' to find connected pages via refs, 'page' to fetch a page's content, 'related' for pages sharing tags.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "Search query or page name"},
					"page":  {"type": "string", "description": "Optional page name to scope search to"},
					"mode":  {"type": "string", "enum": ["search", "traverse", "page", "related"], "description": "Search mode (default: search)"},
					"max_results": {"type": "integer", "description": "Max results to return (default 10, max 100)"}
				},
				"required": ["query"]
			}`),
			Effects: tools.ToolEffectRead,
		},
	}
}

func (kg *KnowledgeGraph) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	if name != "knowledge_graph_search" {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}

	var params kgArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if params.Mode == "" {
		params.Mode = "search"
	}
	if params.MaxResults <= 0 {
		if params.Mode == "page" {
			params.MaxResults = 100
		} else {
			params.MaxResults = 10
		}
	}
	if params.MaxResults > 100 {
		params.MaxResults = 100
	}

	var (
		result string
		err    error
	)
	switch params.Mode {
	case "search":
		result, err = kg.execSearch(ctx, params)
	case "traverse":
		result, err = kg.execTraverse(ctx, params)
	case "page":
		result, err = kg.execPage(ctx, params)
	case "related":
		result, err = kg.execRelated(ctx, params)
	default:
		return nil, fmt.Errorf("unknown mode: %s", params.Mode)
	}
	if err != nil {
		return nil, err
	}
	return json.Marshal(SanitizeExternal(result))
}

func (kg *KnowledgeGraph) execSearch(ctx context.Context, params kgArgs) (string, error) {
	var blocks []KGBlock
	if params.Page != "" {
		page, err := kg.store.GetPage(ctx, strings.ToLower(params.Page))
		if err != nil {
			return "", fmt.Errorf("get page: %w", err)
		}
		if page == nil {
			return fmt.Sprintf("Page %q not found.", params.Page), nil
		}
		inPageBlocks, err := kg.store.SearchInPage(ctx, page.ID, params.Query, params.MaxResults)
		if err != nil {
			return "", fmt.Errorf("search in page: %w", err)
		}
		blocks = inPageBlocks
	} else {
		searched, err := kg.store.Search(ctx, params.Query, params.MaxResults)
		if err != nil {
			return "", fmt.Errorf("search: %w", err)
		}
		blocks = searched
	}

	if len(blocks) == 0 {
		return fmt.Sprintf("No results for %q.", params.Query), nil
	}

	grouped := map[string][]KGBlock{}
	var pageOrder []string
	for _, b := range blocks {
		if _, seen := grouped[b.PageID]; !seen {
			pageOrder = append(pageOrder, b.PageID)
		}
		grouped[b.PageID] = append(grouped[b.PageID], b)
	}

	var sb strings.Builder
	for _, pid := range pageOrder {
		page, _ := kg.store.GetPageByID(ctx, pid)
		pageName := pid
		if page != nil {
			pageName = page.Name
		}
		fmt.Fprintf(&sb, "## %s\n", pageName)
		for _, b := range grouped[pid] {
			fmt.Fprintf(&sb, "- %s\n", b.Title)
		}
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String()), nil
}

func (kg *KnowledgeGraph) execTraverse(ctx context.Context, params kgArgs) (string, error) {
	pageName := strings.TrimSuffix(strings.ToLower(params.Query), ".md")
	page, err := kg.store.GetPage(ctx, pageName)
	if err != nil {
		return "", fmt.Errorf("get page: %w", err)
	}
	if page == nil {
		return fmt.Sprintf("Page %q not found.", params.Query), nil
	}

	depth := 2
	if params.MaxResults <= 5 {
		depth = 1
	}

	related, refs, err := kg.store.GetRelatedPages(ctx, page.ID, depth)
	if err != nil {
		return "", fmt.Errorf("get related pages: %w", err)
	}
	if len(related) == 0 {
		return fmt.Sprintf("No pages connected to %q.", pageName), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Pages connected to %q:\n", pageName)
	pageRefs := map[string][]string{}
	for _, r := range refs {
		pageRefs[r.TargetPageID] = append(pageRefs[r.TargetPageID], r.RefType)
	}

	count := 0
	for _, p := range related {
		if count >= params.MaxResults {
			break
		}
		refTypes := uniqueStrings(pageRefs[p.ID])
		if len(refTypes) > 0 {
			fmt.Fprintf(&sb, "## %s (via %s)\n", p.Name, strings.Join(refTypes, ", "))
		} else {
			fmt.Fprintf(&sb, "## %s\n", p.Name)
		}
		snippets, _ := kg.store.GetBlockSnippets(ctx, p.ID, 3)
		for _, s := range snippets {
			if len(s) > 200 {
				s = s[:200] + "..."
			}
			fmt.Fprintf(&sb, "- %s\n", s)
		}
		sb.WriteString("\n")
		count++
	}
	return strings.TrimSpace(sb.String()), nil
}

func (kg *KnowledgeGraph) execPage(ctx context.Context, params kgArgs) (string, error) {
	pageName := strings.TrimSuffix(strings.ToLower(params.Query), ".md")
	page, err := kg.store.GetPage(ctx, pageName)
	if err != nil {
		return "", fmt.Errorf("get page: %w", err)
	}
	if page == nil {
		return fmt.Sprintf("Page %q not found.", pageName), nil
	}

	blocks, err := kg.store.GetBlockTree(ctx, page.ID)
	if err != nil {
		return "", fmt.Errorf("get block tree: %w", err)
	}

	type node struct {
		block    KGBlock
		children []*node
	}
	nodeMap := map[string]*node{}
	var roots []*node
	for i := range blocks {
		nodeMap[blocks[i].ID] = &node{block: blocks[i]}
	}
	for i := range blocks {
		n := nodeMap[blocks[i].ID]
		if blocks[i].ParentID != nil {
			if parent, ok := nodeMap[*blocks[i].ParentID]; ok {
				parent.children = append(parent.children, n)
				continue
			}
		}
		roots = append(roots, n)
	}
	if params.MaxResults < len(roots) {
		roots = roots[:params.MaxResults]
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s\n", page.Name)
	var writeNode func(n *node, depth int)
	writeNode = func(n *node, depth int) {
		indent := strings.Repeat("  ", depth)
		fmt.Fprintf(&sb, "%s- %s\n", indent, n.block.Title)
		for _, c := range n.children {
			writeNode(c, depth+1)
		}
	}
	for _, r := range roots {
		writeNode(r, 0)
	}
	return strings.TrimSpace(sb.String()), nil
}

func (kg *KnowledgeGraph) execRelated(ctx context.Context, params kgArgs) (string, error) {
	pageName := strings.TrimSuffix(strings.ToLower(params.Query), ".md")
	page, err := kg.store.GetPage(ctx, pageName)
	if err != nil {
		return "", fmt.Errorf("get page: %w", err)
	}
	if page == nil {
		return fmt.Sprintf("Page %q not found.", params.Query), nil
	}
	related, err := kg.store.GetPagesWithSharedTags(ctx, page.ID, params.MaxResults)
	if err != nil {
		return "", fmt.Errorf("get related pages: %w", err)
	}
	if len(related) == 0 {
		return fmt.Sprintf("No pages share tags with %q.", pageName), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Pages sharing tags with %q:\n\n", pageName)
	for _, p := range related {
		fmt.Fprintf(&sb, "## %s\n", p.Name)
		snippets, _ := kg.store.GetBlockSnippets(ctx, p.ID, 3)
		for _, s := range snippets {
			if len(s) > 200 {
				s = s[:200] + "..."
			}
			fmt.Fprintf(&sb, "- %s\n", s)
		}
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String()), nil
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
