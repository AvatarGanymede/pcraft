package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	sqliterepo "github.com/AvatarGanymede/pcraft/internal/task/repository/sqlite"

	"github.com/AvatarGanymede/pcraft/internal/agent/runtime/lifecycle"
	"github.com/AvatarGanymede/pcraft/internal/agentctl/types/streams"
	"github.com/AvatarGanymede/pcraft/internal/events"
	"github.com/AvatarGanymede/pcraft/internal/orchestrator/dto"
	"github.com/AvatarGanymede/pcraft/internal/orchestrator/executor"
	"github.com/AvatarGanymede/pcraft/internal/orchestrator/messagequeue"
	"github.com/AvatarGanymede/pcraft/internal/orchestrator/queue"
	"github.com/AvatarGanymede/pcraft/internal/orchestrator/scheduler"
	"github.com/AvatarGanymede/pcraft/internal/sysprompt"
	"github.com/AvatarGanymede/pcraft/internal/task/models"
	wfmodels "github.com/AvatarGanymede/pcraft/internal/workflow/models"
	v1 "github.com/AvatarGanymede/pcraft/pkg/api/v1"
)

// seedTaskAndSession inserts a workspace, workflow, task, and session with the given state.
func seedTaskAndSession(t *testing.T, repo *sqliterepo.Repository, taskID, sessionID string, sessionState models.TaskSessionState) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	ws := &models.Workspace{ID: "ws1", Name: "Test", CreatedAt: now, UpdatedAt: now}
	_ = repo.CreateWorkspace(ctx, ws)

	wf := &models.Workflow{ID: "wf1", WorkspaceID: "ws1", Name: "Test Workflow", CreatedAt: now, UpdatedAt: now}
	_ = repo.CreateWorkflow(ctx, wf)

	task := &models.Task{
		ID:          taskID,
		WorkflowID:  "wf1",
		Title:       "Test Task",
		Description: "desc",
		State:       v1.TaskStateInProgress,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	session := &models.TaskSession{
		ID:        sessionID,
		TaskID:    taskID,
		State:     sessionState,
		StartedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateTaskSession(ctx, session); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
}

// --- PromptTask ---

func TestPromptTask_EmptySessionID(t *testing.T) {
	repo := setupTestRepo(t)
	svc := createTestService(repo, newMockStepGetter(), newMockTaskRepo())
	_, err := svc.PromptTask(context.Background(), "task1", "", "hello", "", false, nil, false)
	if err == nil {
		t.Fatal("expected error for empty session_id")
	}
}

func TestPromptTask_SessionAlreadyRunning(t *testing.T) {
	repo := setupTestRepo(t)
	svc := createTestService(repo, newMockStepGetter(), newMockTaskRepo())

	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateRunning)

	_, err := svc.PromptTask(context.Background(), "task1", "session1", "hello", "", false, nil, false)
	if err == nil {
		t.Fatal("expected error when session is already RUNNING")
	}
}

func TestPromptTask_WaitsForStartingSessionBeforePrompt(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateStarting)

	session, err := repo.GetTaskSession(ctx, "session1")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	session.AgentExecutionID = "exec-resumed-1"
	session.AgentProfileID = "profile1"
	if err := repo.UpdateTaskSession(ctx, session); err != nil {
		t.Fatalf("failed to update session: %v", err)
	}
	seedExecutorRunning(t, repo, session.ID, session.TaskID, "exec-resumed-1")

	promptReady := make(chan struct{})
	readinessChecked := make(chan struct{}, 1)
	agentMgr := &mockAgentManager{
		isAgentRunning:         true,
		repoForExecutionLookup: repo,
		promptResult: &executor.PromptResult{
			StopReason:   "end_turn",
			AgentMessage: "simple mock response",
		},
		isAgentReadyFn: func(_ context.Context, _ string) bool {
			select {
			case readinessChecked <- struct{}{}:
			default:
			}
			select {
			case <-promptReady:
				return true
			default:
				return false
			}
		},
	}
	taskRepo := newMockTaskRepo()
	taskRepo.tasks["task1"] = &v1.Task{ID: "task1", State: v1.TaskStateInProgress}
	svc := createTestServiceWithAgent(repo, newMockStepGetter(), taskRepo, agentMgr)
	svc.executor = executor.NewExecutor(agentMgr, repo, testLogger(), executor.ExecutorConfig{})

	done := make(chan struct {
		result *PromptResult
		err    error
	}, 1)
	go func() {
		result, err := svc.PromptTask(ctx, "task1", "session1", "/e2e:simple-message", "", false, nil, false)
		done <- struct {
			result *PromptResult
			err    error
		}{result: result, err: err}
	}()

	go func() {
		time.Sleep(25 * time.Millisecond)
		readySession, err := repo.GetTaskSession(context.Background(), "session1")
		if err != nil || readySession == nil {
			return
		}
		readySession.State = models.TaskSessionStateWaitingForInput
		readySession.UpdatedAt = time.Now().UTC()
		_ = repo.UpdateTaskSession(context.Background(), readySession)
	}()

	select {
	case <-readinessChecked:
	case <-time.After(2 * time.Second):
		t.Fatal("expected PromptTask to wait for agent prompt readiness")
	}

	select {
	case result := <-done:
		t.Fatalf("PromptTask returned before prompt readiness: result=%#v err=%v", result.result, result.err)
	default:
	}

	close(promptReady)

	select {
	case result := <-done:
		if result.err != nil {
			t.Fatalf("PromptTask failed after prompt readiness: %v", result.err)
		}
		if result.result == nil {
			t.Fatal("PromptTask returned nil result")
		}
		if result.result.AgentMessage != "simple mock response" {
			t.Fatalf("unexpected agent message: %q", result.result.AgentMessage)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("PromptTask did not return after prompt readiness")
	}

	agentMgr.mu.Lock()
	prompts := append([]string(nil), agentMgr.capturedPrompts...)
	calls := append([]promptCall(nil), agentMgr.capturedPromptCalls...)
	agentMgr.mu.Unlock()
	if len(prompts) != 1 {
		t.Fatalf("expected one prompt after readiness, got %d", len(prompts))
	}
	if prompts[0] != "/e2e:simple-message" {
		t.Fatalf("unexpected prompt: %q", prompts[0])
	}
	if len(calls) != 1 || calls[0].ExecutionID != "exec-resumed-1" {
		t.Fatalf("unexpected prompt calls: %#v", calls)
	}
}

func TestTrySwitchModelUpdatesRuntimeModelCache(t *testing.T) {
	repo := setupTestRepo(t)
	agentMgr := &mockAgentManager{
		isAgentRunning:           true,
		setSessionModelSupported: true,
	}
	svc := createTestServiceWithAgent(repo, newMockStepGetter(), newMockTaskRepo(), agentMgr)
	svc.executor = executor.NewExecutor(agentMgr, repo, testLogger(), executor.ExecutorConfig{})

	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateWaitingForInput)
	session, err := repo.GetTaskSession(context.Background(), "session1")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	session.AgentProfileSnapshot = map[string]interface{}{"model": "gpt-5.5"}
	seedExecutorRunning(t, repo, session.ID, session.TaskID, "exec-1")
	if err := repo.UpdateTaskSession(context.Background(), session); err != nil {
		t.Fatalf("failed to update session: %v", err)
	}
	svc.runtimeModelBySession.Store("session1", "gpt-5.5")

	result, switched, err := svc.trySwitchModel(context.Background(), "task1", "session1", "gpt-5.3-codex-spark", "continue", session)
	if err != nil {
		t.Fatalf("trySwitchModel returned error: %v", err)
	}
	if switched {
		t.Fatal("in-place model switch should let prompt dispatch continue")
	}
	if result != nil {
		t.Fatalf("expected nil prompt result for in-place switch, got %#v", result)
	}
	if len(agentMgr.setSessionModelCalls) != 1 {
		t.Fatalf("expected one model switch call, got %d", len(agentMgr.setSessionModelCalls))
	}
	if agentMgr.setSessionModelCalls[0] != (sessionModelCall{SessionID: "session1", ModelID: "gpt-5.3-codex-spark"}) {
		t.Fatalf("unexpected model switch call: %#v", agentMgr.setSessionModelCalls[0])
	}
	cached, ok := svc.runtimeModelBySession.Load("session1")
	if !ok {
		t.Fatal("expected runtime model cache entry")
	}
	if cached != "gpt-5.3-codex-spark" {
		t.Fatalf("expected runtime model cache to update, got %#v", cached)
	}
}

