package models

import (
	"testing"
	"time"

	"github.com/AvatarGanymede/pcraft/internal/agentruntime"
	v1 "github.com/AvatarGanymede/pcraft/pkg/api/v1"
)

func TestLoadSessionRuntimeConfigMapStringExtractsReservedKeys(t *testing.T) {
	cfg, ok := LoadSessionRuntimeConfig(map[string]interface{}{
		SessionMetaKeyRuntimeConfig: map[string]string{
			"model":            "gpt-5.3-codex-spark",
			"mode":             "acceptEdits",
			"reasoning_effort": "low",
		},
	})
	if !ok {
		t.Fatal("expected runtime config")
	}
	if cfg.Model != "gpt-5.3-codex-spark" {
		t.Fatalf("Model = %q", cfg.Model)
	}
	if cfg.Mode != "acceptEdits" {
		t.Fatalf("Mode = %q", cfg.Mode)
	}
	if got := cfg.ConfigOptions["reasoning_effort"]; got != "low" {
		t.Fatalf("reasoning_effort = %q", got)
	}
	if _, ok := cfg.ConfigOptions["model"]; ok {
		t.Fatal("model key should not remain in ConfigOptions")
	}
	if _, ok := cfg.ConfigOptions["mode"]; ok {
		t.Fatal("mode key should not remain in ConfigOptions")
	}
}

func TestTaskStateConstants(t *testing.T) {
	tests := []struct {
		name     string
		state    v1.TaskState
		expected string
	}{
		{"IN_PROGRESS state", v1.TaskStateInProgress, "IN_PROGRESS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.state) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.state))
			}
		})
	}
}

func TestIsTerminalTaskState(t *testing.T) {
	tests := []struct {
		state v1.TaskState
		want  bool
	}{
		{v1.TaskStateInProgress, false},
	}
	for _, tt := range tests {
		if got := IsTerminalTaskState(tt.state); got != tt.want {
			t.Errorf("IsTerminalTaskState(%s) = %v, want %v", tt.state, got, tt.want)
		}
	}
}

func TestWorkflowStructInitialization(t *testing.T) {
	now := time.Now().UTC()
	wf := Workflow{
		ID:          "workflow-123",
		WorkspaceID: "workspace-001",
		Name:        "Test Workflow",
		Description: "A test workflow",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if wf.ID != "workflow-123" {
		t.Errorf("expected ID workflow-123, got %s", wf.ID)
	}
	if wf.WorkspaceID != "workspace-001" {
		t.Errorf("expected WorkspaceID workspace-001, got %s", wf.WorkspaceID)
	}
	if wf.Name != "Test Workflow" {
		t.Errorf("expected Name 'Test Workflow', got %s", wf.Name)
	}
	if wf.Description != "A test workflow" {
		t.Errorf("expected Description 'A test workflow', got %s", wf.Description)
	}
}

func TestTaskToAPI(t *testing.T) {
	now := time.Now().UTC()
	task := &Task{
		ID:             "task-123",
		WorkspaceID:    "workspace-001",
		WorkflowID:     "workflow-456",
		WorkflowStepID: "step-789",
		Title:          "Test Task",
		Description:    "A test task description",
		State:          v1.TaskStateInProgress,
		Priority:       "medium",
		Repositories: []*TaskRepository{
			{
				ID:           "task-repo-1",
				TaskID:       "task-123",
				RepositoryID: "repo-123",
				BaseBranch:   "main",
				Position:     0,
				Metadata:     map[string]interface{}{"role": "primary"},
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		},
		Position:  2,
		Metadata:  map[string]interface{}{"key": "value"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	apiTask := task.ToAPI()

	if apiTask.ID != task.ID {
		t.Errorf("expected ID %s, got %s", task.ID, apiTask.ID)
	}
	if apiTask.WorkspaceID != task.WorkspaceID {
		t.Errorf("expected WorkspaceID %s, got %s", task.WorkspaceID, apiTask.WorkspaceID)
	}
	if apiTask.WorkflowID != task.WorkflowID {
		t.Errorf("expected WorkflowID %s, got %s", task.WorkflowID, apiTask.WorkflowID)
	}
	if apiTask.Title != task.Title {
		t.Errorf("expected Title %s, got %s", task.Title, apiTask.Title)
	}
	if apiTask.Description != task.Description {
		t.Errorf("expected Description %s, got %s", task.Description, apiTask.Description)
	}
	if apiTask.State != task.State {
		t.Errorf("expected State %s, got %s", task.State, apiTask.State)
	}
	if apiTask.Priority != task.Priority {
		t.Errorf("expected Priority %s, got %s", task.Priority, apiTask.Priority)
	}
	if len(apiTask.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(apiTask.Repositories))
	}
	if apiTask.Repositories[0].RepositoryID != "repo-123" {
		t.Errorf("expected RepositoryID repo-123, got %s", apiTask.Repositories[0].RepositoryID)
	}
	if apiTask.Repositories[0].BaseBranch != "main" {
		t.Errorf("expected BaseBranch main, got %s", apiTask.Repositories[0].BaseBranch)
	}
	if apiTask.Metadata["key"] != "value" {
		t.Errorf("expected Metadata key 'value', got %v", apiTask.Metadata["key"])
	}
}

// TestTaskIsFromOfficeField verifies the IsFromOffice field round-trips
// through the model. The actual office-vs-kanban predicate is computed in
// SQL by isFromOfficeProjection (see repository/sqlite/task.go) so the
// scan layer is the only thing that sets the field. A round-trip test is
// the right scope at this layer; the SQL projection itself is covered by
// TestIsFromOfficeProjection_RealWorkspaceWorkflow in
// repository/sqlite/is_from_office_test.go, which exercises all three
// branches (office workflow, project link, neither) against a real DB.
func TestTaskIsFromOfficeField(t *testing.T) {
	tests := []struct {
		name string
		task Task
		want bool
	}{
		{"default zero value", Task{}, false},
		{"explicit false", Task{IsFromOffice: false}, false},
		{"explicit true", Task{IsFromOffice: true}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.task.IsFromOffice; got != tt.want {
				t.Errorf("IsFromOffice = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecutorTypeRuntime(t *testing.T) {
	cases := []struct {
		in   ExecutorType
		want agentruntime.Runtime
	}{
		{ExecutorTypeLocal, agentruntime.RuntimeStandalone},
		{ExecutorTypeMockRemote, agentruntime.RuntimeStandalone},
	}
	for _, tc := range cases {
		t.Run(string(tc.in), func(t *testing.T) {
			if got := tc.in.Runtime(); got != tc.want {
				t.Errorf("ExecutorType(%q).Runtime() = %q, want %q", tc.in, got, tc.want)
			}
		})
	}

		// Unknown ExecutorType falls back to standalone.
		t.Run("unknown_falls_back_to_standalone", func(t *testing.T) {
		got := ExecutorType("not-a-real-type").Runtime()
		if got != agentruntime.RuntimeStandalone {
			t.Errorf("unknown ExecutorType.Runtime() = %q, want %q (host-side fallback)",
				got, agentruntime.RuntimeStandalone)
		}
	})
}
