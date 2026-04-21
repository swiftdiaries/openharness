package core

import "github.com/swiftdiaries/openharness/tools"

// SanitizeRead applies credential scrubbing to any string that a
// Read-effect tool is about to return to the LLM. Kept as a single
// chokepoint so future sanitization (e.g. PII redaction) extends in
// one place.
func SanitizeRead(s string) string {
	return tools.ScrubCredentials(s)
}

// SanitizeExternal applies scrubbing AND wraps the result in
// external-content markers. Use for content originating off-box (web,
// knowledge graph). For on-box reads (filesystem, memory, tasks, local
// exec), use SanitizeRead.
func SanitizeExternal(s string) string {
	return tools.WrapExternalContent(tools.ScrubCredentials(s))
}
