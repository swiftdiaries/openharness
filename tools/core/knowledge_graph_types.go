package core

import "time"

// KGBlock is the atomic content unit in an openharness knowledge graph.
// The field set mirrors ghostfin's internal/notes.Block exactly so Plan 7's
// adapter is a pure field-for-field copy. Future verticals whose knowledge
// schema diverges can still populate these types without embedding.
type KGBlock struct {
	ID         string
	ParentID   *string
	PageID     string
	Title      string
	Order      int
	Collapsed  bool
	Properties map[string]string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// KGPage is a page in the knowledge graph. Fields are flattened rather than
// embedded so openharness does not assume Page structurally specializes Block.
type KGPage struct {
	// Block-level fields (flattened from KGBlock).
	ID         string
	ParentID   *string
	PageID     string
	Title      string
	Order      int
	Collapsed  bool
	Properties map[string]string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	// Page-only fields.
	Name       string
	JournalDay *string
	Icon       string
}

// KGRef is a reference edge between blocks. RefType values in ghostfin today
// are "page_ref", "tag", "block_ref"; the field is an opaque string so other
// verticals can define their own taxonomies.
type KGRef struct {
	SourceBlockID string
	TargetPageID  string
	RefType       string
}
