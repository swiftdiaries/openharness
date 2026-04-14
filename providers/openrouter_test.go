package providers

import (
	"encoding/json"
	"testing"
)

func TestChatRequestFormat(t *testing.T) {
	req := openAIRequest{
		Model: "anthropic/claude-sonnet-4-6",
		Messages: []Message{
			NewTextMessage("system", "You are helpful."),
			NewTextMessage("user", "Hello"),
		},
		Tools: []ToolDefinition{
			{
				Type: "function",
				Function: ToolFunctionSchema{
					Name:        "test_tool",
					Description: "A test tool",
					Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
				},
			},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if parsed["model"] != "anthropic/claude-sonnet-4-6" {
		t.Errorf("model = %v, want anthropic/claude-sonnet-4-6", parsed["model"])
	}

	messages, ok := parsed["messages"].([]interface{})
	if !ok || len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %v", parsed["messages"])
	}

	tools, ok := parsed["tools"].([]interface{})
	if !ok || len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %v", parsed["tools"])
	}

	tool := tools[0].(map[string]interface{})
	if tool["type"] != "function" {
		t.Errorf("tool type = %v, want function", tool["type"])
	}

	fn := tool["function"].(map[string]interface{})
	if fn["name"] != "test_tool" {
		t.Errorf("tool name = %v, want test_tool", fn["name"])
	}
}

func TestNewTextMessage(t *testing.T) {
	msg := NewTextMessage("user", "hello world")
	if msg.Role != "user" {
		t.Errorf("role = %q, want user", msg.Role)
	}

	var content string
	if err := json.Unmarshal(msg.Content, &content); err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}
	if content != "hello world" {
		t.Errorf("content = %q, want 'hello world'", content)
	}
}

func TestResolveAPIKey_Plaintext(t *testing.T) {
	key := ResolveAPIKey("my-key", "UNUSED_ENV")
	if key != "my-key" {
		t.Errorf("expected my-key, got %q", key)
	}
}

func TestResolveAPIKey_Empty(t *testing.T) {
	t.Setenv("TEST_API_KEY", "from-env")
	key := ResolveAPIKey("", "TEST_API_KEY")
	if key != "from-env" {
		t.Errorf("expected from-env, got %q", key)
	}
}

func TestResolveAPIKey_EnvVar(t *testing.T) {
	t.Setenv("MY_KEY", "env-value")
	key := ResolveAPIKey("$MY_KEY", "UNUSED")
	if key != "env-value" {
		t.Errorf("expected env-value, got %q", key)
	}
}
