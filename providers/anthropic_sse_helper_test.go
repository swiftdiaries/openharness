package providers

import (
	"fmt"
	"io"
)

// writeSSEEvent writes a single Server-Sent Event to w.
// Format: "event: <name>\ndata: <json>\n\n"
func writeSSEEvent(w io.Writer, event, jsonData string) error {
	_, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonData)
	return err
}

// writeAnthropicTextStream writes a minimal valid Anthropic streaming
// response consisting of a single text-delta content block.
// The pieces slice is joined into one assistant message.
func writeAnthropicTextStream(w io.Writer, pieces []string, inputTokens, outputTokens int) error {
	if err := writeSSEEvent(w, "message_start", fmt.Sprintf(
		`{"type":"message_start","message":{"id":"msg_stream_1","type":"message","role":"assistant","model":"claude-opus-4-6","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":%d,"output_tokens":0}}}`,
		inputTokens,
	)); err != nil {
		return err
	}
	if err := writeSSEEvent(w, "content_block_start",
		`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`); err != nil {
		return err
	}
	for _, piece := range pieces {
		payload := fmt.Sprintf(
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":%q}}`,
			piece,
		)
		if err := writeSSEEvent(w, "content_block_delta", payload); err != nil {
			return err
		}
	}
	if err := writeSSEEvent(w, "content_block_stop",
		`{"type":"content_block_stop","index":0}`); err != nil {
		return err
	}
	if err := writeSSEEvent(w, "message_delta", fmt.Sprintf(
		`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":%d}}`,
		outputTokens,
	)); err != nil {
		return err
	}
	return writeSSEEvent(w, "message_stop", `{"type":"message_stop"}`)
}
