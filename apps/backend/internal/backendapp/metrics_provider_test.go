package backendapp

import (
	"reflect"
	"testing"
	"unsafe"

	agentctl "github.com/AvatarGanymede/pcraft/internal/agent/runtime/agentctl"
	"github.com/AvatarGanymede/pcraft/internal/agent/runtime/lifecycle"
	"github.com/AvatarGanymede/pcraft/internal/agentruntime"
	"github.com/AvatarGanymede/pcraft/internal/common/logger"
	"go.uber.org/zap"
)

type metricExecutionListStub struct {
	executions []*lifecycle.AgentExecution
}

func (s metricExecutionListStub) ListExecutions() []*lifecycle.AgentExecution {
	return s.executions
}

func TestLifecycleMetricProviderMetricExecutions(t *testing.T) {
	standalone := executionWithAgentCtl(t, &lifecycle.AgentExecution{
		ID:          "exec-standalone",
		TaskID:      "task-1",
		SessionID:   "session-1",
		RuntimeName: agentruntime.RuntimeStandalone,
	})

	provider := lifecycleMetricProvider{manager: metricExecutionListStub{executions: []*lifecycle.AgentExecution{
		nil,
		standalone,
		{
			ID:          "exec-no-client",
			TaskID:      "task-2",
			SessionID:   "session-2",
			RuntimeName: agentruntime.RuntimeStandalone,
		},
	}}}

	sources := provider.MetricExecutions()
	if len(sources) != 0 {
		t.Fatalf("expected 0 execution sources, got %d", len(sources))
	}
}

func TestLifecycleMetricProviderMetricExecutionsNilManager(t *testing.T) {
	sources := (lifecycleMetricProvider{}).MetricExecutions()
	if sources != nil {
		t.Fatalf("expected nil sources, got %#v", sources)
	}
}

func executionWithAgentCtl(t *testing.T, execution *lifecycle.AgentExecution) *lifecycle.AgentExecution {
	t.Helper()
	log, err := logger.NewFromZap(zap.NewNop())
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	setAgentCtlClient(t, execution, agentctl.NewClient("127.0.0.1", 1, log))
	return execution
}

func setAgentCtlClient(t *testing.T, execution *lifecycle.AgentExecution, client *agentctl.Client) {
	t.Helper()
	field := reflect.ValueOf(execution).Elem().FieldByName("agentctl")
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(client))
}