// TestPromptTask_ExecutionNotFoundRevertsStateAndBroadcasts ensures that when
// Prompt returns executor.ErrExecutionNotFound, PromptTask reverts the session
// state via the broadcasting wrapper (not a direct repo write), so the WS
// subscribers receive session.state_changed and the UI can unstick the
// "Agent is running" composer/pause button.
// Regression test for the stuck-UI bug after a prompt failure.
func TestPromptTask_ExecutionNotFoundRevertsStateAndBroadcasts(t *testing.T) {
	repo := setupTestRepo(t)
	taskRepo := newMockTaskRepo()
	agentMgr := &mockAgentManager{
		isAgentRunning: true,
		promptErr:      fmt.Errorf("wrapped: %w", lifecycle.ErrExecutionNotFound),
	}
	eb := &recordingEventBus{}
	svc := createTestServiceWithAgent(repo, newMockStepGetter(), taskRepo, agentMgr)
	svc.executor = executor.NewExecutor(agentMgr, repo, testLogger(), executor.ExecutorConfig{})
	svc.eventBus = eb

	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateWaitingForInput)
	session, err := repo.GetTaskSession(context.Background(), "session1")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	session.AgentExecutionID = "exec-1"
	seedExecutorRunning(t, repo, session.ID, session.TaskID, "exec-1")
	if err := repo.UpdateTaskSession(context.Background(), session); err != nil {
		t.Fatalf("failed to update session: %v", err)
	}

	_, err = svc.PromptTask(context.Background(), "task1", "session1", "hello", "", false, nil, false)
	if err == nil {
		t.Fatal("expected error from prompt, got nil")
	}
	if !errors.Is(err, executor.ErrExecutionNotFound) {
		t.Fatalf("expected ErrExecutionNotFound bubbled up, got: %v", err)
	}

	updated, err := repo.GetTaskSession(context.Background(), "session1")
	if err != nil {
		t.Fatalf("failed to reload session: %v", err)
	}
	if updated.State != models.TaskSessionStateWaitingForInput {
		t.Fatalf("expected session state WAITING_FOR_INPUT after revert, got %q", updated.State)
	}

	var sawRevert bool
	for _, evt := range eb.events {
		if evt.subject != events.TaskSessionStateChanged {
			continue
		}
		payload, ok := evt.event.Data.(map[string]interface{})
		if !ok {
			continue
		}
		oldState, _ := payload["old_state"].(string)
		newState, _ := payload["new_state"].(string)
		sessID, _ := payload["session_id"].(string)
		if sessID == "session1" && oldState == string(models.TaskSessionStateRunning) && newState == string(models.TaskSessionStateWaitingForInput) {
			sawRevert = true
			break
		}
	}
	if !sawRevert {
		t.Fatalf("expected TaskSessionStateChanged RUNNING→WAITING_FOR_INPUT broadcast after prompt failure, got events: %+v", eb.events)
	}
}

func TestPromptTask_PlanModeInjectsPrefix(t *testing.T) {
	repo := setupTestRepo(t)
	taskRepo := newMockTaskRepo()
	agentMgr := &mockAgentManager{
		isAgentRunning: true,
	}
	svc := createTestServiceWithAgent(repo, newMockStepGetter(), taskRepo, agentMgr)
	svc.executor = executor.NewExecutor(agentMgr, repo, testLogger(), executor.ExecutorConfig{})

	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateWaitingForInput)
	session, err := repo.GetTaskSession(context.Background(), "session1")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	session.AgentExecutionID = "exec-1"
	seedExecutorRunning(t, repo, session.ID, session.TaskID, "exec-1")
	if err := repo.UpdateTaskSession(context.Background(), session); err != nil {
		t.Fatalf("failed to update session: %v", err)
	}

	_, err = svc.PromptTask(context.Background(), "task1", "session1", "update the plan", "", true, nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(agentMgr.capturedPrompts) != 1 {
		t.Fatalf("expected 1 prompt, got %d", len(agentMgr.capturedPrompts))
	}
	if !strings.Contains(agentMgr.capturedPrompts[0], "PLAN MODE ACTIVE") {
		t.Fatalf("expected plan mode prefix in prompt, got: %s", agentMgr.capturedPrompts[0])
	}
}

func TestPromptTask_NoPlanModeDoesNotInjectPrefix(t *testing.T) {
	repo := setupTestRepo(t)
	taskRepo := newMockTaskRepo()
	agentMgr := &mockAgentManager{
		isAgentRunning: true,
	}
	svc := createTestServiceWithAgent(repo, newMockStepGetter(), taskRepo, agentMgr)
	svc.executor = executor.NewExecutor(agentMgr, repo, testLogger(), executor.ExecutorConfig{})

	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateWaitingForInput)
	session, err := repo.GetTaskSession(context.Background(), "session1")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	session.AgentExecutionID = "exec-1"
	seedExecutorRunning(t, repo, session.ID, session.TaskID, "exec-1")
	if err := repo.UpdateTaskSession(context.Background(), session); err != nil {
		t.Fatalf("failed to update session: %v", err)
	}

	_, err = svc.PromptTask(context.Background(), "task1", "session1", "implement the feature", "", false, nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(agentMgr.capturedPrompts) != 1 {
		t.Fatalf("expected 1 prompt, got %d", len(agentMgr.capturedPrompts))
	}
	if strings.Contains(agentMgr.capturedPrompts[0], "PLAN MODE ACTIVE") {
		t.Fatalf("expected no plan mode prefix in prompt, got: %s", agentMgr.capturedPrompts[0])
	}
}

func TestPromptTask_ResetInProgressReturnsSentinelError(t *testing.T) {
	repo := setupTestRepo(t)
	taskRepo := newMockTaskRepo()
	agentMgr := &mockAgentManager{repoForExecutionLookup: repo}
	svc := createTestServiceWithAgent(repo, newMockStepGetter(), taskRepo, agentMgr)
	svc.executor = executor.NewExecutor(agentMgr, repo, testLogger(), executor.ExecutorConfig{})

	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateWaitingForInput)
	svc.setSessionResetInProgress("session1", true)
	defer svc.setSessionResetInProgress("session1", false)

	_, err := svc.PromptTask(context.Background(), "task1", "session1", "hello", "", false, nil, false)
	if !errors.Is(err, ErrSessionResetInProgress) {
		t.Fatalf("expected ErrSessionResetInProgress, got %v", err)
	}
}

// --- CancelAgent ---

// TestCancelAgent_DeduplicatesConcurrentCalls covers the impatient-user case:
// the UI's cancel button has no in-flight disable, so users click it multiple
// times while the agent is still tearing down a slow turn (e.g. a Claude
// Monitor tool). Without dedupe each click reaches the lifecycle layer and
// emits its own "Turn cancelled by user" message; phantom turns are also
// lazily started to host those messages. We assert that only one cancel makes
// it through to agentManager.CancelAgent while one is already in flight.
func TestCancelAgent_DeduplicatesConcurrentCalls(t *testing.T) {
	repo := setupTestRepo(t)
	agentMgr := &mockAgentManager{
		isAgentRunning:     true,
		cancelAgentBlock:   make(chan struct{}),
		cancelAgentEntered: make(chan struct{}, 1),
	}
	svc := createTestServiceWithAgent(repo, newMockStepGetter(), newMockTaskRepo(), agentMgr)

	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateRunning)

	// First call goes async and parks inside agentManager.CancelAgent.
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- svc.CancelAgent(context.Background(), "session1")
	}()

	// Wait for the first call to actually enter agentManager.CancelAgent so
	// the dedupe guard is set before the duplicate calls fire. Channel sync
	// (over sleep-based polling) is the project convention for tests that
	// don't depend on real subprocess timing.
	<-agentMgr.cancelAgentEntered

	// Fire several duplicates while the first is still parked. Each must be
	// short-circuited by the dedupe guard and return immediately.
	const duplicates = 5
	for i := 0; i < duplicates; i++ {
		if err := svc.CancelAgent(context.Background(), "session1"); err != nil {
			t.Fatalf("duplicate cancel %d returned error: %v", i, err)
		}
	}
	if got := agentMgr.cancelAgentCalls.Load(); got != 1 {
		t.Fatalf("expected exactly 1 agentManager.CancelAgent call while first is in flight, got %d", got)
	}

	// Release the first call. After it returns, the guard clears and a fresh
	// cancel is allowed through.
	close(agentMgr.cancelAgentBlock)
	if err := <-firstDone; err != nil {
		t.Fatalf("first CancelAgent returned error: %v", err)
	}

	agentMgr.cancelAgentBlock = nil // unblock subsequent calls
	if err := svc.CancelAgent(context.Background(), "session1"); err != nil {
		t.Fatalf("post-release CancelAgent returned error: %v", err)
	}
	if got := agentMgr.cancelAgentCalls.Load(); got != 2 {
		t.Fatalf("expected 2 agentManager.CancelAgent calls after release, got %d", got)
	}
}

func TestCancelAgent_LeavesQueuedMessageForManualDrain(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	agentMgr := &mockAgentManager{isAgentRunning: true}
	svc := createTestServiceWithAgent(repo, newMockStepGetter(), newMockTaskRepo(), agentMgr)
	svc.executor = executor.NewExecutor(agentMgr, repo, testLogger(), executor.ExecutorConfig{})

	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateRunning)
	seedExecutorRunning(t, repo, "session1", "task1", "exec-1")
	if _, err := svc.messageQueue.QueueMessage(
		ctx, "session1", "task1", "queued after cancel", "", messagequeue.QueuedByUser, false, nil,
	); err != nil {
		t.Fatalf("queue message: %v", err)
	}

	if err := svc.CancelAgent(ctx, "session1"); err != nil {
		t.Fatalf("cancel agent: %v", err)
	}

	updated, err := repo.GetTaskSession(ctx, "session1")
	if err != nil {
		t.Fatalf("get updated session: %v", err)
	}
	if updated.State != models.TaskSessionStateWaitingForInput {
		t.Fatalf("expected session state %q, got %q", models.TaskSessionStateWaitingForInput, updated.State)
	}

	status := svc.messageQueue.GetStatus(ctx, "session1")
	if status.Count != 1 {
		t.Fatalf("expected cancel to leave queued message for manual drain, count=%d entries=%+v", status.Count, status.Entries)
	}
	if got := status.Entries[0].Content; got != "queued after cancel" {
		t.Fatalf("queued prompt = %q, want %q", got, "queued after cancel")
	}
	if len(agentMgr.capturedPrompts) != 0 {
		t.Fatalf("expected cancel not to prompt queued message, got %d prompts", len(agentMgr.capturedPrompts))
	}
}

