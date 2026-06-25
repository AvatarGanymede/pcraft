package sqlite

import (
	"context"
	"time"

	"github.com/AvatarGanymede/pcraft/internal/analytics/models"
)

// parseTimeString parses time strings in various SQLite formats
func parseTimeString(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	// Try various common SQLite datetime formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05.000",
		"2006-01-02T15:04:05",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// GetTaskStats retrieves aggregated statistics for tasks in a workspace.
func (r *Repository) GetTaskStats(
	ctx context.Context,
	workspaceID string,
	start *time.Time,
	limit int,
) ([]*models.TaskStats, error) {
	return nil, nil
}

// GetAgentUsage retrieves usage statistics per agent profile.
// Stub: not supported in pcraft slim build.
func (r *Repository) GetAgentUsage(ctx context.Context, workspaceID string, limit int, start *time.Time) ([]*models.AgentUsage, error) {
	return nil, nil
}

// GetGitStats retrieves aggregated git statistics for a workspace.
// Stub: git statistics are not supported (pcraft uses P4 workspaces).
func (r *Repository) GetGitStats(ctx context.Context, workspaceID string, start *time.Time) (*models.GitStats, error) {
	return &models.GitStats{}, nil
}

// GetCompletedTaskActivity retrieves completed task counts per day.
// Stub: not supported in pcraft slim build.
func (r *Repository) GetCompletedTaskActivity(ctx context.Context, workspaceID string, days int) ([]*models.CompletedTaskActivity, error) {
	return nil, nil
}

// GetDailyActivity retrieves daily activity statistics.
// Stub: not supported in pcraft slim build.
func (r *Repository) GetDailyActivity(ctx context.Context, workspaceID string, days int) ([]*models.DailyActivity, error) {
	return nil, nil
}

// GetGlobalStats retrieves global workspace statistics.
// Stub: not supported in pcraft slim build.
func (r *Repository) GetGlobalStats(ctx context.Context, workspaceID string, start *time.Time) (*models.GlobalStats, error) {
	return &models.GlobalStats{}, nil
}

// GetRepositoryStats retrieves per-repository statistics.
// Stub: not supported in pcraft slim build.
func (r *Repository) GetRepositoryStats(ctx context.Context, workspaceID string, start *time.Time) ([]*models.RepositoryStats, error) {
	return nil, nil
}
