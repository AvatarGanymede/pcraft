// Package sqlite provides SQLite-based repository implementations.
package sqlite

import (
	"context"
	"errors"
	"fmt"

	"github.com/AvatarGanymede/pcraft/internal/task/models"
)

// ErrGitSnapshotNotSupported indicates git snapshot functionality has been removed.
var ErrGitSnapshotNotSupported = errors.New("git snapshots are not supported (pcraft uses P4 workspaces)")

// ErrSessionCommitNotSupported indicates session commit functionality has been removed.
var ErrSessionCommitNotSupported = errors.New("session commits are not supported (pcraft uses P4 workspaces)")

// TriggeredByLiveMonitor identifies snapshots written by the orchestrator's live
// git status persistence path. Used to scope the upsert in
// UpsertLatestLiveGitSnapshot so we don't disturb archive/completion snapshots.
const TriggeredByLiveMonitor = "live_monitor"

// UpsertLatestLiveGitSnapshot is not supported (pcraft uses P4 workspaces).
func (r *Repository) UpsertLatestLiveGitSnapshot(ctx context.Context, snapshot *models.GitSnapshot) error {
	return ErrGitSnapshotNotSupported
}

// DeleteLiveMonitorSnapshots is not supported (pcraft uses P4 workspaces).
func (r *Repository) DeleteLiveMonitorSnapshots(ctx context.Context, sessionID string) error {
	return fmt.Errorf("%w: %s", ErrGitSnapshotNotSupported, "DeleteLiveMonitorSnapshots is not supported (pcraft uses P4 workspaces)")
}

// CreateGitSnapshot is not supported (pcraft uses P4 workspaces).
func (r *Repository) CreateGitSnapshot(ctx context.Context, snapshot *models.GitSnapshot) error {
	return ErrGitSnapshotNotSupported
}

// GetLatestGitSnapshot is not supported (pcraft uses P4 workspaces).
func (r *Repository) GetLatestGitSnapshot(ctx context.Context, sessionID string) (*models.GitSnapshot, error) {
	return nil, ErrGitSnapshotNotSupported
}

// GetFirstGitSnapshot is not supported (pcraft uses P4 workspaces).
func (r *Repository) GetFirstGitSnapshot(ctx context.Context, sessionID string) (*models.GitSnapshot, error) {
	return nil, ErrGitSnapshotNotSupported
}

// GetGitSnapshotsBySession is not supported (pcraft uses P4 workspaces).
func (r *Repository) GetGitSnapshotsBySession(ctx context.Context, sessionID string, limit int) ([]*models.GitSnapshot, error) {
	return nil, ErrGitSnapshotNotSupported
}

// CreateSessionCommit is not supported (pcraft uses P4 workspaces).
func (r *Repository) CreateSessionCommit(ctx context.Context, commit *models.SessionCommit) error {
	return ErrSessionCommitNotSupported
}

// GetSessionCommits is not supported (pcraft uses P4 workspaces).
func (r *Repository) GetSessionCommits(ctx context.Context, sessionID string) ([]*models.SessionCommit, error) {
	return nil, ErrSessionCommitNotSupported
}

// GetLatestSessionCommit is not supported (pcraft uses P4 workspaces).
func (r *Repository) GetLatestSessionCommit(ctx context.Context, sessionID string) (*models.SessionCommit, error) {
	return nil, ErrSessionCommitNotSupported
}

// DeleteSessionCommit is not supported (pcraft uses P4 workspaces).
func (r *Repository) DeleteSessionCommit(ctx context.Context, id string) error {
	return ErrSessionCommitNotSupported
}
