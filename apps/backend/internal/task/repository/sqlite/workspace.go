package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/AvatarGanymede/pcraft/internal/task/models"
)

// marshalTaskFormConfig serializes the form config for storage. A zero config
// (no fields) is stored as an empty string so older rows and new "default
// form" workspaces read back identically.
func marshalTaskFormConfig(cfg models.TaskFormConfig) string {
	if cfg.IsZero() {
		return ""
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return ""
	}
	return string(b)
}

// scanTaskFormConfig parses a stored form config. Empty/invalid JSON yields a
// zero config, which callers interpret as the default single-prompt form.
func scanTaskFormConfig(raw sql.NullString) models.TaskFormConfig {
	if !raw.Valid || raw.String == "" {
		return models.TaskFormConfig{}
	}
	var cfg models.TaskFormConfig
	if err := json.Unmarshal([]byte(raw.String), &cfg); err != nil {
		return models.TaskFormConfig{}
	}
	return cfg
}

// CreateWorkspace creates a new workspace
func (r *Repository) CreateWorkspace(ctx context.Context, workspace *models.Workspace) error {
	if workspace.ID == "" {
		workspace.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	workspace.CreatedAt = now
	workspace.UpdatedAt = now
	if workspace.TaskPrefix == "" {
		workspace.TaskPrefix = "KAN"
	}

	_, err := r.db.ExecContext(ctx, r.db.Rebind(`
		INSERT INTO workspaces (
			id,
			name,
			description,
			owner_id,
			default_environment_id,
			default_agent_profile_id,
			default_config_agent_profile_id,
			task_prefix,
			task_sequence,
			office_workflow_id,
			task_form_config,
			p4_client,
			p4_root,
			p4_stream,
			created_at,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`), workspace.ID, workspace.Name, workspace.Description, workspace.OwnerID, workspace.DefaultEnvironmentID, workspace.DefaultAgentProfileID, workspace.DefaultConfigAgentProfileID, workspace.TaskPrefix, workspace.TaskSequence, workspace.OfficeWorkflowID, marshalTaskFormConfig(workspace.TaskFormConfig), workspace.P4Client, workspace.P4Root, workspace.P4Stream, workspace.CreatedAt, workspace.UpdatedAt)

	return err
}

// GetWorkspace retrieves a workspace by ID
func (r *Repository) GetWorkspace(ctx context.Context, id string) (*models.Workspace, error) {
	workspace := &models.Workspace{}
	var defaultEnvironmentID sql.NullString
	var defaultAgentProfileID sql.NullString
	var defaultConfigAgentProfileID sql.NullString
	var taskFormConfig sql.NullString

	err := r.ro.QueryRowContext(ctx, r.ro.Rebind(`
		SELECT id, name, description, owner_id, default_environment_id, default_agent_profile_id, default_config_agent_profile_id, task_prefix, task_sequence, office_workflow_id, task_form_config, p4_client, p4_root, p4_stream, created_at, updated_at
		FROM workspaces WHERE id = ?
	`), id).Scan(
		&workspace.ID,
		&workspace.Name,
		&workspace.Description,
		&workspace.OwnerID,
		&defaultEnvironmentID,
		&defaultAgentProfileID,
		&defaultConfigAgentProfileID,
		&workspace.TaskPrefix,
		&workspace.TaskSequence,
		&workspace.OfficeWorkflowID,
		&taskFormConfig,
		&workspace.P4Client,
		&workspace.P4Root,
		&workspace.P4Stream,
		&workspace.CreatedAt,
		&workspace.UpdatedAt,
	)
	if defaultEnvironmentID.Valid && defaultEnvironmentID.String != "" {
		workspace.DefaultEnvironmentID = &defaultEnvironmentID.String
	}
	if defaultAgentProfileID.Valid && defaultAgentProfileID.String != "" {
		workspace.DefaultAgentProfileID = &defaultAgentProfileID.String
	}
	if defaultConfigAgentProfileID.Valid && defaultConfigAgentProfileID.String != "" {
		workspace.DefaultConfigAgentProfileID = &defaultConfigAgentProfileID.String
	}
	workspace.TaskFormConfig = scanTaskFormConfig(taskFormConfig)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workspace not found: %s", id)
	}
	return workspace, err
}

// UpdateWorkspace updates an existing workspace
func (r *Repository) UpdateWorkspace(ctx context.Context, workspace *models.Workspace) error {
	workspace.UpdatedAt = time.Now().UTC()

	result, err := r.db.ExecContext(ctx, r.db.Rebind(`
		UPDATE workspaces
		SET name = ?,
			description = ?,
			default_environment_id = ?,
			default_agent_profile_id = ?,
			default_config_agent_profile_id = ?,
			task_form_config = ?,
			p4_client = ?,
			p4_root = ?,
			p4_stream = ?,
			updated_at = ?
		WHERE id = ?
	`), workspace.Name, workspace.Description, workspace.DefaultEnvironmentID, workspace.DefaultAgentProfileID, workspace.DefaultConfigAgentProfileID, marshalTaskFormConfig(workspace.TaskFormConfig), workspace.P4Client, workspace.P4Root, workspace.P4Stream, workspace.UpdatedAt, workspace.ID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("workspace not found: %s", workspace.ID)
	}
	return nil
}

// DeleteWorkspace deletes a workspace by ID
func (r *Repository) DeleteWorkspace(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, r.db.Rebind(`DELETE FROM workspaces WHERE id = ?`), id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("workspace not found: %s", id)
	}
	return nil
}

// ListWorkspaces returns all workspaces
func (r *Repository) ListWorkspaces(ctx context.Context) ([]*models.Workspace, error) {
	rows, err := r.ro.QueryContext(ctx, `
		SELECT id, name, description, owner_id, default_environment_id, default_agent_profile_id, default_config_agent_profile_id, task_prefix, task_sequence, office_workflow_id, task_form_config, p4_client, p4_root, p4_stream, created_at, updated_at
		FROM workspaces ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []*models.Workspace
	for rows.Next() {
		workspace := &models.Workspace{}
		var defaultEnvironmentID sql.NullString
		var defaultAgentProfileID sql.NullString
		var defaultConfigAgentProfileID sql.NullString
		var taskFormConfig sql.NullString
		if err := rows.Scan(
			&workspace.ID,
			&workspace.Name,
			&workspace.Description,
			&workspace.OwnerID,
			&defaultEnvironmentID,
			&defaultAgentProfileID,
			&defaultConfigAgentProfileID,
			&workspace.TaskPrefix,
			&workspace.TaskSequence,
			&workspace.OfficeWorkflowID,
			&taskFormConfig,
			&workspace.P4Client,
			&workspace.P4Root,
			&workspace.P4Stream,
			&workspace.CreatedAt,
			&workspace.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if defaultEnvironmentID.Valid && defaultEnvironmentID.String != "" {
			workspace.DefaultEnvironmentID = &defaultEnvironmentID.String
		}
		if defaultAgentProfileID.Valid && defaultAgentProfileID.String != "" {
			workspace.DefaultAgentProfileID = &defaultAgentProfileID.String
		}
		if defaultConfigAgentProfileID.Valid && defaultConfigAgentProfileID.String != "" {
			workspace.DefaultConfigAgentProfileID = &defaultConfigAgentProfileID.String
		}
		workspace.TaskFormConfig = scanTaskFormConfig(taskFormConfig)
		result = append(result, workspace)
	}
	return result, rows.Err()
}
