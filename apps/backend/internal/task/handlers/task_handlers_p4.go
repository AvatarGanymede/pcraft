package handlers

import (
	"context"
	"strings"

	"github.com/AvatarGanymede/pcraft/internal/orchestrator"
	"github.com/AvatarGanymede/pcraft/internal/p4"
	"github.com/AvatarGanymede/pcraft/internal/task/models"
	"github.com/AvatarGanymede/pcraft/internal/task/service"
	v1 "github.com/AvatarGanymede/pcraft/pkg/api/v1"
	"go.uber.org/zap"
)

func (h *TaskHandlers) finalizeP4Task(ctx context.Context, task *models.Task, p4WorkspaceID string) {
	p4WorkspaceID = strings.TrimSpace(p4WorkspaceID)
	if p4WorkspaceID == "" {
		return
	}
	cl := h.p4svc.EnsureChangelist(ctx, task.ID)
	h.p4svc.BindChangelist(task.ID, cl)
	p4WS := p4WorkspaceID
	clStr := cl
	if _, err := h.service.UpdateTask(ctx, task.ID, &service.UpdateTaskRequest{
		P4WorkspaceID: &p4WS,
		P4Changelist:  &clStr,
	}); err != nil {
		h.logger.Warn("failed to persist p4 fields on task",
			zap.String("task_id", task.ID), zap.Error(err))
	}
}

func (h *TaskHandlers) wakeBacklogAfterClose(closedTask *models.Task) {
	if h.orchestrator == nil || closedTask == nil {
		return
	}
	go func() {
		ctx := context.Background()
		tasks, _, err := h.service.ListTasksByWorkspace(ctx, closedTask.WorkspaceID, "", "", "", 1, 200, false, false, false, true)
		if err != nil {
			h.logger.Warn("wakeup: list backlog tasks failed", zap.Error(err))
			return
		}
		candidates := make([]p4.BacklogCandidate, 0)
		for _, task := range tasks {
			if task == nil || task.State != v1.TaskStateBacklog || task.ID == closedTask.ID {
				continue
			}
			if task.BlockedByTaskID != "" && task.BlockedByTaskID != closedTask.ID {
				continue
			}
			candidates = append(candidates, p4.BacklogCandidate{
				ID:          task.ID,
				WorkspaceID: task.WorkspaceID,
				Description: task.Description,
				Metadata:    task.Metadata,
			})
		}
		for _, candidate := range p4.FilterWakeupCandidates(candidates, closedTask.WorkspaceID) {
			prompt := p4.ResumePrompt(candidate.Description, candidate.Metadata)
			if strings.TrimSpace(prompt) == "" {
				continue
			}
			inProgress := v1.TaskStateInProgress
			if _, err := h.service.UpdateTask(ctx, candidate.ID, &service.UpdateTaskRequest{State: &inProgress}); err != nil {
				h.logger.Warn("wakeup: failed to mark task in progress",
					zap.String("task_id", candidate.ID), zap.Error(err))
				continue
			}
			if _, err := h.orchestrator.LaunchSession(ctx, &orchestrator.LaunchSessionRequest{
				TaskID:         candidate.ID,
				Intent:         orchestrator.IntentStartCreated,
				Prompt:         prompt,
				SkipMessageRecord: false,
			}); err != nil {
				h.logger.Warn("wakeup: launch session failed",
					zap.String("task_id", candidate.ID), zap.Error(err))
				backlog := v1.TaskStateBacklog
				_, _ = h.service.UpdateTask(ctx, candidate.ID, &service.UpdateTaskRequest{State: &backlog})
			}
			return
		}
	}()
}
