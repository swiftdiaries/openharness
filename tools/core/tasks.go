package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"

	"github.com/swiftdiaries/openharness/tools"
)

type TaskItem struct {
	ID          string `json:"id"`
	Subject     string `json:"subject"`
	Description string `json:"description,omitempty"`
	ActiveForm  string `json:"activeForm,omitempty"`
	Status      string `json:"status"`
}

type TaskCRUD struct {
	storePath string
	mu        sync.Mutex
}

func NewTaskCRUD(dataDir string) *TaskCRUD {
	return &TaskCRUD{storePath: filepath.Join(dataDir, "tasks.json")}
}

func (t *TaskCRUD) Definitions() []tools.ToolDefinition {
	return []tools.ToolDefinition{
		{
			Name:        "task_create",
			Description: "Create a new task. Returns the task with its assigned ID.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"subject": {"type": "string", "description": "Brief title for the task"},
					"description": {"type": "string", "description": "What needs to be done"},
					"activeForm": {"type": "string", "description": "Present-tense form shown while in progress (e.g. 'Running tests')"}
				},
				"required": ["subject"]
			}`),
			Effects: tools.ToolEffectMutate,
		},
		{
			Name:        "task_update",
			Description: "Update an existing task's status, subject, description, or activeForm.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"id": {"type": "string", "description": "Task ID to update"},
					"status": {"type": "string", "enum": ["pending", "in_progress", "completed", "cancelled"], "description": "New status"},
					"subject": {"type": "string", "description": "Updated subject"},
					"description": {"type": "string", "description": "Updated description"},
					"activeForm": {"type": "string", "description": "Updated active form text"}
				},
				"required": ["id"]
			}`),
			Effects: tools.ToolEffectMutate,
		},
		{
			Name:        "task_get",
			Description: "Get a single task by ID.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"id": {"type": "string", "description": "Task ID"}
				},
				"required": ["id"]
			}`),
			Effects: tools.ToolEffectRead,
		},
		{
			Name:        "task_list",
			Description: "List all tasks. Returns the full task list with current statuses.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
			Effects: tools.ToolEffectRead,
		},
	}
}

func (t *TaskCRUD) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch name {
	case "task_create":
		return t.create(args)
	case "task_update":
		return t.update(args)
	case "task_get":
		return t.get(args)
	case "task_list":
		return t.list()
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (t *TaskCRUD) create(args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		Subject     string `json:"subject"`
		Description string `json:"description"`
		ActiveForm  string `json:"activeForm"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	tasks := t.load()
	task := TaskItem{
		ID:          uuid.New().String()[:8],
		Subject:     params.Subject,
		Description: params.Description,
		ActiveForm:  params.ActiveForm,
		Status:      "pending",
	}
	tasks = append(tasks, task)
	if err := t.save(tasks); err != nil {
		return nil, err
	}
	return json.Marshal(task)
}

func (t *TaskCRUD) update(args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		ID          string  `json:"id"`
		Status      *string `json:"status"`
		Subject     *string `json:"subject"`
		Description *string `json:"description"`
		ActiveForm  *string `json:"activeForm"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	tasks := t.load()
	for i, task := range tasks {
		if task.ID == params.ID {
			if params.Status != nil {
				tasks[i].Status = *params.Status
			}
			if params.Subject != nil {
				tasks[i].Subject = *params.Subject
			}
			if params.Description != nil {
				tasks[i].Description = *params.Description
			}
			if params.ActiveForm != nil {
				tasks[i].ActiveForm = *params.ActiveForm
			}
			if err := t.save(tasks); err != nil {
				return nil, err
			}
			return json.Marshal(tasks[i])
		}
	}
	return nil, fmt.Errorf("task %q not found", params.ID)
}

func (t *TaskCRUD) get(args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	tasks := t.load()
	for _, task := range tasks {
		if task.ID == params.ID {
			return json.Marshal(task)
		}
	}
	return nil, fmt.Errorf("task %q not found", params.ID)
}

func (t *TaskCRUD) list() (json.RawMessage, error) {
	tasks := t.load()
	return json.Marshal(map[string]any{
		"tasks": tasks,
		"count": len(tasks),
	})
}

func (t *TaskCRUD) load() []TaskItem {
	data, err := os.ReadFile(t.storePath)
	if err != nil {
		return nil
	}
	var tasks []TaskItem
	json.Unmarshal(data, &tasks)
	return tasks
}

func (t *TaskCRUD) save(tasks []TaskItem) error {
	os.MkdirAll(filepath.Dir(t.storePath), 0755)
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(t.storePath, data, 0644)
}

func (t *TaskCRUD) Tasks() []TaskItem {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.load()
}
