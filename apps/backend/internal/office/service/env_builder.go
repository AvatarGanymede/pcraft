package service

import (
	"github.com/AvatarGanymede/pcraft/internal/office/models"
)

// buildEnvVars constructs the environment variable map injected into agent
// sessions before launch. The map includes identity, API access, and wake
// context variables described in the agent-context spec.
func (si *SchedulerIntegration) buildEnvVars(
	run *models.Run,
	agent *models.AgentInstance,
	jwt, workspaceID string,
) map[string]string {
	env := map[string]string{
		"PCRAFT_API_URL":      si.svc.apiBaseURL,
		"PCRAFT_API_KEY":      jwt,
		"PCRAFT_RUN_TOKEN":    jwt,
		"PCRAFT_AGENT_ID":     agent.ID,
		"PCRAFT_AGENT_NAME":   agent.Name,
		"PCRAFT_WORKSPACE_ID": workspaceID,
		"PCRAFT_RUN_ID":       run.ID,
		"PCRAFT_WAKE_REASON":  run.Reason,
	}
	if taskID := extractField(run.Payload, "task_id"); taskID != "" {
		env["PCRAFT_TASK_ID"] = taskID
	}
	if commentID := extractField(run.Payload, "comment_id"); commentID != "" {
		env["PCRAFT_WAKE_COMMENT_ID"] = commentID
	}
	// PCRAFT_CLI - path to agentctl binary for CLI operations.
	// Default to host binary path; overridden per executor type by injectKandevCLI.
	if si.svc.agentctlBinaryPath != "" {
		env["PCRAFT_CLI"] = si.svc.agentctlBinaryPath
	}
	return env
}

// injectKandevCLI overrides PCRAFT_CLI for executor types where the
// host binary path does not apply. All executors are currently local,
// so the host binary path is used directly — no override needed.
func (si *SchedulerIntegration) injectKandevCLI(env map[string]string, executorType string) {
	// No-op: only local executor is supported; host binary path is always correct.
}

// extractField parses a single key from a JSON payload string.
func extractField(payloadJSON, key string) string {
	return ParseRunPayload(payloadJSON)[key]
}
