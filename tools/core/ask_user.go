package core

import (
	"context"
	"encoding/json"

	"github.com/swiftdiaries/openharness/tools"
)

type AskUser struct{}

func NewAskUser() *AskUser { return &AskUser{} }

func (a *AskUser) Definitions() []tools.ToolDefinition {
	return []tools.ToolDefinition{
		{
			Name:        "ask_user_question",
			Description: "Ask the user a question and wait for their response. Use this when there is ambiguity in the task, to clarify requirements, confirm an approach, or request approval before proceeding. Prefer asking over guessing.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"question": {
						"type": "string",
						"description": "The question to ask the user"
					},
					"is_plan": {
						"type": "boolean",
						"description": "Set to true when presenting a numbered execution plan for user approval. Leave false or omit for clarifying questions, follow-ups, or any non-plan content."
					},
					"suggestions": {
						"type": "array",
						"items": { "type": "string" },
						"description": "Optional list of suggested answers to present as clickable chips. Each string is one complete option. Omit if the question is open-ended."
					}
				},
				"required": ["question"]
			}`),
			Effects: tools.ToolEffectInteractive,
		},
	}
}

func (a *AskUser) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]string{"status": "waiting_for_user"})
}