// --- StartCreatedSession ---

func TestStartCreatedSession_WrongTask(t *testing.T) {
	repo := setupTestRepo(t)
	svc := createTestService(repo, newMockStepGetter(), newMockTaskRepo())

	// Session belongs to "task-other", not "task1"
	seedTaskAndSession(t, repo, "task-other", "session1", models.TaskSessionStateCreated)

	_, err := svc.StartCreatedSession(context.Background(), "task1", "session1", "profile1", "prompt", false, false, false, nil)
	if err == nil {
		t.Fatal("expected error when session does not belong to task")
	}
}

func TestStartCreatedSession_NotInCreatedState(t *testing.T) {
	repo := setupTestRepo(t)
	svc := createTestService(repo, newMockStepGetter(), newMockTaskRepo())

	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateRunning)

	_, err := svc.StartCreatedSession(context.Background(), "task1", "session1", "profile1", "prompt", false, false, false, nil)
	if err == nil {
		t.Fatal("expected error when session is not in CREATED state")
	}
}

func TestStartCreatedSession_WorkflowOverridePromotesPreparedWhenTaskHasNoPrimary(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	now := time.Now().UTC()

	if err := repo.CreateWorkspace(ctx, &models.Workspace{ID: "ws1", Name: "Test", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if err := repo.CreateWorkflow(ctx, &models.Workflow{ID: "wf1", WorkspaceID: "ws1", Name: "Workflow", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	if err := repo.CreateTask(ctx, &models.Task{
		ID:             "task1",
		WorkspaceID:    "ws1",
		WorkflowID:     "wf1",
		WorkflowStepID: "step1",
		Title:          "Task",
		Description:    "desc",
		State:          v1.TaskStateInProgress,
		Metadata:       map[string]interface{}{models.MetaKeyAgentProfileID: "profile-a"},
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("create task: %v", err)
	}

	stepGetter := newMockStepGetter()
	stepGetter.steps["step1"] = &wfmodels.WorkflowStep{
		ID:             "step1",
		WorkflowID:     "wf1",
		Name:           "Step 1",
		AgentProfileID: "profile-b",
	}
	taskRepo := newMockTaskRepo()
	taskRepo.tasks["task1"] = &v1.Task{
		ID:          "task1",
		WorkspaceID: "ws1",
		WorkflowID:  "wf1",
		Title:       "Task",
		Description: "desc",
		State:       v1.TaskStateInProgress,
		Metadata:    map[string]interface{}{models.MetaKeyAgentProfileID: "profile-a"},
	}

	var launchedProfile string
	agentMgr := &mockAgentManager{
		launchAgentFunc: func(_ context.Context, req *executor.LaunchAgentRequest) (*executor.LaunchAgentResponse, error) {
			launchedProfile = req.AgentProfileID
			return &executor.LaunchAgentResponse{AgentExecutionID: "exec-1"}, nil
		},
	}
	svc := createTestServiceWithScheduler(repo, stepGetter, taskRepo, agentMgr)

	sessionID, err := svc.PrepareTaskSession(ctx, "task1", "profile-a", "", "", "step1", false)
	if err != nil {
		t.Fatalf("PrepareTaskSession: %v", err)
	}
	prepared, err := repo.GetTaskSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetTaskSession after prepare: %v", err)
	}
	if !prepared.IsPrimary {
		t.Fatal("prepared first session should start as primary")
	}
	prepared.IsPrimary = false
	if err := repo.UpdateTaskSession(ctx, prepared); err != nil {
		t.Fatalf("clear prepared primary flag: %v", err)
	}

	if _, err := svc.StartCreatedSession(ctx, "task1", sessionID, "profile-a", "desc", true, false, true, nil); err != nil {
		t.Fatalf("StartCreatedSession: %v", err)
	}

	updated, err := repo.GetTaskSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetTaskSession after start: %v", err)
	}
	if updated.AgentProfileID != "profile-b" {
		t.Fatalf("agent profile = %q, want profile-b", updated.AgentProfileID)
	}
	if !updated.IsPrimary {
		t.Fatal("workflow profile override must promote prepared session when task has no primary")
	}
	if got := updated.Metadata[models.SessionMetaKeyCreatedBy]; got != models.SessionCreatedByWorkflowSwitch {
		t.Fatalf("created_by metadata = %v, want %q", got, models.SessionCreatedByWorkflowSwitch)
	}
	if launchedProfile != "profile-b" {
		t.Fatalf("launched profile = %q, want profile-b", launchedProfile)
	}
}

// TestStartCreatedSession_EmptyProfileFallsBackToWorkflowDefault pins the bug
// where an auto-started session prepared without an agent_profile_id (e.g. a
// task imported from Linear whose metadata agent_profile_id is empty) recorded
// the auto-start step prompt but never launched the agent. StartCreatedSession
// aborted with "agent_profile_id is required" because the required-profile
// guard ran before the workflow-default resolution. The launch must instead
// inherit the workflow's default agent profile and persist it on the session.
func TestStartCreatedSession_EmptyProfileFallsBackToWorkflowDefault(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateCreated)
	// executors_running lets LaunchPreparedSession take the existing-workspace
	// fast path instead of launching a real agent.
	seedExecutorRunning(t, repo, "session1", "task1", "exec-1")

	// Bind the task to a workflow step whose workflow defines a default agent
	// profile, with no step-level override — the Auto Dispatch Workflow shape.
	dbTask, err := repo.GetTask(ctx, "task1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	dbTask.WorkflowStepID = "step1"
	if err := repo.UpdateTask(ctx, dbTask); err != nil {
		t.Fatalf("update task: %v", err)
	}

	stepGetter := newMockStepGetter()
	stepGetter.steps["step1"] = &wfmodels.WorkflowStep{ID: "step1", WorkflowID: "wf1"}
	stepGetter.workflowAgentProfileID = "wf-default-profile"

	taskRepo := newMockTaskRepo()
	taskRepo.tasks["task1"] = &v1.Task{ID: "task1", Title: "Test Task", State: v1.TaskStateInProgress}
	agentMgr := &mockAgentManager{repoForExecutionLookup: repo}
	svc := createTestServiceWithScheduler(repo, stepGetter, taskRepo, agentMgr)
	svc.messageCreator = &mockMessageCreator{}

	// The auto-start path passes the session's stored profile, which is empty
	// here. The previous code aborted with "agent_profile_id is required".
	_, err = svc.StartCreatedSession(ctx, "task1", "session1", "", "Do the work", true, false, true, nil)
	if err != nil {
		t.Fatalf("StartCreatedSession must resolve the workflow default for an empty profile, got error: %v", err)
	}

	// The resolved workflow default must be persisted on the session so the
	// agent actually launches under it (and the UI shows the right agent).
	got, err := repo.GetTaskSession(ctx, "session1")
	if err != nil {
		t.Fatalf("reload session: %v", err)
	}
	if got.AgentProfileID != "wf-default-profile" {
		t.Errorf("expected session to inherit workflow default %q, got %q", "wf-default-profile", got.AgentProfileID)
	}
}

// --- recordInitialMessage ---

// mockMessageCreator implements MessageCreator for testing.
// Only CreateUserMessage is tracked; all other methods are no-op stubs.
type mockMessageCreator struct {
	userMessages    []mockUserMessage
	sessionMessages []mockSessionMessage
	userMessageErr  error
}

type mockUserMessage struct {
	taskID, content, sessionID, turnID string
	metadata                           map[string]interface{}
}

type mockSessionMessage struct {
	taskID, content, sessionID, messageType, turnID string
	metadata                                        map[string]interface{}
	requestsInput                                   bool
}

func (m *mockMessageCreator) CreateUserMessage(_ context.Context, taskID, content, sessionID, turnID string, metadata map[string]interface{}) error {
	if m.userMessageErr != nil {
		return m.userMessageErr
	}
	m.userMessages = append(m.userMessages, mockUserMessage{taskID, content, sessionID, turnID, metadata})
	return nil
}

func (m *mockMessageCreator) CreateAgentMessage(context.Context, string, string, string, string) error {
	return nil
}

func (m *mockMessageCreator) CreateToolCallMessage(context.Context, string, string, string, string, string, string, string, *streams.NormalizedPayload) error {
	return nil
}

func (m *mockMessageCreator) UpdateToolCallMessage(context.Context, string, string, string, string, string, string, string, string, string, *streams.NormalizedPayload) error {
	return nil
}

func (m *mockMessageCreator) CreateSessionMessage(_ context.Context, taskID, content, sessionID, messageType, turnID string, metadata map[string]interface{}, requestsInput bool) error {
	m.sessionMessages = append(m.sessionMessages, mockSessionMessage{
		taskID:        taskID,
		content:       content,
		sessionID:     sessionID,
		messageType:   messageType,
		turnID:        turnID,
		metadata:      metadata,
		requestsInput: requestsInput,
	})
	return nil
}

func (m *mockMessageCreator) CreatePermissionRequestMessage(context.Context, string, string, string, string, string, string, []map[string]interface{}, string, map[string]interface{}) (string, error) {
	return "", nil
}

func (m *mockMessageCreator) UpdatePermissionMessage(context.Context, string, string, models.PermissionStatus) error {
	return nil
}

func (m *mockMessageCreator) CreateAgentMessageStreaming(context.Context, string, string, string, string, string) error {
	return nil
}

func (m *mockMessageCreator) AppendAgentMessage(context.Context, string, string) error {
	return nil
}

func (m *mockMessageCreator) CreateThinkingMessageStreaming(context.Context, string, string, string, string, string) error {
	return nil
}

func (m *mockMessageCreator) AppendThinkingMessage(context.Context, string, string) error {
	return nil
}
func (m *mockMessageCreator) InvalidateModelCache(string) {}

// --- backfillInitialUserMessageIfMissing ---

func TestBackfillInitialUserMessageIfMissing_RecordsWhenSessionEmpty(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateFailed)

	mc := &mockMessageCreator{}
	svc := createTestService(repo, newMockStepGetter(), newMockTaskRepo())
	svc.messageCreator = mc

	// Session has zero messages — backfill should record the prompt.
	svc.backfillInitialUserMessageIfMissing(ctx, "task1", "session1", "original prompt")

	if len(mc.userMessages) != 1 {
		t.Fatalf("expected 1 user message recorded, got %d", len(mc.userMessages))
	}
	if mc.userMessages[0].content != "original prompt" {
		t.Errorf("content = %q, want %q", mc.userMessages[0].content, "original prompt")
	}
}

func TestBackfillInitialUserMessageIfMissing_SkipsWhenUserMessageExists(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateFailed)

	// Seed an existing user message — the backfill must be a no-op so a
	// successful prior launch isn't duplicated on a subsequent resume.
	if err := repo.CreateTurn(ctx, &models.Turn{ID: "turn1", TaskSessionID: "session1", TaskID: "task1", StartedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("create turn: %v", err)
	}
	if err := repo.CreateMessage(ctx, &models.Message{
		ID:            "msg1",
		TaskSessionID: "session1",
		TaskID:        "task1",
		TurnID:        "turn1",
		AuthorType:    models.MessageAuthorUser,
		Content:       "user already sent this",
		CreatedAt:     time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create message: %v", err)
	}

	mc := &mockMessageCreator{}
	svc := createTestService(repo, newMockStepGetter(), newMockTaskRepo())
	svc.messageCreator = mc

	svc.backfillInitialUserMessageIfMissing(ctx, "task1", "session1", "would be a duplicate")

	if len(mc.userMessages) != 0 {
		t.Fatalf("expected no user message recorded (one already exists), got %d", len(mc.userMessages))
	}
}

// TestBackfillInitialUserMessageIfMissing_SkipsWhenAgentMessageExists covers
// the regression where a partial prior run produced agent output but never
// recorded the initial user message. Recording the user message now with
// CreatedAt=time.Now() would place it at the bottom of the chat (after the
// agent messages), which is worse than leaving the chat alone.
func TestBackfillInitialUserMessageIfMissing_SkipsWhenAgentMessageExists(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateFailed)

	if err := repo.CreateTurn(ctx, &models.Turn{ID: "turn1", TaskSessionID: "session1", TaskID: "task1", StartedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("create turn: %v", err)
	}
	if err := repo.CreateMessage(ctx, &models.Message{
		ID:            "agent-msg-1",
		TaskSessionID: "session1",
		TaskID:        "task1",
		TurnID:        "turn1",
		AuthorType:    models.MessageAuthorAgent,
		Content:       "agent partial output from a prior run",
		CreatedAt:     time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create agent message: %v", err)
	}

	mc := &mockMessageCreator{}
	svc := createTestService(repo, newMockStepGetter(), newMockTaskRepo())
	svc.messageCreator = mc

	svc.backfillInitialUserMessageIfMissing(ctx, "task1", "session1", "the original prompt")

	if len(mc.userMessages) != 0 {
		t.Fatalf("expected no backfill when agent messages exist, got %d", len(mc.userMessages))
	}
}

func TestBackfillInitialUserMessageIfMissing_SkipsEmptyPrompt(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateFailed)

	mc := &mockMessageCreator{}
	svc := createTestService(repo, newMockStepGetter(), newMockTaskRepo())
	svc.messageCreator = mc

	svc.backfillInitialUserMessageIfMissing(ctx, "task1", "session1", "")

	if len(mc.userMessages) != 0 {
		t.Fatalf("expected no user message for empty prompt, got %d", len(mc.userMessages))
	}
}

func TestRecordInitialMessage_DoesNotChangeSessionState(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateStarting)

	mc := &mockMessageCreator{}
	svc := createTestService(repo, newMockStepGetter(), newMockTaskRepo())
	svc.messageCreator = mc

	svc.recordInitialMessage(ctx, "task1", "session1", "hello world", false, false, nil)

	// Session state must remain STARTING — recordInitialMessage should not modify state.
	session, err := repo.GetTaskSession(ctx, "session1")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	if session.State != models.TaskSessionStateStarting {
		t.Fatalf("expected session state %q, got %q", models.TaskSessionStateStarting, session.State)
	}

	// User message should have been created.
	if len(mc.userMessages) != 1 {
		t.Fatalf("expected 1 user message, got %d", len(mc.userMessages))
	}
	if mc.userMessages[0].content != "hello world" {
		t.Fatalf("expected message content %q, got %q", "hello world", mc.userMessages[0].content)
	}
}

func TestPostLaunchCreated_SkipMessage_DoesNotChangeSessionState(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateStarting)

	svc := createTestService(repo, newMockStepGetter(), newMockTaskRepo())

	svc.postLaunchCreated(ctx, "task1", "session1", "prompt", true, false, false, nil)

	// Session state must remain STARTING when skipMessage=true.
	session, err := repo.GetTaskSession(ctx, "session1")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	if session.State != models.TaskSessionStateStarting {
		t.Fatalf("expected session state %q, got %q", models.TaskSessionStateStarting, session.State)
	}
}

func TestPostLaunchCreated_WithMessage_DoesNotChangeSessionState(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateStarting)

	mc := &mockMessageCreator{}
	svc := createTestService(repo, newMockStepGetter(), newMockTaskRepo())
	svc.messageCreator = mc

	svc.postLaunchCreated(ctx, "task1", "session1", "hello", false, false, false, nil)

	// Session state must remain STARTING — postLaunchCreated delegates to
	// recordInitialMessage which only creates the message.
	session, err := repo.GetTaskSession(ctx, "session1")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	if session.State != models.TaskSessionStateStarting {
		t.Fatalf("expected session state %q, got %q", models.TaskSessionStateStarting, session.State)
	}

	if len(mc.userMessages) != 1 {
		t.Fatalf("expected 1 user message, got %d", len(mc.userMessages))
	}
}

