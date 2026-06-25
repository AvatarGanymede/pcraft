package lifecycle

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/AvatarGanymede/pcraft/internal/agent/executor"
	"github.com/AvatarGanymede/pcraft/internal/agentctl/tracing"
	agentctltypes "github.com/AvatarGanymede/pcraft/internal/agentctl/types"
	v1 "github.com/AvatarGanymede/pcraft/pkg/api/v1"
)

const (
	containerStateCreated = "created"
	containerStateExited  = "exited"
	containerStateRunning = "running"
)

// Start starts the lifecycle manager background tasks
func (m *Manager) Start(ctx context.Context) error {
	if m.executorRegistry == nil {
		m.logger.Warn("no runtime registry configured")
		return nil
	}

	runtimeNames := m.executorRegistry.List()
	m.logger.Info("starting lifecycle manager", zap.Int("runtimes", len(runtimeNames)))

	// Check health of all registered runtimes
	healthResults := m.executorRegistry.HealthCheckAll(ctx)
	for name, err := range healthResults {
		if err != nil {
			m.logger.Warn("runtime health check failed",
				zap.String("runtime", string(name)),
				zap.Error(err))
		} else {
			m.logger.Info("runtime is healthy", zap.String("runtime", string(name)))
		}
	}

	// Try to recover executions from all runtimes
	recovered, err := m.executorRegistry.RecoverAll(ctx)
	if err != nil {
		m.logger.Warn("failed to recover executions from some runtimes", zap.Error(err))
	}
	if len(recovered) > 0 {
		for _, ri := range recovered {
			execution := &AgentExecution{
				ID:                   ri.InstanceID,
				TaskID:               ri.TaskID,
				SessionID:            ri.SessionID,
				ContainerID:          ri.ContainerID,
				ContainerIP:          ri.ContainerIP,
				WorkspacePath:        ri.WorkspacePath,
				RuntimeName:          ri.RuntimeName,
				Status:               v1.AgentStatusRunning,
				StartedAt:            time.Now(),
				Metadata:             ri.Metadata,
				agentctl:             ri.Client,
				standaloneInstanceID: ri.StandaloneInstanceID,
				standalonePort:       ri.StandalonePort,
				promptDoneCh:         make(chan PromptCompletionSignal, 1),
			}
			// Create trace span for the recovered session
			_, recoverySpan := tracing.TraceSessionRecovered(
				context.Background(), execution.TaskID, execution.SessionID, execution.ID,
			)
			execution.SetSessionSpan(recoverySpan)
			if execution.agentctl != nil {
				execution.agentctl.SetTraceContext(execution.SessionTraceContext())
			}

			// Create short-lived init span so recovery-phase operations are visible
			_, initSpan := tracing.TraceSessionInit(
				execution.SessionTraceContext(), execution.TaskID, execution.SessionID, execution.ID,
			)

			if err := m.executionStore.Add(execution); err != nil {
				// Should not happen at startup — duplicate sessions in the recovery
				// list signal a DB consistency issue, not a normal race. Log loudly
				// and skip; the first one to land wins.
				m.logger.Error("skipping duplicate execution during recovery",
					zap.String("execution_id", execution.ID),
					zap.String("session_id", execution.SessionID),
					zap.Error(err))
				if execution.agentctl != nil {
					execution.agentctl.Close()
				}
				execution.EndSessionSpan()
				initSpan.End()
				continue
			}

			// Reconcile the persistence row to match the recovered in-memory ID.
			// If executors_running.agent_execution_id had drifted (e.g. from a
			// prior bug or manual edit), the recovered runtime instance is the
			// truth — overwrite the row to match. No-op if already in sync.
			m.persistExecutorRunning(ctx, execution)

			// Reconnect to workspace streams (shell, git, file changes) in background
			// This is needed so shell.input, git status, etc. work after backend restart
			go m.streamManager.ReconnectAll(execution)

			initSpan.End()
		}
		m.logger.Info("recovered executions", zap.Int("count", len(recovered)))
	}

	// Start remote status polling loop for runtimes exposing remote status.
	m.wg.Add(1)
	go m.remoteStatusLoop(ctx)
	m.logger.Info("remote status loop started")
	// Set up callbacks for passthrough mode (using standalone runtime)
	if standaloneRT, err := m.executorRegistry.GetBackend(executor.NameStandalone); err == nil {
		if interactiveRunner := standaloneRT.GetInteractiveRunner(); interactiveRunner != nil {
			// Turn complete callback
			interactiveRunner.SetTurnCompleteCallback(func(sessionID string) {
				m.handlePassthroughTurnComplete(sessionID)
			})

			// Output callback for standalone passthrough (no WorkspaceTracker)
			interactiveRunner.SetOutputCallback(func(output *agentctltypes.ProcessOutput) {
				m.handlePassthroughOutput(output)
			})

			// Status callback for standalone passthrough (no WorkspaceTracker)
			interactiveRunner.SetStatusCallback(func(status *agentctltypes.ProcessStatusUpdate) {
				m.handlePassthroughStatus(status)
			})

			m.logger.Info("passthrough callbacks configured")
		}
	}

	return nil
}

// GetRecoveredExecutions returns a snapshot of all currently tracked executions
// This can be used by the orchestrator to sync with the database
func (m *Manager) GetRecoveredExecutions() []RecoveredExecution {
	executions := m.executionStore.List()
	result := make([]RecoveredExecution, 0, len(executions))
	for _, exec := range executions {
		result = append(result, RecoveredExecution{
			ExecutionID:    exec.ID,
			TaskID:         exec.TaskID,
			SessionID:      exec.SessionID,
			ContainerID:    exec.ContainerID,
			AgentProfileID: exec.AgentProfileID,
		})
	}
	return result
}

// IsShuttingDown reports whether graceful shutdown has begun. Set by
// StopAllAgents before it starts tearing down executions so concurrent
// handlers (e.g. passthrough exit auto-restart, agentctl HTTP calls) can
// skip or downgrade work that would otherwise race the teardown.
func (m *Manager) IsShuttingDown() bool {
	return m.shuttingDown.Load()
}

// closeStopCh closes the manager shutdown channel at most once.
func (m *Manager) closeStopCh() {
	m.stopOnce.Do(func() { close(m.stopCh) })
}

// Stop stops the lifecycle manager and releases resources held by executors.
func (m *Manager) Stop() error {
	m.logger.Info("stopping lifecycle manager")

	m.closeStopCh()
	if m.streamManager != nil {
		m.streamManager.Wait()
	}
	m.wg.Wait()

	// Close executor backends that hold resources (e.g., Docker SDK client).
	if m.executorRegistry != nil {
		m.executorRegistry.CloseAll()
	}

	return nil
}

// StopAllAgents attempts a graceful shutdown of all active agents concurrently.
func (m *Manager) StopAllAgents(ctx context.Context) error {
	m.shuttingDown.Store(true)

	executions := m.executionStore.List()
	if len(executions) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(executions))

	for _, exec := range executions {
		wg.Add(1)
		go func(e *AgentExecution) {
			defer wg.Done()
			if err := m.StopAgent(ctx, e.ID, false); err != nil {
				errCh <- err
				m.logger.Warn("failed to stop agent during shutdown",
					zap.String("execution_id", e.ID),
					zap.Error(err))
			}
		}(exec)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// CleanupStaleExecutionBySessionID clears stale execution state for a session.
// Stub: Docker executor removed in pcraft slim build.
func (m *Manager) CleanupStaleExecutionBySessionID(ctx context.Context, sessionID string) error {
	return nil
}

