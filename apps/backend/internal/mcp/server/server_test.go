package mcp

import (
	"testing"

	"github.com/AvatarGanymede/pcraft/internal/common/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger(t *testing.T) *logger.Logger {
	t.Helper()
	log, err := logger.NewLogger(logger.LoggingConfig{Level: "error", Format: "console"})
	require.NoError(t, err)
	return log
}

// getRegisteredToolNames returns the names of all tools registered on the MCP server.
func getRegisteredToolNames(s *Server) []string {
	toolsMap := s.mcpServer.ListTools()
	names := make([]string, 0, len(toolsMap))
	for name := range toolsMap {
		names = append(names, name)
	}
	return names
}

func TestServerModeTask_RegistersCorrectTools(t *testing.T) {
	log := newTestLogger(t)
	backend := NewChannelBackendClient(log)
	defer backend.Close()

	s := New(backend, "test-session", "test-task", 10005, log, "", false, ModeTask)
	require.NotNil(t, s)

	tools := getRegisteredToolNames(s)

	// Task mode should have kanban tools
	assert.Contains(t, tools, "list_workspaces_pcraft")
	assert.Contains(t, tools, "list_workflows_pcraft")
	assert.Contains(t, tools, "list_workflow_steps_pcraft")
	assert.Contains(t, tools, "list_tasks_pcraft")
	assert.Contains(t, tools, "create_task_pcraft")
	assert.Contains(t, tools, "update_task_pcraft")
	assert.Contains(t, tools, "move_task_pcraft")
	assert.Contains(t, tools, "message_task_pcraft")
	assert.Contains(t, tools, "get_task_conversation_pcraft")

	// Task mode should have plan tools
	assert.Contains(t, tools, "create_task_plan_pcraft")
	assert.Contains(t, tools, "get_task_plan_pcraft")
	assert.Contains(t, tools, "update_task_plan_pcraft")
	assert.Contains(t, tools, "delete_task_plan_pcraft")

	// Task mode should have interaction tools
	assert.Contains(t, tools, "ask_user_question_pcraft")

	// Task mode should have profile listing tools (needed for create_task)
	assert.Contains(t, tools, "list_agents_pcraft")
	assert.Contains(t, tools, "list_executor_profiles_pcraft")

	// Task mode keeps list_related_tasks_pcraft (sibling discovery) but
	// drops the task-document tools — those are office-only.
	assert.Contains(t, tools, "list_related_tasks_pcraft")
	assert.NotContains(t, tools, "list_task_documents_pcraft")
	assert.NotContains(t, tools, "get_task_document_pcraft")
	assert.NotContains(t, tools, "write_task_document_pcraft")

	// Task mode exposes delete + archive so agents can clean up the tasks
	// they fan out. Restore/unarchive is intentionally NOT exposed via MCP —
	// it stays a user action in the UI.
	assert.Contains(t, tools, "delete_task_pcraft")
	assert.Contains(t, tools, "archive_task_pcraft")
	assert.NotContains(t, tools, "restore_task_pcraft")

	// Task mode should NOT have config/mutation tools
	assert.NotContains(t, tools, "create_workflow_pcraft")
	assert.NotContains(t, tools, "update_workflow_pcraft")
	assert.NotContains(t, tools, "delete_workflow_pcraft")
	assert.NotContains(t, tools, "create_workflow_step_pcraft")
	assert.NotContains(t, tools, "update_workflow_step_pcraft")
	assert.NotContains(t, tools, "update_agent_pcraft")
	assert.NotContains(t, tools, "create_agent_profile_pcraft")
	assert.NotContains(t, tools, "delete_agent_profile_pcraft")
	assert.NotContains(t, tools, "list_agent_profiles_pcraft")
	assert.NotContains(t, tools, "update_agent_profile_pcraft")
	assert.NotContains(t, tools, "get_mcp_config_pcraft")
	assert.NotContains(t, tools, "update_mcp_config_pcraft")
	assert.NotContains(t, tools, "list_executors_pcraft")
	assert.NotContains(t, tools, "create_executor_profile_pcraft")
	assert.NotContains(t, tools, "update_executor_profile_pcraft")
	assert.NotContains(t, tools, "delete_executor_profile_pcraft")
	assert.NotContains(t, tools, "update_task_state_pcraft")
	assert.NotContains(t, tools, "delete_workflow_step_pcraft")
	assert.NotContains(t, tools, "reorder_workflow_steps_pcraft")
}