func TestPostLaunchCreated_AutoStart_SetsMetadata(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateStarting)

	mc := &mockMessageCreator{}
	svc := createTestService(repo, newMockStepGetter(), newMockTaskRepo())
	svc.messageCreator = mc

	// autoStart=true should land an `auto_start: true` tag on the
	// recorded user message so HasUserAuthoredMessage skips it. This
	// asserts the metadata wiring in recordInitialMessage directly —
	// the broader behavior is tested in cmd/kandev TestHasUserAuthoredMessage.
	svc.postLaunchCreated(ctx, "task1", "session1", "auto-started by workflow", false, false, true, nil)

	if len(mc.userMessages) != 1 {
		t.Fatalf("expected 1 user message, got %d", len(mc.userMessages))
	}
	if mc.userMessages[0].metadata["auto_start"] != true {
		t.Fatalf("expected auto_start=true in metadata, got %v", mc.userMessages[0].metadata)
	}
}

func TestPostLaunchCreated_PlanMode_SetsMetadata(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateStarting)

	mc := &mockMessageCreator{}
	svc := createTestService(repo, newMockStepGetter(), newMockTaskRepo())
	svc.messageCreator = mc

	svc.postLaunchCreated(ctx, "task1", "session1", "plan this", false, true, false, nil)

	// User message should have plan_mode metadata.
	if len(mc.userMessages) != 1 {
		t.Fatalf("expected 1 user message, got %d", len(mc.userMessages))
	}
	if mc.userMessages[0].metadata["plan_mode"] != true {
		t.Fatalf("expected plan_mode=true in metadata, got %v", mc.userMessages[0].metadata)
	}

	// Session metadata should contain plan_mode.
	session, err := repo.GetTaskSession(ctx, "session1")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	if session.Metadata == nil {
		t.Fatal("expected session metadata to be set")
	}
	if session.Metadata["plan_mode"] != true {
		t.Fatalf("expected plan_mode=true in session metadata, got %v", session.Metadata["plan_mode"])
	}
}

// --- StartCreatedSession: Kandev system prompt wrap on first launch ---

