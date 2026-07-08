package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/AvatarGanymede/pcraft/internal/orchestrator/executor"
	"github.com/AvatarGanymede/pcraft/internal/task/models"
	v1 "github.com/AvatarGanymede/pcraft/pkg/api/v1"
)

func TestGetTaskSessionStatus_AutoResumesNormalWaitingSession(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateWaitingForInput)

	// No ErrorMessage — normal idle session
	session, err := repo.GetTaskSession(ctx, "session1")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	session.AgentProfileID = "profile1"
	if err := repo.UpdateTaskSession(ctx, session); err != nil {
		t.Fatalf("failed to update session: %v", err)
	}

	now := time.Now().UTC()
	if err := repo.UpsertExecutorRunning(ctx, &models.ExecutorRunning{
		ID:        "er1",
		SessionID: "session1",
		TaskID:    "task1",
		Status:    "ready",
		Resumable: true,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("failed to upsert executor running: %v", err)
	}

	taskRepo := newMockTaskRepo()
	taskRepo.tasks["task1"] = &v1.Task{ID: "task1", State: v1.TaskStateInProgress}
	agentMgr := &mockAgentManager{repoForExecutionLookup: repo}
	svc := createTestServiceWithAgent(repo, newMockStepGetter(), taskRepo, agentMgr)
	svc.executor = executor.NewExecutor(agentMgr, repo, testLogger(), executor.ExecutorConfig{})

	resp, err := svc.GetTaskSessionStatus(ctx, "task1", "session1")
	if err != nil {
		t.Fatalf("GetTaskSessionStatus returned error: %v", err)
	}
	if !resp.NeedsResume {
		t.Fatal("expected NeedsResume=true for normal waiting session")
	}
	if !resp.IsResumable {
		t.Fatal("expected IsResumable=true")
	}
	if resp.ResumeReason != "agent_not_running_fresh_start" {
		t.Fatalf("expected ResumeReason=%q, got %q", "agent_not_running_fresh_start", resp.ResumeReason)
	}
}

func TestResumeTaskSession_WaitsForPromptReady(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateWaitingForInput)

	session, err := repo.GetTaskSession(ctx, "session1")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	session.AgentExecutionID = "exec-old-1"
	session.AgentProfileID = "profile1"
	if err := repo.UpdateTaskSession(ctx, session); err != nil {
		t.Fatalf("failed to update session: %v", err)
	}
	seedExecutorRunning(t, repo, session.ID, session.TaskID, "exec-old-1")

	ready := make(chan struct{})
	checked := make(chan struct{}, 1)
	agentMgr := &mockAgentManager{
		isAgentRunning:         false,
		repoForExecutionLookup: repo,
		isAgentReadyFn: func(_ context.Context, _ string) bool {
			select {
			case checked <- struct{}{}:
			default:
			}
			select {
			case <-ready:
				return true
			default:
				return false
			}
		},
		launchAgentFunc: func(_ context.Context, req *executor.LaunchAgentRequest) (*executor.LaunchAgentResponse, error) {
			go func(sessID string) {
				tick := time.NewTicker(5 * time.Millisecond)
				defer tick.Stop()
				timeout := time.After(5 * time.Second)
				for {
					select {
					case <-tick.C:
						sess, err := repo.GetTaskSession(context.Background(), sessID)
						if err == nil && sess != nil && sess.State == models.TaskSessionStateStarting {
							sess.State = models.TaskSessionStateWaitingForInput
							sess.UpdatedAt = time.Now().UTC()
							_ = repo.UpdateTaskSession(context.Background(), sess)
							return
						}
					case <-timeout:
						return
					}
				}
			}(req.SessionID)
			return &executor.LaunchAgentResponse{AgentExecutionID: "exec-resumed-1"}, nil
		},
	}

	taskRepo := newMockTaskRepo()
	taskRepo.tasks["task1"] = &v1.Task{
		ID:    "task1",
		Title: "Test Task",
		State: v1.TaskStateInProgress,
	}
	svc := createTestServiceWithAgent(repo, newMockStepGetter(), taskRepo, agentMgr)
	svc.executor = executor.NewExecutor(agentMgr, repo, testLogger(), executor.ExecutorConfig{})

	done := make(chan struct {
		exec *executor.TaskExecution
		err  error
	}, 1)
	go func() {
		exec, err := svc.ResumeTaskSession(ctx, "task1", "session1")
		done <- struct {
			exec *executor.TaskExecution
			err  error
		}{exec: exec, err: err}
	}()

	select {
	case <-checked:
	case <-time.After(3 * time.Second):
		t.Fatal("expected ResumeTaskSession to check prompt readiness")
	}

	select {
	case result := <-done:
		t.Fatalf("ResumeTaskSession returned before prompt readiness: %v", result.err)
	default:
	}

	close(ready)

	select {
	case result := <-done:
		if result.err != nil {
			t.Fatalf("ResumeTaskSession failed after prompt readiness: %v", result.err)
		}
		if result.exec == nil {
			t.Fatal("ResumeTaskSession returned nil execution")
		}
		if result.exec.SessionState != v1.TaskSessionStateWaitingForInput {
			t.Fatalf("expected WAITING_FOR_INPUT response state, got %s", result.exec.SessionState)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ResumeTaskSession did not return after prompt readiness")
	}
}