func TestServerModeConfig_RegistersCorrectTools(t *testing.T) {
	log := newTestLogger(t)
	backend := NewChannelBackendClient(log)
	defer backend.Close()

	s := New(backend, "test-session", "test-task", 10005, log, "", false, ModeConfig)
	require.NotNil(t, s)

	tools := getRegisteredToolNames(s)

	// Config mode should have workflow config tools
	assert.Contains(t, tools, "list_workspaces_pcraft")
	assert.Contains(t, tools, "list_workflows_pcraft")
	assert.Contains(t, tools, "list_repositories_pcraft")
	assert.Contains(t, tools, "create_workflow_pcraft")
	assert.Contains(t, tools, "update_workflow_pcraft")
	assert.Contains(t, tools, "delete_workflow_pcraft")
	assert.Contains(t, tools, "import_workflow_pcraft")
	assert.Contains(t, tools, "list_workflow_steps_pcraft")
	assert.Contains(t, tools, "create_workflow_step_pcraft")
	assert.Contains(t, tools, "update_workflow_step_pcraft")
	assert.Contains(t, tools, "delete_workflow_step_pcraft")
	assert.Contains(t, tools, "reorder_workflow_steps_pcraft")

	// Config mode should have agent tools
	assert.Contains(t, tools, "list_agents_pcraft")
	assert.Contains(t, tools, "update_agent_pcraft")
	assert.Contains(t, tools, "create_agent_profile_pcraft")
	assert.Contains(t, tools, "delete_agent_profile_pcraft")

	// Config mode should have MCP config tools
	assert.Contains(t, tools, "list_agent_profiles_pcraft")
	assert.Contains(t, tools, "update_agent_profile_pcraft")
	assert.Contains(t, tools, "get_mcp_config_pcraft")
	assert.Contains(t, tools, "update_mcp_config_pcraft")

	// Config mode should have executor profile tools
	assert.Contains(t, tools, "list_executors_pcraft")
	assert.Contains(t, tools, "list_executor_profiles_pcraft")
	assert.Contains(t, tools, "create_executor_profile_pcraft")
	assert.Contains(t, tools, "update_executor_profile_pcraft")
	assert.Contains(t, tools, "delete_executor_profile_pcraft")

	// Config mode should have task tools
	assert.Contains(t, tools, "list_tasks_pcraft")
	assert.Contains(t, tools, "move_task_pcraft")
	assert.Contains(t, tools, "delete_task_pcraft")
	assert.Contains(t, tools, "archive_task_pcraft")
	assert.Contains(t, tools, "update_task_state_pcraft")
	assert.Contains(t, tools, "get_task_conversation_pcraft")

	// Config mode should have interaction tools
	assert.Contains(t, tools, "ask_user_question_pcraft")

	// Config mode should NOT have plan tools
	assert.NotContains(t, tools, "create_task_plan_pcraft")
	assert.NotContains(t, tools, "get_task_plan_pcraft")
	assert.NotContains(t, tools, "update_task_plan_pcraft")
	assert.NotContains(t, tools, "delete_task_plan_pcraft")

	// Config mode should NOT have task-mode kanban create/update tools
	assert.NotContains(t, tools, "create_task_pcraft")
	assert.NotContains(t, tools, "update_task_pcraft")
}

func TestServerModeDefault_DefaultsToTask(t *testing.T) {
	log := newTestLogger(t)
	backend := NewChannelBackendClient(log)
	defer backend.Close()

	s := New(backend, "test-session", "test-task", 10005, log, "", false, "")
	require.NotNil(t, s)
	assert.Equal(t, ModeTask, s.mode)

	tools := getRegisteredToolNames(s)
	assert.Contains(t, tools, "create_task_pcraft")
	assert.Contains(t, tools, "create_task_plan_pcraft")
	assert.NotContains(t, tools, "create_workflow_step_pcraft")
}

func TestServerModeConfig_DisableAskQuestion(t *testing.T) {
	log := newTestLogger(t)
	backend := NewChannelBackendClient(log)
	defer backend.Close()

	s := New(backend, "test-session", "test-task", 10005, log, "", true, ModeConfig)
	require.NotNil(t, s)

	tools := getRegisteredToolNames(s)
	assert.NotContains(t, tools, "ask_user_question_pcraft")
	assert.Contains(t, tools, "list_agents_pcraft")
	assert.Contains(t, tools, "create_workflow_step_pcraft")
}

func TestServerModeTask_DisableAskQuestion(t *testing.T) {
	log := newTestLogger(t)
	backend := NewChannelBackendClient(log)
	defer backend.Close()

	s := New(backend, "test-session", "test-task", 10005, log, "", true, ModeTask)
	require.NotNil(t, s)

	tools := getRegisteredToolNames(s)
	assert.NotContains(t, tools, "ask_user_question_pcraft")
	assert.Contains(t, tools, "create_task_pcraft")
	assert.Contains(t, tools, "create_task_plan_pcraft")
}