// TestStartCreatedSession_WrapsFirstPromptWithKandevSystemBlock verifies that
// the recorded user message persists the <kandev-system> wrap that the
// orchestrator now injects in startTask / StartCreatedSession. The wrap must
// be in the raw row so the chat UI can show it under "Show formatted" and the
// agent CLI's first ACP prompt includes the MCP tools list and task/session
// IDs. Regression guard for the case the user reported: "tasks I create from
// the kanban mode don't have the kandev system prompt."
func TestStartCreatedSession_WrapsFirstPromptWithKandevSystemBlock(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateCreated)
	// Seed executors_running so LaunchPreparedSession takes the fast path
	// (startAgentOnExistingWorkspace) and never reaches the real LaunchAgent.
	seedExecutorRunning(t, repo, "session1", "task1", "exec-1")

	taskRepo := newMockTaskRepo()
	taskRepo.tasks["task1"] = &v1.Task{
		ID:          "task1",
		Title:       "Test Task",
		Description: "Original task description",
		State:       v1.TaskStateInProgress,
	}
	agentMgr := &mockAgentManager{repoForExecutionLookup: repo}
	svc := createTestServiceWithScheduler(repo, newMockStepGetter(), taskRepo, agentMgr)
	mc := &mockMessageCreator{}
	svc.messageCreator = mc

	_, err := svc.StartCreatedSession(ctx, "task1", "session1", "profile1", "Build me a feature", false, false, false, nil)
	if err != nil {
		t.Fatalf("StartCreatedSession failed: %v", err)
	}

	if len(mc.userMessages) != 1 {
		t.Fatalf("expected 1 user message recorded, got %d", len(mc.userMessages))
	}
	content := mc.userMessages[0].content

	// The wrap is the outermost layer; the user's typed text must still be inside it.
	if !strings.Contains(content, "<kandev-system>") {
		t.Errorf("expected <kandev-system> opening tag in recorded content, got %q", content)
	}
	if !strings.Contains(content, "</kandev-system>") {
		t.Errorf("expected </kandev-system> closing tag in recorded content, got %q", content)
	}
	if !strings.Contains(content, "Build me a feature") {
		t.Errorf("expected user text preserved in recorded content, got %q", content)
	}
	// The wrap must carry the task and session IDs so the agent can call the
	// kandev MCP tools without re-discovering its own identifiers.
	if !strings.Contains(content, "Kandev Task ID: task1") {
		t.Errorf("expected Kandev Task ID in wrap, got %q", content)
	}
	if !strings.Contains(content, "Kandev Session ID: session1") {
		t.Errorf("expected Kandev Session ID in wrap, got %q", content)
	}
	// The MCP tool list is the whole point of the wrap — guard a representative one.
	if !strings.Contains(content, "ask_user_question_pcraft") {
		t.Errorf("expected ask_user_question_pcraft tool in wrap, got %q", content)
	}
}

// TestStartCreatedSession_DoesNotDoubleWrapPreWrappedPrompt verifies the
// idempotency guard on the orchestrator's wrap step. Upstream call sites
// (wsAddMessage on CREATED sessions, recordAutoStartMessage) wrap before
// recording the user message so the DB row carries the <kandev-system>
// block. When the wrapped content is later passed through StartCreatedSession,
// the orchestrator must NOT wrap it a second time — otherwise the agent
// receives nested system blocks and the strip pipeline behaves unpredictably.
func TestStartCreatedSession_DoesNotDoubleWrapPreWrappedPrompt(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateCreated)
	seedExecutorRunning(t, repo, "session1", "task1", "exec-1")

	taskRepo := newMockTaskRepo()
	taskRepo.tasks["task1"] = &v1.Task{
		ID:    "task1",
		Title: "Test Task",
		State: v1.TaskStateInProgress,
	}
	agentMgr := &mockAgentManager{repoForExecutionLookup: repo}
	svc := createTestServiceWithScheduler(repo, newMockStepGetter(), taskRepo, agentMgr)
	mc := &mockMessageCreator{}
	svc.messageCreator = mc

	// Simulate an upstream caller (e.g. wsAddMessage) that has already wrapped.
	preWrapped := sysprompt.InjectKandevContext("task1", "session1", "Build me a feature", false)

	_, err := svc.StartCreatedSession(ctx, "task1", "session1", "profile1", preWrapped, false, false, false, nil)
	if err != nil {
		t.Fatalf("StartCreatedSession failed: %v", err)
	}

	if len(mc.userMessages) != 1 {
		t.Fatalf("expected 1 user message recorded, got %d", len(mc.userMessages))
	}
	content := mc.userMessages[0].content

	// Exactly one opening tag and one closing tag — not nested.
	openCount := strings.Count(content, "<kandev-system>")
	closeCount := strings.Count(content, "</kandev-system>")
	if openCount != 1 {
		t.Errorf("expected exactly 1 <kandev-system> tag, got %d in %q", openCount, content)
	}
	if closeCount != 1 {
		t.Errorf("expected exactly 1 </kandev-system> tag, got %d in %q", closeCount, content)
	}
	// The user's text is preserved.
	if !strings.Contains(content, "Build me a feature") {
		t.Errorf("expected user text preserved, got %q", content)
	}
}

// TestStartCreatedSession_EmptyPromptSkipsWrap verifies the orchestrator does
// not synthesize a <kandev-system>-only message when the user has nothing to
// say yet. recordInitialMessage already skips empty prompts, but wrapping
// "" would defeat that guard and pollute the chat with a tag-only row.
func TestStartCreatedSession_EmptyPromptSkipsWrap(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateCreated)
	seedExecutorRunning(t, repo, "session1", "task1", "exec-1")

	taskRepo := newMockTaskRepo()
	// No description on the task and no prompt from the caller — startTask's
	// `effectivePrompt == ""` branch must short-circuit before InjectKandevContext.
	taskRepo.tasks["task1"] = &v1.Task{ID: "task1", Title: "Empty", State: v1.TaskStateInProgress}
	agentMgr := &mockAgentManager{repoForExecutionLookup: repo}
	svc := createTestServiceWithScheduler(repo, newMockStepGetter(), taskRepo, agentMgr)
	mc := &mockMessageCreator{}
	svc.messageCreator = mc

	_, err := svc.StartCreatedSession(ctx, "task1", "session1", "profile1", "", false, false, false, nil)
	if err != nil {
		t.Fatalf("StartCreatedSession failed: %v", err)
	}

	// No user message should be recorded — wrapping an empty prompt would
	// produce a tag-only row.
	if len(mc.userMessages) != 0 {
		t.Fatalf("expected 0 user messages for empty prompt, got %d (content=%q)",
			len(mc.userMessages), mc.userMessages[0].content)
	}
}

// --- ResumeTaskSession ---

func TestResumeTaskSession_WrongTask(t *testing.T) {
	repo := setupTestRepo(t)
	svc := createTestService(repo, newMockStepGetter(), newMockTaskRepo())

	seedTaskAndSession(t, repo, "task-other", "session1", models.TaskSessionStateWaitingForInput)

	_, err := svc.ResumeTaskSession(context.Background(), "task1", "session1")
	if err == nil {
		t.Fatal("expected error when session does not belong to task")
	}
}

func TestResumeTaskSession_NotResumable(t *testing.T) {
	repo := setupTestRepo(t)
	svc := createTestService(repo, newMockStepGetter(), newMockTaskRepo())

	// Session exists and belongs to task, but there is no ExecutorRunning record
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateWaitingForInput)

	_, err := svc.ResumeTaskSession(context.Background(), "task1", "session1")
	if err == nil {
		t.Fatal("expected error when no executor running record exists")
	}
}

func TestResumeTaskSession_ArchivedTaskSkipsFailedState(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	taskRepo := newMockTaskRepo()
	agentMgr := &mockAgentManager{repoForExecutionLookup: repo}
	svc := createTestServiceWithAgent(repo, newMockStepGetter(), taskRepo, agentMgr)
	svc.executor = executor.NewExecutor(agentMgr, repo, testLogger(), executor.ExecutorConfig{})

	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateWaitingForInput)

	// Archive the task after seeding
	if err := repo.ArchiveTask(ctx, "task1"); err != nil {
		t.Fatalf("failed to archive task: %v", err)
	}

	// Insert executor running record so we pass the "not resumable" check
	now := time.Now().UTC()
	_ = repo.UpsertExecutorRunning(ctx, &models.ExecutorRunning{
		ID: "er1", SessionID: "session1", TaskID: "task1",
		CreatedAt: now, UpdatedAt: now,
	})

	_, err := svc.ResumeTaskSession(ctx, "task1", "session1")
	if !errors.Is(err, executor.ErrTaskArchived) {
		t.Fatalf("expected ErrTaskArchived, got: %v", err)
	}

	// Task state should NOT have been updated to FAILED
	if _, ok := taskRepo.updatedStates["task1"]; ok {
		t.Error("task state should not be updated when task is archived")
	}
}

func TestResumeTaskSession_ArchivedDuringLaunch(t *testing.T) {
	// Simulates the race: task is NOT archived when the executor checks,
	// but LaunchAgent fails (archive's async cleanup killed the agent),
	// and by the time the error path re-reads the task it IS archived.
	ctx := context.Background()
	repo := setupTestRepo(t)
	taskRepo := newMockTaskRepo()
	agentMgr := &mockAgentManager{
		launchAgentFunc: func(_ context.Context, _ *executor.LaunchAgentRequest) (*executor.LaunchAgentResponse, error) {
			// Simulate archive completing while launch is in progress:
			// archive the task, then fail the launch (as if async cleanup killed the process).
			_ = repo.ArchiveTask(ctx, "task1")
			return nil, fmt.Errorf("connection refused")
		},
	}
	svc := createTestServiceWithAgent(repo, newMockStepGetter(), taskRepo, agentMgr)
	svc.executor = executor.NewExecutor(agentMgr, repo, testLogger(), executor.ExecutorConfig{})

	now := time.Now().UTC()
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateWaitingForInput)

	// Set agent profile ID so the executor doesn't reject the session early
	session, _ := repo.GetTaskSession(ctx, "session1")
	session.AgentProfileID = "profile-1"
	_ = repo.UpdateTaskSession(ctx, session)

	_ = repo.UpsertExecutorRunning(ctx, &models.ExecutorRunning{
		ID: "er1", SessionID: "session1", TaskID: "task1",
		CreatedAt: now, UpdatedAt: now,
	})

	_, err := svc.ResumeTaskSession(ctx, "task1", "session1")
	if !errors.Is(err, executor.ErrTaskArchived) {
		t.Fatalf("expected ErrTaskArchived, got: %v", err)
	}

	// Task state should NOT have been updated to FAILED
	if _, ok := taskRepo.updatedStates["task1"]; ok {
		t.Error("task state should not be updated when task is archived during launch")
	}
}

