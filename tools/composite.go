package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// Composite wraps multiple Tool implementations into one.
type Composite struct {
	tools []Tool
}

// NewComposite creates a Composite that delegates to the given tools in order.
func NewComposite(tools ...Tool) *Composite { return &Composite{tools: tools} }

// Definitions returns the merged definitions from all wrapped tools.
func (c *Composite) Definitions() []ToolDefinition {
	var defs []ToolDefinition
	for _, t := range c.tools {
		defs = append(defs, t.Definitions()...)
	}
	return defs
}

// Execute dispatches the call to whichever wrapped tool owns the named tool.
func (c *Composite) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	for _, t := range c.tools {
		for _, d := range t.Definitions() {
			if d.Name == name {
				return t.Execute(ctx, name, args)
			}
		}
	}
	return nil, fmt.Errorf("tool not found: %s", name)
}
