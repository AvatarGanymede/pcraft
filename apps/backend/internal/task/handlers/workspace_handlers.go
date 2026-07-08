package handlers

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/AvatarGanymede/pcraft/internal/common/logger"
	"github.com/AvatarGanymede/pcraft/internal/task/dto"
	"github.com/AvatarGanymede/pcraft/internal/task/models"
	"github.com/AvatarGanymede/pcraft/internal/task/service"
	ws "github.com/AvatarGanymede/pcraft/pkg/websocket"
	"go.uber.org/zap"
)

// taskFormDefPattern restricts a field's placeholder key to English
// letters/digits/underscore, starting with a letter — matching the
// {{prompt_<def>}} token the frontend builds and the backend interpolates.
var taskFormDefPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// validateTaskFormConfig enforces the per-workspace new-task form invariants:
// every field needs a label and a unique, English-only def. An empty config
// (no fields) is valid and means "use the default single-prompt form".
func validateTaskFormConfig(cfg models.TaskFormConfig) error {
	seen := make(map[string]struct{}, len(cfg.Fields))
	for i, f := range cfg.Fields {
		def := strings.TrimSpace(f.Def)
		if def == "" {
			return fmt.Errorf("task form field %d: def is required", i+1)
		}
		if !taskFormDefPattern.MatchString(def) {
			return fmt.Errorf("task form field %q: def must be English letters, digits or underscore and start with a letter", def)
		}
		if _, dup := seen[def]; dup {
			return fmt.Errorf("task form field def %q is duplicated", def)
		}
		seen[def] = struct{}{}
		if strings.TrimSpace(f.Label) == "" {
			return fmt.Errorf("task form field %q: label is required", def)
		}
	}
	return nil
}

type WorkspaceHandlers struct {
	service *service.Service
	logger  *logger.Logger
}

func NewWorkspaceHandlers(svc *service.Service, log *logger.Logger) *WorkspaceHandlers {
	return &WorkspaceHandlers{
		service: svc,
		logger:  log.WithFields(zap.String("component", "task-workspace-handlers")),
	}
}

func RegisterWorkspaceRoutes(router *gin.Engine, dispatcher *ws.Dispatcher, svc *service.Service, log *logger.Logger) {
	handlers := NewWorkspaceHandlers(svc, log)
	handlers.registerHTTP(router)
	handlers.registerWS(dispatcher)
}

func (h *WorkspaceHandlers) registerHTTP(router *gin.Engine) {
	api := router.Group("/api/v1")
	api.GET("/workspaces", h.httpListWorkspaces)
	api.POST("/workspaces", h.httpCreateWorkspace)
	api.GET("/workspaces/:id", h.httpGetWorkspace)
	api.PATCH("/workspaces/:id", h.httpUpdateWorkspace)
	api.DELETE("/workspaces/:id", h.httpDeleteWorkspace)
}

func (h *WorkspaceHandlers) registerWS(dispatcher *ws.Dispatcher) {
	dispatcher.RegisterFunc(ws.ActionWorkspaceList, h.wsListWorkspaces)
	dispatcher.RegisterFunc(ws.ActionWorkspaceCreate, h.wsCreateWorkspace)
	dispatcher.RegisterFunc(ws.ActionWorkspaceGet, h.wsGetWorkspace)
	dispatcher.RegisterFunc(ws.ActionWorkspaceUpdate, h.wsUpdateWorkspace)
	dispatcher.RegisterFunc(ws.ActionWorkspaceDelete, h.wsDeleteWorkspace)
}

func (h *WorkspaceHandlers) listWorkspaces(ctx context.Context) (dto.ListWorkspacesResponse, error) {
	workspaces, err := h.service.ListWorkspaces(ctx)
	if err != nil {
		return dto.ListWorkspacesResponse{}, err
	}
	dtos := make([]dto.WorkspaceDTO, 0, len(workspaces))
	for _, w := range workspaces {
		dtos = append(dtos, dto.FromWorkspace(w))
	}
	return dto.ListWorkspacesResponse{Workspaces: dtos, Total: len(dtos)}, nil
}

// HTTP handlers

