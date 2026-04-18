package core

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/swiftdiaries/openharness/tools"
)

func TestTaskCreate(t *testing.T) {
	dir := t.TempDir()
	tc := NewTaskCRUD(dir)

	args := `{"subject":"Write tests","description":"Write unit tests for task CRUD","activeForm":"Writing tests"}`
	result, err := tc.Execute(context.Background(), "task_create", json.RawMessage(args))
	if err != nil {
		t.Fatalf("task_create: %v", err)
	}

	var resp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	json.Unmarshal(result, &resp)
	if resp.ID == "" {
		t.Error("expected non-empty task ID")
	}
	if resp.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", resp.Status)
	}

	// Verify persisted to disk
	data, err := os.ReadFile(filepath.Join(dir, "tasks.json"))
	if err != nil {
		t.Fatalf("read tasks.json: %v", err)
	}
	var stored []TaskItem
	json.Unmarshal(data, &stored)
	if len(stored) != 1 {
		t.Fatalf("expected 1 stored task, got %d", len(stored))
	}
	if stored[0].Subject != "Write tests" {
		t.Errorf("expected subject 'Write tests', got %q", stored[0].Subject)
	}
}

func TestTaskUpdate(t *testing.T) {
	dir := t.TempDir()
	tc := NewTaskCRUD(dir)

	createArgs := `{"subject":"Build feature"}`
	createResult, _ := tc.Execute(context.Background(), "task_create", json.RawMessage(createArgs))
	var created struct{ ID string `json:"id"` }
	json.Unmarshal(createResult, &created)

	updateArgs := `{"id":"` + created.ID + `","status":"in_progress","activeForm":"Building feature"}`
	result, err := tc.Execute(context.Background(), "task_update", json.RawMessage(updateArgs))
	if err != nil {
		t.Fatalf("task_update: %v", err)
	}

	var resp struct{ Status string `json:"status"` }
	json.Unmarshal(result, &resp)
	if resp.Status != "in_progress" {
		t.Errorf("expected 'in_progress', got %q", resp.Status)
	}
}

func TestTaskUpdateNotFound(t *testing.T) {
	dir := t.TempDir()
	tc := NewTaskCRUD(dir)

	args := `{"id":"nonexistent","status":"completed"}`
	_, err := tc.Execute(context.Background(), "task_update", json.RawMessage(args))
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestTaskGet(t *testing.T) {
	dir := t.TempDir()
	tc := NewTaskCRUD(dir)

	createArgs := `{"subject":"Test task","description":"A test"}`
	createResult, _ := tc.Execute(context.Background(), "task_create", json.RawMessage(createArgs))
	var created struct{ ID string `json:"id"` }
	json.Unmarshal(createResult, &created)

	getArgs := `{"id":"` + created.ID + `"}`
	result, err := tc.Execute(context.Background(), "task_get", json.RawMessage(getArgs))
	if err != nil {
		t.Fatalf("task_get: %v", err)
	}

	var task TaskItem
	json.Unmarshal(result, &task)
	if task.Subject != "Test task" {
		t.Errorf("expected 'Test task', got %q", task.Subject)
	}
}

func TestTaskList(t *testing.T) {
	dir := t.TempDir()
	tc := NewTaskCRUD(dir)

	tc.Execute(context.Background(), "task_create", json.RawMessage(`{"subject":"First"}`))
	tc.Execute(context.Background(), "task_create", json.RawMessage(`{"subject":"Second"}`))

	result, err := tc.Execute(context.Background(), "task_list", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("task_list: %v", err)
	}

	var resp struct {
		Tasks []TaskItem `json:"tasks"`
		Count int        `json:"count"`
	}
	json.Unmarshal(result, &resp)
	if resp.Count != 2 {
		t.Errorf("expected 2 tasks, got %d", resp.Count)
	}
}

func TestTaskDefinitions(t *testing.T) {
	tc := NewTaskCRUD(t.TempDir())
	defs := tc.Definitions()
	names := make(map[string]bool)
	for _, d := range defs {
		names[d.Name] = true
	}
	for _, expected := range []string{"task_create", "task_update", "task_get", "task_list"} {
		if !names[expected] {
			t.Errorf("missing definition for %q", expected)
		}
	}
}

func TestTaskPersistenceAcrossInstances(t *testing.T) {
	dir := t.TempDir()

	tc1 := NewTaskCRUD(dir)
	tc1.Execute(context.Background(), "task_create", json.RawMessage(`{"subject":"Persistent task"}`))

	tc2 := NewTaskCRUD(dir)
	result, _ := tc2.Execute(context.Background(), "task_list", json.RawMessage(`{}`))
	var resp struct {
		Tasks []TaskItem `json:"tasks"`
		Count int        `json:"count"`
	}
	json.Unmarshal(result, &resp)
	if resp.Count != 1 {
		t.Errorf("expected 1 task from new instance, got %d", resp.Count)
	}
}

func TestTaskCRUDEffects(t *testing.T) {
	tc := NewTaskCRUD(t.TempDir())
	want := map[string]tools.ToolEffect{
		"task_create": tools.ToolEffectMutate,
		"task_update": tools.ToolEffectMutate,
		"task_get":    tools.ToolEffectRead,
		"task_list":   tools.ToolEffectRead,
	}
	for _, d := range tc.Definitions() {
		if got := d.Effects; got != want[d.Name] {
			t.Errorf("%s: Effects = %v, want %v", d.Name, got, want[d.Name])
		}
	}
}
