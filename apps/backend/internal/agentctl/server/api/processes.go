package api

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/AvatarGanymede/pcraft/internal/agentctl/server/process"
	"go.uber.org/zap"
)

// safeBranchRefPattern mirrors the one in workspace_git_status.go so the
// workspace base-branches handler can validate branch refs at the HTTP boundary.
var safeBranchRefPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*$`)

// queryParamTrue is the string value used to indicate a true boolean in query parameters.
const queryParamTrue = "true"

type startProcessResponse struct {
	Process *process.ProcessInfo `json:"process,omitempty"`
	Error   string               `json:"error,omitempty"`
}

type stopProcessResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (s *Server) handleStartProcess(c *gin.Context) {
	var req process.StartProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, startProcessResponse{Error: "invalid request body"})
		return
	}

	proc, err := s.procMgr.StartProcess(context.Background(), req)
	if err != nil {
		s.logger.Error("failed to start process", zap.Error(err))
		c.JSON(http.StatusBadRequest, startProcessResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, startProcessResponse{Process: proc})
}

func (s *Server) handleStopProcess(c *gin.Context) {
	var req process.StopProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, stopProcessResponse{Success: false, Error: "invalid request body"})
		return
	}
	if req.ProcessID == "" {
		c.JSON(http.StatusBadRequest, stopProcessResponse{Success: false, Error: "process_id is required"})
		return
	}
	if err := s.procMgr.StopProcess(c.Request.Context(), req); err != nil {
		if strings.Contains(err.Error(), "process not found") {
			c.JSON(http.StatusOK, stopProcessResponse{Success: true})
			return
		}
		s.logger.Error("failed to stop process", zap.Error(err))
		c.JSON(http.StatusInternalServerError, stopProcessResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, stopProcessResponse{Success: true})
}

func (s *Server) handleListProcesses(c *gin.Context) {
	sessionID := c.Query("session_id")
	processes := s.procMgr.ListProcesses(sessionID)
	c.JSON(http.StatusOK, processes)
}

func (s *Server) handleGetProcess(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "process id is required"})
		return
	}
	includeOutput := c.Query("include_output") == queryParamTrue
	proc, ok := s.procMgr.GetProcess(id, includeOutput)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "process not found"})
		return
	}
	c.JSON(http.StatusOK, proc)
}