func TestServerModeTask_ToolCount(t *testing.T) {
	log := newTestLogger(t)
	backend := NewChannelBackendClient(log)
	defer backend.Close()

	s := New(backend, "test-session", "test-task", 10005, log, "", false, ModeTask)
	tools := getRegisteredToolNames(s)
	// 13 kanban (incl. delete + archive task) + 1 add_branch_to_task +
	// 1 update_repository_base_branch + 1 step_complete (ADR 0015) +
	// 1 interaction + 4 plan + 1 related-tasks = 22. Task-document tools
	// (list/get/write) are office-only.
	assert.Contains(t, tools, "step_complete_pcraft", "ADR 0015 explicit-completion signal must be registered in task mode")
	assert.Equal(t, 22, len(tools))
}

func TestServerModeConfig_ToolCount(t *testing.T) {
	log := newTestLogger(t)
	backend := NewChannelBackendClient(log)
	defer backend.Close()

	s := New(backend, "test-session", "test-task", 10005, log, "", false, ModeConfig)
	tools := getRegisteredToolNames(s)
	// 12 workflow (incl. list_repositories + import_workflow) + 4 agent + 4 mcp + 5 executor + 6 task + 1 interaction = 32
	assert.NotContains(t, tools, "step_complete_pcraft", "step_complete_pcraft requires a live task session; must NOT register in config mode")
	assert.Equal(t, 32, len(tools))
}

func TestServerModeConfig_ToolDescriptions(t *testing.T) {
	log := newTestLogger(t)
	backend := NewChannelBackendClient(log)
	defer backend.Close()

	s := New(backend, "test-session", "test-task", 10005, log, "", false, ModeConfig)

	toolsMap := s.mcpServer.ListTools()

	assert.Contains(t, toolsMap["create_workflow_step_pcraft"].Tool.Description, "Create a new workflow step")
	assert.Contains(t, toolsMap["list_agents_pcraft"].Tool.Description, "List all configured agents")
	assert.Contains(t, toolsMap["get_mcp_config_pcraft"].Tool.Description, "Get MCP server configuration")
}

func TestServerModeOffice_RegistersCorrectTools(t *testing.T) {
	log := newTestLogger(t)
	backend := NewChannelBackendClient(log)
	defer backend.Close()

	s := New(backend, "test-session", "test-task", 10005, log, "", false, ModeOffice)
	require.NotNil(t, s)

	tools := getRegisteredToolNames(s)

	// Office mode should have plan tools
	assert.Contains(t, tools, "create_task_plan_pcraft")
	assert.Contains(t, tools, "get_task_plan_pcraft")
	assert.Contains(t, tools, "update_task_plan_pcraft")
	assert.Contains(t, tools, "delete_task_plan_pcraft")

	// Office mode should have interaction tools
	assert.Contains(t, tools, "ask_user_question_pcraft")

	// delegate_task_pcraft was removed from ModeOffice when the
	// agentctl CLI started covering task creation/delegation via
	// `agentctl kandev task create --parent $KANDEV_TASK_ID …`.
	assert.NotContains(t, tools, "delegate_task_pcraft")

	// Office mode exposes the cross-task handoff tools.
	assert.Contains(t, tools, "list_related_tasks_pcraft")
	assert.Contains(t, tools, "list_task_documents_pcraft")
	assert.Contains(t, tools, "get_task_document_pcraft")
	assert.Contains(t, tools, "write_task_document_pcraft")

	// Office mode should NOT have kanban tools
	assert.NotContains(t, tools, "create_task_pcraft")
	assert.NotContains(t, tools, "list_tasks_pcraft")
	assert.NotContains(t, tools, "update_task_pcraft")
	assert.NotContains(t, tools, "list_workspaces_pcraft")
	assert.NotContains(t, tools, "list_workflows_pcraft")
	assert.NotContains(t, tools, "list_workflow_steps_pcraft")
	assert.NotContains(t, tools, "list_agents_pcraft")
	assert.NotContains(t, tools, "list_executor_profiles_pcraft")

	// Office mode should NOT have config tools
	assert.NotContains(t, tools, "create_workflow_pcraft")
	assert.NotContains(t, tools, "update_workflow_pcraft")
	assert.NotContains(t, tools, "update_agent_pcraft")
}