// TestResumeTaskSession_FailedKeepsResumeToken verifies that resuming a FAILED
// session preserves the ACP resume token so the relaunched agent restores the
// prior conversation via ACP session/load (for native-resume agents).
// Regression test for the "Resume blocked by stale state" bug where FAILED sessions
// couldn't be restarted at all; the fix also changes policy to keep the token
// (previously it was cleared to force a fresh agent).
func TestResumeTaskSession_FailedKeepsResumeToken(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	taskRepo := newMockTaskRepo()

	// Agent launch succeeds so the resume path does not unwind and mark the task
	// FAILED, which would exercise a separate state-mutation code path.
	startAgentProcessCalled := false
	agentMgr := &sessionUpdatingAgentManager{
		mockAgentManager: &mockAgentManager{
			launchAgentFunc: func(_ context.Context, _ *executor.LaunchAgentRequest) (*executor.LaunchAgentResponse, error) {
				return &executor.LaunchAgentResponse{
					AgentExecutionID: "exec-new",
					Status:           v1.AgentStatusStarting,
				}, nil
			},
		},
		repo:          repo,
		sessionID:     "session1",
		taskID:        "task1",
		onStartCalled: &startAgentProcessCalled,
	}
	svc := createTestServiceWithAgent(repo, newMockStepGetter(), taskRepo, agentMgr)
	svc.executor = executor.NewExecutor(agentMgr, repo, testLogger(), executor.ExecutorConfig{})

	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateFailed)
	session, _ := repo.GetTaskSession(ctx, "session1")
	session.AgentProfileID = "profile-1"
	_ = repo.UpdateTaskSession(ctx, session)

	now := time.Now().UTC()
	_ = repo.UpsertExecutorRunning(ctx, &models.ExecutorRunning{
		ID: "er1", SessionID: "session1", TaskID: "task1",
		ResumeToken: "acp-session-xyz",
		Resumable:   true,
		CreatedAt:   now, UpdatedAt: now,
	})

	if _, err := svc.ResumeTaskSession(ctx, "task1", "session1"); err != nil {
		t.Fatalf("ResumeTaskSession on FAILED session returned: %v", err)
	}

	er, err := repo.GetExecutorRunningBySessionID(ctx, "session1")
	if err != nil || er == nil {
		t.Fatalf("ExecutorRunning lookup failed: %v (nil=%v)", err, er == nil)
	}
	if er.ResumeToken != "acp-session-xyz" {
		t.Errorf("expected resume token to be preserved on FAILED resume, got %q", er.ResumeToken)
	}
}

// --- CompleteTask ---

// --- Error Classification Functions ---

func TestErrorClassificationFunctions(t *testing.T) {
	t.Run("isAgentPromptInProgressError", func(t *testing.T) {
		tests := []struct {
			name string
			err  error
			want bool
		}{
			{"nil error", nil, false},
			{"unrelated error", errors.New("something else"), false},
			{"exact match", ErrAgentPromptInProgress, true},
			{"wrapped error", fmt.Errorf("outer: %w", ErrAgentPromptInProgress), true},
			{"untyped string match no longer accepted", errors.New("prefix: agent is currently processing a prompt, try later"), false},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				if got := isAgentPromptInProgressError(tc.err); got != tc.want {
					t.Errorf("isAgentPromptInProgressError(%v) = %v, want %v", tc.err, got, tc.want)
				}
			})
		}
	})

	t.Run("isSessionResetInProgressError", func(t *testing.T) {
		tests := []struct {
			name string
			err  error
			want bool
		}{
			{"nil error", nil, false},
			{"unrelated error", errors.New("something else"), false},
			{"exact match", ErrSessionResetInProgress, true},
			{"wrapped error", fmt.Errorf("outer: %w", ErrSessionResetInProgress), true},
			{"untyped string match no longer accepted", errors.New("prefix: session reset in progress, please wait"), false},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				if got := isSessionResetInProgressError(tc.err); got != tc.want {
					t.Errorf("isSessionResetInProgressError(%v) = %v, want %v", tc.err, got, tc.want)
				}
			})
		}
	})

	t.Run("isAgentAlreadyRunningError", func(t *testing.T) {
		tests := []struct {
			name string
			err  error
			want bool
		}{
			{"nil error", nil, false},
			{"unrelated error", errors.New("something else"), false},
			{"lifecycle manager error", fmt.Errorf("%w: session %q (execution: %s)", lifecycle.ErrAgentAlreadyRunning, "s1", "exec-1"), true},
			{"wrapped error", fmt.Errorf("failed to resume session: %w", fmt.Errorf("%w: session %q (execution: %s)", lifecycle.ErrAgentAlreadyRunning, "s1", "exec-1")), true},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				if got := isAgentAlreadyRunningError(tc.err); got != tc.want {
					t.Errorf("isAgentAlreadyRunningError(%v) = %v, want %v", tc.err, got, tc.want)
				}
			})
		}
	})

	t.Run("isTransientPromptError", func(t *testing.T) {
		tests := []struct {
			name string
			err  error
			want bool
		}{
			{"nil error", nil, false},
			{"unrelated error", errors.New("something else"), false},
			{"agent stream disconnected", errors.New("agent stream disconnected: read tcp"), true},
			{"use of closed network connection", errors.New("write: use of closed network connection"), true},
			{"case insensitive match", errors.New("Agent Stream Disconnected: EOF"), true},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				if got := isTransientPromptError(tc.err); got != tc.want {
					t.Errorf("isTransientPromptError(%v) = %v, want %v", tc.err, got, tc.want)
				}
			})
		}
	})
}

// --- GetTaskSessionStatus ---

// TestGetTaskSessionStatus_DoesNotHealOnMismatchedExecution was removed.
// The pre-refactor heal check skipped healing when session.AgentExecutionID and
// running.AgentExecutionID disagreed — a band-aid for the very divergence bug
// this PR fixes structurally. With executors_running as the single source of
// truth (lifecycle-owned, persisted in lockstep with executionStore.Add), the
// mismatch this test simulated cannot occur, and the band-aid was removed
// (see shouldHealStuckStartingSession in task_operations.go).

// TestGetTaskSessionStatus_UsesTaskEnvironmentBranchForDocker was removed:
// it tested Docker executor-specific branch retrieval. Docker executors have
// been removed.

// --- ReconcileSessionsOnStartup ---

