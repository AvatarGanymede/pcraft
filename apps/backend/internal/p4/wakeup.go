package p4

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/AvatarGanymede/pcraft/pkg/api/v1"
)

// BacklogCandidate is the minimal task shape needed for auto-resume after close.
type BacklogCandidate struct {
	ID          string
	WorkspaceID string
	Description string
	Metadata    map[string]interface{}
}

// ResumePrompt builds the dev-gui-plugin resume prompt for a backlog task.
func ResumePrompt(description string, metadata map[string]interface{}) string {
	base := strings.TrimSpace(description)
	panel := stringMeta(metadata, "panel_id")
	req := stringMeta(metadata, "requirement")
	prefab := stringMeta(metadata, "prefab")
	if panel != "" && req != "" {
		prompt := fmt.Sprintf("/dev-gui-plugin:run %s, %s, resume=true", panel, req)
		if prefab != "" {
			prompt += ", prefab=" + prefab
		}
		return prompt
	}
	if base != "" {
		if strings.Contains(base, "resume=true") {
			return base
		}
		return base + ", resume=true"
	}
	return "/dev-gui-plugin:run resume=true"
}

func stringMeta(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	raw, ok := metadata[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(raw)
}

// FilterWakeupCandidates returns backlog tasks in the same workspace, oldest first.
func FilterWakeupCandidates(all []BacklogCandidate, workspaceID string) []BacklogCandidate {
	out := make([]BacklogCandidate, 0)
	for _, task := range all {
		if task.WorkspaceID == workspaceID {
			out = append(out, task)
		}
	}
	return out
}

// NormalizeState maps legacy aliases to canonical four-state values.
// Legacy names are type aliases of the canonical constants in pkg/api/v1.
func NormalizeState(state v1.TaskState) v1.TaskState {
	return state
}

// OpenedFilesForTask returns pending files for a task changelist.
func (s *Service) OpenedFilesForTask(ctx context.Context, taskID string) ([]string, error) {
	cl := s.GetChangelist(taskID)
	if cl == "" {
		return nil, nil
	}
	return s.client.OpenedFiles(ctx, cl)
}
