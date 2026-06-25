package service

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/AvatarGanymede/pcraft/internal/task/models"
)

// EnvironmentDestroyer tears down the runtime resources recorded on a TaskEnvironment.
// Implemented by an adapter over the lifecycle.Manager and worktree.Manager.
// Worktree destruction preserves the underlying branch — user data that hasn't been
// pushed is never deleted by reset.
type EnvironmentDestroyer interface {
	DestroyWorktree(ctx context.Context, worktreeID string) error
	// PushEnvironmentBranch best-effort pushes the current branch of the environment's
	// workspace to its upstream. Returns an error if the push fails; callers can decide
	// whether to abort the reset on failure.
	PushEnvironmentBranch(ctx context.Context, env *models.TaskEnvironment) error
}

// ResetOptions controls destructive behavior of ResetTaskEnvironment.
type ResetOptions struct {
	// PushBranch runs `git push` on the environment's current branch before teardown.
	// If the push fails, ResetTaskEnvironment aborts and the environment stays intact.
	PushBranch bool
}

// SessionRunningChecker reports whether a session has a live executor running.
// Used by ResetTaskEnvironment to block resets when the user still has an agent
// attached to the environment.
type SessionRunningChecker interface {
	IsAnySessionRunningForTask(ctx context.Context, taskID string) (bool, error)
}

// ErrSessionRunning is returned by ResetTaskEnvironment when at least one session
// on the task is still actively running and the caller must stop it first.
var ErrSessionRunning = errors.New("active session is running on this task; stop it before resetting the environment")

// ErrNoEnvironment is returned when the task has no TaskEnvironment to reset.
var ErrNoEnvironment = errors.New("no environment exists for this task")

// SetEnvironmentDestroyer wires the runtime-resource destroyer used by ResetTaskEnvironment.
func (s *Service) SetEnvironmentDestroyer(d EnvironmentDestroyer) {
	s.envDestroyer = d
}

// SetSessionRunningChecker wires a custom running-session guard. When unset, the
// service falls back to a default implementation over the existing session repo.
func (s *Service) SetSessionRunningChecker(c SessionRunningChecker) {
	s.sessionRunningChecker = c
}

// checkAnySessionRunning delegates to the configured checker or falls back to
// the default (session has an ExecutorRunning row).
func (s *Service) checkAnySessionRunning(ctx context.Context, taskID string) (bool, error) {
	if s.sessionRunningChecker != nil {
		return s.sessionRunningChecker.IsAnySessionRunningForTask(ctx, taskID)
	}
	return s.isAnySessionRunning(ctx, taskID)
}

// isAnySessionRunning is the default SessionRunningChecker: a session is running
// if it has an ExecutorRunning row and is actively starting or processing a turn.
// Idle sessions can keep an executor row so terminals and resumed prompts have
// a live workspace, but Reset Environment must still be able to tear that
// workspace down once the agent is waiting for input.
func (s *Service) isAnySessionRunning(ctx context.Context, taskID string) (bool, error) {
	sessions, err := s.sessions.ListTaskSessions(ctx, taskID)
	if err != nil {
		return false, err
	}
	for _, sess := range sessions {
		if sess == nil {
			continue
		}
		running, err := s.executors.GetExecutorRunningBySessionID(ctx, sess.ID)
		if err != nil {
			if errors.Is(err, models.ErrExecutorRunningNotFound) {
				continue
			}
			return false, err
		}
		if running != nil && sessionBlocksEnvironmentReset(sess.State) {
			return true, nil
		}
	}
	return false, nil
}

func sessionBlocksEnvironmentReset(state models.TaskSessionState) bool {
	return state == models.TaskSessionStateStarting || state == models.TaskSessionStateRunning
}

// GetTaskEnvironmentByTaskID returns the active task environment for a task.
// Returns nil if no environment exists yet.
func (s *Service) GetTaskEnvironmentByTaskID(ctx context.Context, taskID string) (*models.TaskEnvironment, error) {
	return s.taskEnvironments.GetTaskEnvironmentByTaskID(ctx, taskID)
}

// ResetTaskEnvironment tears down the task's current environment (worktree)
// and deletes the TaskEnvironment row, so the next session launch starts fresh.
//
// Guards:
//   - No environment → ErrNoEnvironment
//   - Any session on the task is actively running → ErrSessionRunning
//
// Cleanup is best-effort. If the worktree fails to destroy, the
// TaskEnvironment row is preserved so the user can retry. Success deletes the row.
//
// If opts.PushBranch is set, the branch is pushed before teardown; a failed push
// aborts the reset and leaves the environment intact so the user can investigate.
func (s *Service) ResetTaskEnvironment(ctx context.Context, taskID string, opts ResetOptions) error {
	env, err := s.taskEnvironments.GetTaskEnvironmentByTaskID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("lookup environment: %w", err)
	}
	if env == nil {
		return ErrNoEnvironment
	}

	// Fail closed: if the running-session check itself errors (DB hiccup,
	// locked table) we cannot prove the task is idle and must abort rather
	// than risk destroying a worktree while an agent is still writing to it.
	running, err := s.checkAnySessionRunning(ctx, taskID)
	if err != nil {
		return fmt.Errorf("check running sessions before reset: %w", err)
	}
	if running {
		return ErrSessionRunning
	}

	s.logger.Info("resetting task environment",
		zap.String("task_id", taskID),
		zap.String("env_id", env.ID),
		zap.String("executor_type", env.ExecutorType),
		zap.String("worktree_id", env.WorktreeID),
		zap.Bool("push_branch", opts.PushBranch))

	if opts.PushBranch {
		if s.envDestroyer == nil {
			return fmt.Errorf("environment destroyer not configured; cannot push branch")
		}
		if err := s.envDestroyer.PushEnvironmentBranch(ctx, env); err != nil {
			return fmt.Errorf("push branch before reset: %w", err)
		}
	}

	if err := s.teardownEnvironmentResources(ctx, env); err != nil {
		return err
	}

	if err := s.taskEnvironments.DeleteTaskEnvironment(ctx, env.ID); err != nil {
		return fmt.Errorf("delete task environment row: %w", err)
	}
	s.logger.Info("task environment reset complete",
		zap.String("task_id", taskID),
		zap.String("env_id", env.ID))
	return nil
}

// teardownEnvironmentResources destroys the worktree recorded on a TaskEnvironment.
// On any error, the caller should preserve the row so the user can retry.
func (s *Service) teardownEnvironmentResources(ctx context.Context, env *models.TaskEnvironment) error {
	if env.WorktreeID == "" {
		return nil
	}
	if s.envDestroyer == nil {
		return fmt.Errorf("environment destroyer not configured; cannot tear down resources")
	}

	if err := s.envDestroyer.DestroyWorktree(ctx, env.WorktreeID); err != nil {
		return fmt.Errorf("destroy worktree %s: %w", env.WorktreeID, err)
	}
	return nil
}