func (h *WorkspaceHandlers) httpListWorkspaces(c *gin.Context) {
	resp, err := h.listWorkspaces(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to list workspaces", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list workspaces"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

type httpCreateWorkspaceRequest struct {
	Name                        string  `json:"name"`
	Description                 string  `json:"description,omitempty"`
	OwnerID                     string  `json:"owner_id,omitempty"`
	DefaultEnvironmentID        *string `json:"default_environment_id,omitempty"`
	DefaultAgentProfileID       *string `json:"default_agent_profile_id,omitempty"`
	DefaultConfigAgentProfileID *string `json:"default_config_agent_profile_id,omitempty"`
}

func (h *WorkspaceHandlers) httpCreateWorkspace(c *gin.Context) {
	var body httpCreateWorkspaceRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if body.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	workspace, err := h.service.CreateWorkspace(c.Request.Context(), &service.CreateWorkspaceRequest{
		Name:                        body.Name,
		Description:                 body.Description,
		OwnerID:                     body.OwnerID,
		DefaultEnvironmentID:        body.DefaultEnvironmentID,
		DefaultAgentProfileID:       body.DefaultAgentProfileID,
		DefaultConfigAgentProfileID: body.DefaultConfigAgentProfileID,
	})
	if err != nil {
		handleNotFound(c, h.logger, err, "workspace not created")
		return
	}
	c.JSON(http.StatusOK, dto.FromWorkspace(workspace))
}

func (h *WorkspaceHandlers) httpGetWorkspace(c *gin.Context) {
	workspace, err := h.service.GetWorkspace(c.Request.Context(), c.Param("id"))
	if err != nil {
		handleNotFound(c, h.logger, err, "workspace not found")
		return
	}
	c.JSON(http.StatusOK, dto.FromWorkspace(workspace))
}

type httpUpdateWorkspaceRequest struct {
	Name                        *string                `json:"name,omitempty"`
	Description                 *string                `json:"description,omitempty"`
	DefaultEnvironmentID        *string                `json:"default_environment_id,omitempty"`
	DefaultAgentProfileID       *string                `json:"default_agent_profile_id,omitempty"`
	DefaultConfigAgentProfileID *string                `json:"default_config_agent_profile_id,omitempty"`
	TaskFormConfig              *models.TaskFormConfig `json:"task_form_config,omitempty"`
	P4Client                    *string                `json:"p4_client,omitempty"`
}

func (h *WorkspaceHandlers) httpUpdateWorkspace(c *gin.Context) {
	var body httpUpdateWorkspaceRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if body.TaskFormConfig != nil {
		if err := validateTaskFormConfig(*body.TaskFormConfig); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	workspace, err := h.service.UpdateWorkspace(c.Request.Context(), c.Param("id"), &service.UpdateWorkspaceRequest{
		Name:                        body.Name,
		Description:                 body.Description,
		DefaultEnvironmentID:        body.DefaultEnvironmentID,
		DefaultAgentProfileID:       body.DefaultAgentProfileID,
		DefaultConfigAgentProfileID: body.DefaultConfigAgentProfileID,
		TaskFormConfig:              body.TaskFormConfig,
		P4Client:                    body.P4Client,
	})
	if err != nil {
		handleNotFound(c, h.logger, err, "workspace not updated")
		return
	}
	c.JSON(http.StatusOK, dto.FromWorkspace(workspace))
}

func (h *WorkspaceHandlers) httpDeleteWorkspace(c *gin.Context) {
	if err := h.service.DeleteWorkspace(c.Request.Context(), c.Param("id")); err != nil {
		handleNotFound(c, h.logger, err, "workspace not deleted")
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse{Success: true})
}

// WS handlers

func (h *WorkspaceHandlers) wsListWorkspaces(ctx context.Context, msg *ws.Message) (*ws.Message, error) {
	resp, err := h.listWorkspaces(ctx)
	if err != nil {
		h.logger.Error("failed to list workspaces", zap.Error(err))
		return ws.NewError(msg.ID, msg.Action, ws.ErrorCodeInternalError, "Failed to list workspaces", nil)
	}
	return ws.NewResponse(msg.ID, msg.Action, resp)
}

type wsCreateWorkspaceRequest struct {
	Name                        string  `json:"name"`
	Description                 string  `json:"description,omitempty"`
	OwnerID                     string  `json:"owner_id,omitempty"`
	DefaultEnvironmentID        *string `json:"default_environment_id,omitempty"`
	DefaultAgentProfileID       *string `json:"default_agent_profile_id,omitempty"`
	DefaultConfigAgentProfileID *string `json:"default_config_agent_profile_id,omitempty"`
}

func (h *WorkspaceHandlers) wsCreateWorkspace(ctx context.Context, msg *ws.Message) (*ws.Message, error) {
	var req wsCreateWorkspaceRequest
	if err := msg.ParsePayload(&req); err != nil {
		return ws.NewError(msg.ID, msg.Action, ws.ErrorCodeBadRequest, "Invalid payload: "+err.Error(), nil)
	}
	if req.Name == "" {
		return ws.NewError(msg.ID, msg.Action, ws.ErrorCodeValidation, "name is required", nil)
	}
	workspace, err := h.service.CreateWorkspace(ctx, &service.CreateWorkspaceRequest{
		Name:                        req.Name,
		Description:                 req.Description,
		OwnerID:                     req.OwnerID,
		DefaultEnvironmentID:        req.DefaultEnvironmentID,
		DefaultAgentProfileID:       req.DefaultAgentProfileID,
		DefaultConfigAgentProfileID: req.DefaultConfigAgentProfileID,
	})
	if err != nil {
		h.logger.Error("failed to create workspace", zap.Error(err))
		return ws.NewError(msg.ID, msg.Action, ws.ErrorCodeInternalError, "Failed to create workspace", nil)
	}
	return ws.NewResponse(msg.ID, msg.Action, dto.FromWorkspace(workspace))
}

type wsGetWorkspaceRequest struct {
	ID string `json:"id"`
}

func (h *WorkspaceHandlers) wsGetWorkspace(ctx context.Context, msg *ws.Message) (*ws.Message, error) {
	var req wsGetWorkspaceRequest
	if err := msg.ParsePayload(&req); err != nil {
		return ws.NewError(msg.ID, msg.Action, ws.ErrorCodeBadRequest, "Invalid payload: "+err.Error(), nil)
	}
	if req.ID == "" {
		return ws.NewError(msg.ID, msg.Action, ws.ErrorCodeValidation, "id is required", nil)
	}

	workspace, err := h.service.GetWorkspace(ctx, req.ID)
	if err != nil {
		return ws.NewError(msg.ID, msg.Action, ws.ErrorCodeNotFound, "Workspace not found", nil)
	}
	return ws.NewResponse(msg.ID, msg.Action, dto.FromWorkspace(workspace))
}

type wsUpdateWorkspaceRequest struct {
	ID                          string                 `json:"id"`
	Name                        *string                `json:"name,omitempty"`
	Description                 *string                `json:"description,omitempty"`
	DefaultEnvironmentID        *string                `json:"default_environment_id,omitempty"`
	DefaultAgentProfileID       *string                `json:"default_agent_profile_id,omitempty"`
	DefaultConfigAgentProfileID *string                `json:"default_config_agent_profile_id,omitempty"`
	TaskFormConfig              *models.TaskFormConfig `json:"task_form_config,omitempty"`
	P4Client                    *string                `json:"p4_client,omitempty"`
}

func (h *WorkspaceHandlers) wsUpdateWorkspace(ctx context.Context, msg *ws.Message) (*ws.Message, error) {
	var req wsUpdateWorkspaceRequest
	if err := msg.ParsePayload(&req); err != nil {
		return ws.NewError(msg.ID, msg.Action, ws.ErrorCodeBadRequest, "Invalid payload: "+err.Error(), nil)
	}
	if req.ID == "" {
		return ws.NewError(msg.ID, msg.Action, ws.ErrorCodeValidation, "id is required", nil)
	}
	if req.TaskFormConfig != nil {
		if err := validateTaskFormConfig(*req.TaskFormConfig); err != nil {
			return ws.NewError(msg.ID, msg.Action, ws.ErrorCodeValidation, err.Error(), nil)
		}
	}

	workspace, err := h.service.UpdateWorkspace(ctx, req.ID, &service.UpdateWorkspaceRequest{
		Name:                        req.Name,
		Description:                 req.Description,
		DefaultEnvironmentID:        req.DefaultEnvironmentID,
		DefaultAgentProfileID:       req.DefaultAgentProfileID,
		DefaultConfigAgentProfileID: req.DefaultConfigAgentProfileID,
		TaskFormConfig:              req.TaskFormConfig,
		P4Client:                    req.P4Client,
	})
	if err != nil {
		h.logger.Error("failed to update workspace", zap.Error(err))
		return ws.NewError(msg.ID, msg.Action, ws.ErrorCodeInternalError, "Failed to update workspace", nil)
	}
	return ws.NewResponse(msg.ID, msg.Action, dto.FromWorkspace(workspace))
}

func (h *WorkspaceHandlers) wsDeleteWorkspace(ctx context.Context, msg *ws.Message) (*ws.Message, error) {
	return wsHandleIDRequest(ctx, msg, h.logger, "failed to delete workspace",
		func(ctx context.Context, id string) (any, error) {
			if err := h.service.DeleteWorkspace(ctx, id); err != nil {
				return nil, err
			}
			return dto.SuccessResponse{Success: true}, nil
		})
}
