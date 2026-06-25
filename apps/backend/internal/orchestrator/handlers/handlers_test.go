package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/AvatarGanymede/pcraft/internal/common/logger"
	"github.com/AvatarGanymede/pcraft/internal/orchestrator"
	ws "github.com/AvatarGanymede/pcraft/pkg/websocket"
	"github.com/stretchr/testify/require"
)

func setupOrchestratorHandlers(t *testing.T) *Handlers {
	t.Helper()
	log, err := logger.NewLogger(logger.LoggingConfig{
		Level:      "error",
		Format:     "console",
		OutputPath: "stderr",
	})
	require.NoError(t, err)
	return NewHandlers(&orchestrator.Service{}, log)
}

func TestWsRecoverSessionCancelRetryReportsServiceResult(t *testing.T) {
	handlers := setupOrchestratorHandlers(t)
	response, err := handlers.wsRecoverSession(context.Background(), createTestMessage(t, ws.ActionSessionRecover, map[string]interface{}{
		"task_id":    "t1",
		"session_id": "s1",
		"action":     "cancel_retry",
	}))
	require.NoError(t, err)

	var payload struct {
		Cancelled bool `json:"cancelled"`
	}
	require.NoError(t, json.Unmarshal(response.Payload, &payload))
	require.False(t, payload.Cancelled)
}
