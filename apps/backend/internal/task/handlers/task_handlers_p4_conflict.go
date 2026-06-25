package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/AvatarGanymede/pcraft/internal/task/service"
	v1 "github.com/AvatarGanymede/pcraft/pkg/api/v1"
	"go.uber.org/zap"
)

type httpP4ConflictRequest struct {
	ConflictTaskID string `json:"conflict_task_id"`
	SessionID      string `json:"session_id"`
}

type p4SessionStopper interface {
	StopTask(ctx context.Context, taskID, reason string, force bool) error
	StopSession(ctx context.Context, sessionID, reason string, force bool) error
}

func (h *TaskHandlers) httpP4Conflict(c *gin.Context) {
	var body httpP4ConflictRequest
	if err := c.ShouldBindJSON(&body); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	taskID := c.Param("id")
	task, err := h.service.GetTask(c.Request.Context(), taskID)
	if err != nil {
		handleNotFound(c, h.logger, err, "task not found")
		return
	}

	if stopper, ok := h.orchestrator.(p4SessionStopper); ok {
		if err := stopper.StopTask(c.Request.Context(), taskID, "p4 file conflict", true); err != nil {
			h.logger.Warn("p4 conflict: stop task failed",
				zap.String("task_id", taskID),
				zap.Error(err))
			sessionID := strings.TrimSpace(body.SessionID)
			if sessionID != "" {
				if err := stopper.StopSession(c.Request.Context(), sessionID, "p4 file conflict", true); err != nil {
					h.logger.Warn("p4 conflict: stop session failed",
						zap.String("task_id", taskID),
						zap.String("session_id", sessionID),
						zap.Error(err))
				}
			}
		}
	}

	cl := strings.TrimSpace(task.P4Changelist)
	if cl == "" {
		cl = h.p4svc.GetChangelist(taskID)
	}
	if cl != "" {
		if err := h.p4svc.RevertChangelist(c.Request.Context(), cl); err != nil {
			h.logger.Warn("p4 conflict: revert failed", zap.String("task_id", taskID), zap.Error(err))
		}
	}
	h.p4svc.ReleaseByTask(taskID)

	backlog := v1.TaskStateBacklog
	conflictID := strings.TrimSpace(body.ConflictTaskID)
	updated, err := h.service.UpdateTask(c.Request.Context(), taskID, &service.UpdateTaskRequest{
		State:           &backlog,
		BlockedByTaskID: &conflictID,
	})
	if err != nil {
		handleNotFound(c, h.logger, err, "task not updated")
		return
	}
	c.JSON(http.StatusOK, gin.H{"task_id": updated.ID, "state": backlog, "blocked_by_task_id": conflictID})
}
