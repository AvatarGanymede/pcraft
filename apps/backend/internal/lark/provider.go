package lark

import (
	"os"

	"github.com/AvatarGanymede/pcraft/internal/common/logger"
)

// Environment variables that configure the Lark bot client.
const (
	envBaseDomain = "PCRAFT_LARK_BASE_DOMAIN"
	envAppID      = "PCRAFT_LARK_APP_ID"
	envAppSecret  = "PCRAFT_LARK_APP_SECRET"
	mockEnvVar    = "PCRAFT_MOCK_LARK"
)

// MockEnabled reports whether the in-memory mock client should be used (E2E).
func MockEnabled() bool {
	return os.Getenv(mockEnvVar) == "true"
}

// Provide builds the Lark notifier from environment configuration. When app
// credentials are missing (and mock is off), the notifier is returned with a
// nil client so Enabled() is false — Lark delivery is skipped rather than
// erroring the whole notification pipeline.
func Provide(log *logger.Logger) *Notifier {
	if MockEnabled() {
		log.Info("lark: using in-memory mock client (PCRAFT_MOCK_LARK=true)")
		return NewNotifier(NewMockClient(), log)
	}
	appID := os.Getenv(envAppID)
	appSecret := os.Getenv(envAppSecret)
	if appID == "" || appSecret == "" {
		log.Info("lark: PCRAFT_LARK_APP_ID/SECRET not set; bot notifications disabled")
		return NewNotifier(nil, log)
	}
	client := NewHTTPClient(os.Getenv(envBaseDomain), appID, appSecret)
	return NewNotifier(client, log)
}