func TestReconcileSessionsOnStartup(t *testing.T) {
	t.Run("terminal_session_cleaned_up", func(t *testing.T) {
		repo := setupTestRepo(t)
		ctx := context.Background()
		now := time.Now().UTC()

		seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateCompleted)

		err := repo.UpsertExecutorRunning(ctx, &models.ExecutorRunning{
			ID:               "er1",
			SessionID:        "session1",
			TaskID:           "task1",
			AgentExecutionID: "exec-terminal",
			CreatedAt:        now,
			UpdatedAt:        now,
		})
		if err != nil {
			t.Fatalf("failed to upsert executor running: %v", err)
		}

		agentMgr := &mockAgentManager{}
		svc := createTestServiceWithAgent(repo, newMockStepGetter(), newMockTaskRepo(), agentMgr)
		svc.reconcileSessionsOnStartup(ctx)

		_, err = repo.GetExecutorRunningBySessionID(ctx, "session1")
		if err == nil {
			t.Fatal("expected ExecutorRunning record to be deleted for terminal session")
		}
		agentMgr.mu.Lock()
		stopCalls := append([]stopAgentCall(nil), agentMgr.stopAgentWithReasonArgs...)
		agentMgr.mu.Unlock()
		if len(stopCalls) != 1 {
			t.Fatalf("expected one StopAgentWithReason call, got %d", len(stopCalls))
		}
		if stopCalls[0] != (stopAgentCall{
			ExecutionID: "exec-terminal",
			Reason:      "startup terminal session cleanup",
			Force:       true,
		}) {
			t.Fatalf("unexpected StopAgentWithReason call: %#v", stopCalls[0])
		}
	})

	t.Run("terminal_session_stop_failure_preserves_executor_row", func(t *testing.T) {
		repo := setupTestRepo(t)
		ctx := context.Background()
		now := time.Now().UTC()

		seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateCompleted)

		err := repo.UpsertExecutorRunning(ctx, &models.ExecutorRunning{
			ID:               "er1",
			SessionID:        "session1",
			TaskID:           "task1",
			AgentExecutionID: "exec-terminal",
			CreatedAt:        now,
			UpdatedAt:        now,
		})
		if err != nil {
			t.Fatalf("failed to upsert executor running: %v", err)
		}

		agentMgr := &mockAgentManager{stopAgentWithReasonErr: errors.New("runtime still running")}
		svc := createTestServiceWithAgent(repo, newMockStepGetter(), newMockTaskRepo(), agentMgr)
		svc.reconcileSessionsOnStartup(ctx)

		running, err := repo.GetExecutorRunningBySessionID(ctx, "session1")
		if err != nil {
			t.Fatalf("expected ExecutorRunning record to be preserved after stop failure: %v", err)
		}
		if running.AgentExecutionID != "exec-terminal" {
			t.Fatalf("expected execution ID to be preserved, got %q", running.AgentExecutionID)
		}
		agentMgr.mu.Lock()
		stopCalls := append([]stopAgentCall(nil), agentMgr.stopAgentWithReasonArgs...)
		agentMgr.mu.Unlock()
		if len(stopCalls) != 1 {
			t.Fatalf("expected one StopAgentWithReason call, got %d", len(stopCalls))
		}
		if stopCalls[0] != (stopAgentCall{
			ExecutionID: "exec-terminal",
			Reason:      "startup terminal session cleanup",
			Force:       true,
		}) {
			t.Fatalf("unexpected StopAgentWithReason call: %#v", stopCalls[0])
		}
	})

	t.Run("missing_session_runtime_cleaned_up", func(t *testing.T) {
		repo := setupTestRepo(t)
		ctx := context.Background()
		now := time.Now().UTC()

		err := repo.UpsertExecutorRunning(ctx, &models.ExecutorRunning{
			ID:               "er1",
			SessionID:        "session-deleted",
			TaskID:           "task-deleted",
			AgentExecutionID: "exec-deleted",
			CreatedAt:        now,
			UpdatedAt:        now,
		})
		if err != nil {
			t.Fatalf("failed to upsert executor running: %v", err)
		}

		agentMgr := &mockAgentManager{}
		svc := createTestServiceWithAgent(repo, newMockStepGetter(), newMockTaskRepo(), agentMgr)
		svc.reconcileSessionsOnStartup(ctx)

		_, err = repo.GetExecutorRunningBySessionID(ctx, "session-deleted")
		if err == nil {
			t.Fatal("expected ExecutorRunning record to be deleted for missing session after stop")
		}
		agentMgr.mu.Lock()
		stopCalls := append([]stopAgentCall(nil), agentMgr.stopAgentWithReasonArgs...)
		agentMgr.mu.Unlock()
		if len(stopCalls) != 1 {
			t.Fatalf("expected one StopAgentWithReason call, got %d", len(stopCalls))
		}
		if stopCalls[0] != (stopAgentCall{
			ExecutionID: "exec-deleted",
			Reason:      "startup missing session cleanup",
			Force:       true,
		}) {
			t.Fatalf("unexpected StopAgentWithReason call: %#v", stopCalls[0])
		}
	})

	t.Run("missing_session_stop_failure_preserves_executor_row", func(t *testing.T) {
		repo := setupTestRepo(t)
		ctx := context.Background()
		now := time.Now().UTC()

		err := repo.UpsertExecutorRunning(ctx, &models.ExecutorRunning{
			ID:               "er1",
			SessionID:        "session-deleted",
			TaskID:           "task-deleted",
			AgentExecutionID: "exec-deleted",
			CreatedAt:        now,
			UpdatedAt:        now,
		})
		if err != nil {
			t.Fatalf("failed to upsert executor running: %v", err)
		}

		agentMgr := &mockAgentManager{stopAgentWithReasonErr: errors.New("runtime still running")}
		svc := createTestServiceWithAgent(repo, newMockStepGetter(), newMockTaskRepo(), agentMgr)
		svc.reconcileSessionsOnStartup(ctx)

		running, err := repo.GetExecutorRunningBySessionID(ctx, "session-deleted")
		if err != nil {
			t.Fatalf("expected ExecutorRunning record to be preserved after stop failure: %v", err)
		}
		if running.AgentExecutionID != "exec-deleted" {
			t.Fatalf("expected execution ID to be preserved, got %q", running.AgentExecutionID)
		}
		agentMgr.mu.Lock()
		stopCalls := append([]stopAgentCall(nil), agentMgr.stopAgentWithReasonArgs...)
		agentMgr.mu.Unlock()
		if len(stopCalls) != 1 {
			t.Fatalf("expected one StopAgentWithReason call, got %d", len(stopCalls))
		}
		if stopCalls[0] != (stopAgentCall{
			ExecutionID: "exec-deleted",
			Reason:      "startup missing session cleanup",
			Force:       true,
		}) {
			t.Fatalf("unexpected StopAgentWithReason call: %#v", stopCalls[0])
		}
	})

	t.Run("active_session_set_to_waiting", func(t *testing.T) {
		repo := setupTestRepo(t)
		ctx := context.Background()
		now := time.Now().UTC()

		seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateRunning)

		err := repo.UpsertExecutorRunning(ctx, &models.ExecutorRunning{
			ID:        "er1",
			SessionID: "session1",
			TaskID:    "task1",
			CreatedAt: now,
			UpdatedAt: now,
		})
		if err != nil {
			t.Fatalf("failed to upsert executor running: %v", err)
		}

		svc := createTestServiceWithAgent(repo, newMockStepGetter(), newMockTaskRepo(), &mockAgentManager{})
		svc.reconcileSessionsOnStartup(ctx)

		session, err := repo.GetTaskSession(ctx, "session1")
		if err != nil {
			t.Fatalf("failed to get session: %v", err)
		}
		if session.State != models.TaskSessionStateWaitingForInput {
			t.Fatalf("expected session state %q, got %q", models.TaskSessionStateWaitingForInput, session.State)
		}

		// ExecutorRunning should be preserved for lazy resume
		_, err = repo.GetExecutorRunningBySessionID(ctx, "session1")
		if err != nil {
			t.Fatalf("expected ExecutorRunning record to be preserved, got error: %v", err)
		}
	})

	t.Run("failed_session_without_resume_token_stops_runtime_before_cleanup", func(t *testing.T) {
		repo := setupTestRepo(t)
		ctx := context.Background()
		now := time.Now().UTC()

		seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateFailed)

		err := repo.UpsertExecutorRunning(ctx, &models.ExecutorRunning{
			ID:               "er1",
			SessionID:        "session1",
			TaskID:           "task1",
			AgentExecutionID: "exec-failed",
			CreatedAt:        now,
			UpdatedAt:        now,
		})
		if err != nil {
			t.Fatalf("failed to upsert executor running: %v", err)
		}

		agentMgr := &mockAgentManager{}
		svc := createTestServiceWithAgent(repo, newMockStepGetter(), newMockTaskRepo(), agentMgr)
		svc.reconcileSessionsOnStartup(ctx)

		_, err = repo.GetExecutorRunningBySessionID(ctx, "session1")
		if err == nil {
			t.Fatal("expected ExecutorRunning record to be deleted for failed session after stop")
		}
		agentMgr.mu.Lock()
		stopCalls := append([]stopAgentCall(nil), agentMgr.stopAgentWithReasonArgs...)
		agentMgr.mu.Unlock()
		if len(stopCalls) != 1 {
			t.Fatalf("expected one StopAgentWithReason call, got %d", len(stopCalls))
		}
		if stopCalls[0] != (stopAgentCall{
			ExecutionID: "exec-failed",
			Reason:      "startup failed session cleanup",
			Force:       true,
		}) {
			t.Fatalf("unexpected StopAgentWithReason call: %#v", stopCalls[0])
		}
	})

	t.Run("failed_session_stop_failure_preserves_executor_row", func(t *testing.T) {
		repo := setupTestRepo(t)
		ctx := context.Background()
		now := time.Now().UTC()

		seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateFailed)

		err := repo.UpsertExecutorRunning(ctx, &models.ExecutorRunning{
			ID:               "er1",
			SessionID:        "session1",
			TaskID:           "task1",
			AgentExecutionID: "exec-failed",
			CreatedAt:        now,
			UpdatedAt:        now,
		})
		if err != nil {
			t.Fatalf("failed to upsert executor running: %v", err)
		}

		agentMgr := &mockAgentManager{stopAgentWithReasonErr: errors.New("runtime still running")}
		svc := createTestServiceWithAgent(repo, newMockStepGetter(), newMockTaskRepo(), agentMgr)
		svc.reconcileSessionsOnStartup(ctx)

		running, err := repo.GetExecutorRunningBySessionID(ctx, "session1")
		if err != nil {
			t.Fatalf("expected ExecutorRunning record to be preserved after stop failure: %v", err)
		}
		if running.AgentExecutionID != "exec-failed" {
			t.Fatalf("expected execution ID to be preserved, got %q", running.AgentExecutionID)
		}
		agentMgr.mu.Lock()
		stopCalls := append([]stopAgentCall(nil), agentMgr.stopAgentWithReasonArgs...)
		agentMgr.mu.Unlock()
		if len(stopCalls) != 1 {
			t.Fatalf("expected one StopAgentWithReason call, got %d", len(stopCalls))
		}
		if stopCalls[0] != (stopAgentCall{
			ExecutionID: "exec-failed",
			Reason:      "startup failed session cleanup",
			Force:       true,
		}) {
			t.Fatalf("unexpected StopAgentWithReason call: %#v", stopCalls[0])
		}
	})

	// Pins office IDLE preservation: an office session sitting in IDLE
	// (agent torn down between turns, conversation parked for the next
	// run) MUST stay IDLE after backend restart. The previous code
	// path flipped any non-WAITING_FOR_INPUT active state — including
	// IDLE — to WAITING_FOR_INPUT, which made the chat UI render as
	// "Agent working" on a restored task even when nothing was running.
	t.Run("idle_office_session_state_preserved", func(t *testing.T) {
		repo := setupTestRepo(t)
		ctx := context.Background()
		now := time.Now().UTC()

		seedTaskAndSession(t, repo, "task-idle", "session-idle", models.TaskSessionStateIdle)

		err := repo.UpsertExecutorRunning(ctx, &models.ExecutorRunning{
			ID:          "er-idle",
			SessionID:   "session-idle",
			TaskID:      "task-idle",
			ResumeToken: "acp-session-xyz",
			CreatedAt:   now,
			UpdatedAt:   now,
		})
		if err != nil {
			t.Fatalf("failed to upsert executor running: %v", err)
		}

		svc := createTestServiceWithAgent(repo, newMockStepGetter(), newMockTaskRepo(), &mockAgentManager{})
		svc.reconcileSessionsOnStartup(ctx)

		session, err := repo.GetTaskSession(ctx, "session-idle")
		if err != nil {
			t.Fatalf("failed to get session: %v", err)
		}
		if session.State != models.TaskSessionStateIdle {
			t.Fatalf("expected IDLE to be preserved, got %q", session.State)
		}
		// ExecutorRunning row must be preserved — the resume token is
		// what powers the next run's session/load.
		er, err := repo.GetExecutorRunningBySessionID(ctx, "session-idle")
		if err != nil {
			t.Fatalf("expected ExecutorRunning to be preserved for IDLE office session: %v", err)
		}
		if er.ResumeToken != "acp-session-xyz" {
			t.Fatalf("expected resume token to be preserved, got %q", er.ResumeToken)
		}
	})
}

