package sqlite

import (
	"context"

	"github.com/AvatarGanymede/pcraft/internal/task/models"
)

// ListSessionsWithBranches returns sessions that have worktree branches
// on non-archived tasks. Stub: pcraft uses P4 workspaces, not git branches.
func (r *Repository) ListSessionsWithBranches(ctx context.Context) ([]models.SessionBranchInfo, error) {
	return nil, nil
}