func TestServerModeOffice_ToolCount(t *testing.T) {
	log := newTestLogger(t)
	backend := NewChannelBackendClient(log)
	defer backend.Close()

	s := New(backend, "test-session", "test-task", 10005, log, "", false, ModeOffice)
	tools := getRegisteredToolNames(s)
	// 4 plan + 1 interaction + 1 related-tasks + 3 task-documents = 9
	// (delegate_task_pcraft retired in favour of `agentctl kandev task create …`).
	assert.NotContains(t, tools, "step_complete_pcraft", "step_complete_pcraft is kanban-task-only; office mode advances tasks via its own approval surface")
	assert.Equal(t, 9, len(tools))
}

func TestServerModeOffice_DisableAskQuestion(t *testing.T) {
	log := newTestLogger(t)
	backend := NewChannelBackendClient(log)
	defer backend.Close()

	s := New(backend, "test-session", "test-task", 10005, log, "", true, ModeOffice)
	require.NotNil(t, s)

	tools := getRegisteredToolNames(s)
	assert.NotContains(t, tools, "ask_user_question_pcraft")
	assert.Contains(t, tools, "create_task_plan_pcraft")
	// delegate_task_pcraft was retired from ModeOffice (now lives in
	// the agentctl CLI as `agentctl kandev task create --parent …`).
	assert.NotContains(t, tools, "delegate_task_pcraft")
	// 4 plan + 1 related-tasks + 3 task-documents = 8 (no ask_user_question, no delegate)
	assert.Equal(t, 8, len(tools))
}

func TestServerModeConstants(t *testing.T) {
	assert.Equal(t, "task", ModeTask)
	assert.Equal(t, "config", ModeConfig)
	assert.Equal(t, "external", ModeExternal)
	assert.Equal(t, "office", ModeOffice)
}

func TestServerModeExternal_RegistersCorrectTools(t *testing.T) {
	log := newTestLogger(t)
	backend := NewChannelBackendClient(log)
	defer backend.Close()

	s := New(backend, "", "", 0, log, "", true, ModeExternal)
	require.NotNil(t, s)

	tools := getRegisteredToolNames(s)

	// External mode includes all config tools
	assert.Contains(t, tools, "list_workspaces_pcraft")
	assert.Contains(t, tools, "list_repositories_pcraft")
	assert.Contains(t, tools, "create_workflow_pcraft")
	assert.Contains(t, tools, "list_agents_pcraft")
	assert.Contains(t, tools, "get_mcp_config_pcraft")
	assert.Contains(t, tools, "list_executors_pcraft")
	assert.Contains(t, tools, "move_task_pcraft")

	// External mode includes create_task_pcraft so external agents can spawn tasks
	assert.Contains(t, tools, "create_task_pcraft")

	// External mode does NOT include session-scoped tools
	assert.NotContains(t, tools, "ask_user_question_pcraft")
	assert.NotContains(t, tools, "create_task_plan_pcraft")
	assert.NotContains(t, tools, "get_task_plan_pcraft")
	assert.NotContains(t, tools, "update_task_plan_pcraft")
	assert.NotContains(t, tools, "delete_task_plan_pcraft")

	// External mode does NOT include kanban update_task_pcraft (config has its own update_task_state)
	assert.NotContains(t, tools, "update_task_pcraft")

	// External mode does NOT include message_task_pcraft (no live session context)
	assert.NotContains(t, tools, "message_task_pcraft")
}

func TestServerModeExternal_ToolCount(t *testing.T) {
	log := newTestLogger(t)
	backend := NewChannelBackendClient(log)
	defer backend.Close()

	s := New(backend, "", "", 0, log, "", true, ModeExternal)
	tools := getRegisteredToolNames(s)
	// 12 workflow (incl. list_repositories + import_workflow) + 4 agent + 4 mcp + 5 executor + 6 task + 1 create_task = 32.
	// add_branch_to_task_pcraft is task-mode only — external coding agents have no live session to attach a worktree to.
	assert.Equal(t, 32, len(tools))
	assert.NotContains(t, tools, "add_branch_to_task_pcraft")
}

func TestNewExternal_Constructs(t *testing.T) {
	log := newTestLogger(t)
	backend := NewChannelBackendClient(log)
	defer backend.Close()

	s := NewExternal(backend, log, "")
	require.NotNil(t, s)
	assert.Equal(t, ModeExternal, s.mode)
	assert.True(t, s.disableAskQuestion)
	assert.Empty(t, s.sessionID)
	assert.Empty(t, s.taskID)
	assert.NotNil(t, s.sseServer)
	assert.NotNil(t, s.httpServer)
	assert.Equal(t, "/mcp/message?sessionId=session-1", s.sseServer.GetMessageEndpointForClient(nil, "session-1"))
}