// --- ensureSessionRunning: prepared workspace ---

func TestEnsureSessionRunning_PreparedWorkspace(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)

	// Seed task and session in CREATED state (workspace prepared, agent not started)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateCreated)

	// Set AgentExecutionID to simulate a prepared workspace
	session, err := repo.GetTaskSession(ctx, "session1")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	session.AgentExecutionID = "exec-prepare-1"
	seedExecutorRunning(t, repo, session.ID, session.TaskID, "exec-prepare-1")
	session.AgentProfileID = "profile1"
	if err := repo.UpdateTaskSession(ctx, session); err != nil {
		t.Fatalf("failed to update session: %v", err)
	}

	// Create a wrapped mock agent manager that transitions the session to WAITING_FOR_INPUT
	// when StartAgentProcess is called (simulating the agent starting successfully).
	startAgentProcessCalled := false
	wrappedMgr := &sessionUpdatingAgentManager{
		mockAgentManager: &mockAgentManager{
			isAgentRunning: false,
			// Return the execution ID so the existing-workspace path proceeds
			getExecutionIDForSessionFunc: func(_ context.Context, sid string) (string, error) {
				if sid == "session1" {
					return "exec-prepare-1", nil
				}
				return "", fmt.Errorf("no execution found")
			},
		},
		repo:          repo,
		sessionID:     "session1",
		taskID:        "task1",
		onStartCalled: &startAgentProcessCalled,
	}

	taskRepo := newMockTaskRepo()
	taskRepo.tasks["task1"] = &v1.Task{
		ID:          "task1",
		Title:       "Test Task",
		Description: "desc",
		State:       v1.TaskStateInProgress,
	}

	log := testLogger()
	exec := executor.NewExecutor(wrappedMgr, repo, log, executor.ExecutorConfig{})
	sched := scheduler.NewScheduler(queue.NewTaskQueue(100), exec, taskRepo, log, scheduler.DefaultSchedulerConfig())

	svc := createTestServiceWithAgent(repo, newMockStepGetter(), taskRepo, wrappedMgr)
	svc.executor = exec
	svc.scheduler = sched

	// Re-load session for the call
	session, err = repo.GetTaskSession(ctx, "session1")
	if err != nil {
		t.Fatalf("failed to reload session: %v", err)
	}

	err = svc.ensureSessionRunning(ctx, "session1", session)
	if err != nil {
		t.Fatalf("ensureSessionRunning failed: %v", err)
	}

	if !startAgentProcessCalled {
		t.Fatal("expected StartAgentProcess to be called (prepared workspace path)")
	}

	// Verify the session transitioned through STARTING
	updated, err := repo.GetTaskSession(ctx, "session1")
	if err != nil {
		t.Fatalf("failed to reload session: %v", err)
	}
	if updated.State != models.TaskSessionStateWaitingForInput {
		t.Fatalf("expected session state %q, got %q", models.TaskSessionStateWaitingForInput, updated.State)
	}
}

func TestEnsureSessionRunning_WaitingForInputUsesResumePath(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)

	// Session in WAITING_FOR_INPUT without executor running record → resume path fails gracefully
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateWaitingForInput)

	agentMgr := &mockAgentManager{isAgentRunning: false}
	log := testLogger()
	exec := executor.NewExecutor(agentMgr, repo, log, executor.ExecutorConfig{})

	svc := createTestServiceWithAgent(repo, newMockStepGetter(), newMockTaskRepo(), agentMgr)
	svc.executor = exec

	session, err := repo.GetTaskSession(ctx, "session1")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	// Should fail because there is no executor running record (resume path)
	err = svc.ensureSessionRunning(ctx, "session1", session)
	if err == nil {
		t.Fatal("expected error for WAITING_FOR_INPUT session without executor record")
	}
	// Verify it took the resume path (error mentions "not resumable")
	if !strings.Contains(err.Error(), "not resumable") {
		t.Fatalf("expected 'not resumable' error from resume path, got: %v", err)
	}
}

func TestEnsureSessionRunning_CreatedWithoutExecutionUsesResumePath(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)

	// Session in CREATED state WITHOUT AgentExecutionID → resume path (not prepared workspace)
	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateCreated)

	agentMgr := &mockAgentManager{isAgentRunning: false}
	log := testLogger()
	exec := executor.NewExecutor(agentMgr, repo, log, executor.ExecutorConfig{})

	svc := createTestServiceWithAgent(repo, newMockStepGetter(), newMockTaskRepo(), agentMgr)
	svc.executor = exec

	session, err := repo.GetTaskSession(ctx, "session1")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	// AgentExecutionID is empty → should NOT take prepared workspace path
	// Should fail with "not resumable" because no executor running record
	err = svc.ensureSessionRunning(ctx, "session1", session)
	if err == nil {
		t.Fatal("expected error for CREATED session without executor record")
	}
	if !strings.Contains(err.Error(), "not resumable") {
		t.Fatalf("expected 'not resumable' error from resume path, got: %v", err)
	}
}

// --- canRestoreWorkspace ---

func TestCanRestoreWorkspace(t *testing.T) {
	tests := []struct {
		name string
		resp *dto.TaskSessionStatusResponse
		want bool
	}{
		{
			name: "nil response",
			resp: nil,
			want: false,
		},
		{
			name: "nil worktree path",
			resp: &dto.TaskSessionStatusResponse{},
			want: false,
		},
		{
			name: "empty worktree path",
			resp: &dto.TaskSessionStatusResponse{WorktreePath: strPtr("")},
			want: false,
		},
		{
			name: "valid worktree path",
			resp: &dto.TaskSessionStatusResponse{WorktreePath: strPtr("/tmp/worktrees/session1")},
			want: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := canRestoreWorkspace(tc.resp); got != tc.want {
				t.Errorf("canRestoreWorkspace() = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- GetTaskSessionStatus: NeedsWorkspaceRestore ---

func TestGetTaskSessionStatus_NeedsWorkspaceRestore_TerminalWithWorktree(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)

	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateCompleted)

	// Add worktree to session
	now := time.Now().UTC()
	if err := repo.CreateTaskSessionWorktree(ctx, &models.TaskSessionWorktree{
		ID:             "wt1",
		SessionID:      "session1",
		WorktreeID:     "wid1",
		RepositoryID:   "repo1",
		WorktreePath:   "/tmp/worktrees/session1",
		WorktreeBranch: "feature/test",
		CreatedAt:      now,
	}); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	taskRepo := newMockTaskRepo()
	agentMgr := &mockAgentManager{repoForExecutionLookup: repo}
	svc := createTestServiceWithAgent(repo, newMockStepGetter(), taskRepo, agentMgr)
	svc.executor = executor.NewExecutor(agentMgr, repo, testLogger(), executor.ExecutorConfig{})

	resp, err := svc.GetTaskSessionStatus(ctx, "task1", "session1")
	if err != nil {
		t.Fatalf("GetTaskSessionStatus returned error: %v", err)
	}
	if !resp.NeedsWorkspaceRestore {
		t.Fatal("expected NeedsWorkspaceRestore=true for terminal session with worktree")
	}
	if resp.NeedsResume {
		t.Fatal("expected NeedsResume=false for terminal session")
	}
}

func TestGetTaskSessionStatus_NeedsWorkspaceRestore_TerminalWithoutWorktree(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)

	seedTaskAndSession(t, repo, "task1", "session1", models.TaskSessionStateCompleted)

	taskRepo := newMockTaskRepo()
	agentMgr := &mockAgentManager{repoForExecutionLookup: repo}
	svc := createTestServiceWithAgent(repo, newMockStepGetter(), taskRepo, agentMgr)
	svc.executor = executor.NewExecutor(agentMgr, repo, testLogger(), executor.ExecutorConfig{})

	resp, err := svc.GetTaskSessionStatus(ctx, "task1", "session1")
	if err != nil {
		t.Fatalf("GetTaskSessionStatus returned error: %v", err)
	}
	if resp.NeedsWorkspaceRestore {
		t.Fatal("expected NeedsWorkspaceRestore=false for terminal session without worktree")
	}
}

// sessionUpdatingAgentManager wraps mockAgentManager to update session state
// when StartAgentProcess is called, simulating the agent initialization flow.
type sessionUpdatingAgentManager struct {
	*mockAgentManager
	repo          *sqliterepo.Repository
	sessionID     string
	taskID        string
	onStartCalled *bool
}

func (m *sessionUpdatingAgentManager) StartAgentProcess(_ context.Context, _ string) error {
	*m.onStartCalled = true
	// Simulate the agent starting by transitioning session to WAITING_FOR_INPUT
	ctx := context.Background()
	sess, err := m.repo.GetTaskSession(ctx, m.sessionID)
	if err == nil && sess != nil {
		sess.State = models.TaskSessionStateWaitingForInput
		sess.UpdatedAt = time.Now().UTC()
		_ = m.repo.UpdateTaskSession(ctx, sess)
	}
	return nil
}
