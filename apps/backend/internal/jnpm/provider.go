package jnpm

import (
	"os"

	"github.com/AvatarGanymede/pcraft/internal/common/logger"
)

// Environment variables that configure the JNPM client.
const (
	envBaseURL = "PCRAFT_JNPM_BASE_URL"
	envToken   = "PCRAFT_JNPM_TOKEN"
	mockEnvVar = "PCRAFT_MOCK_JNPM"
)

// MockEnabled reports whether the in-memory mock client should be used (E2E).
func MockEnabled() bool {
	return os.Getenv(mockEnvVar) == "true"
}

// Provide builds the JNPM service from environment configuration. When no token
// is set (and mock is off), the service is returned with a nil client so
// Enabled() is false and callers fall back to the admin recipient — JNPM is an
// optional enhancement, never a hard dependency.
func Provide(log *logger.Logger) *Service {
	if MockEnabled() {
		log.Info("jnpm: using in-memory mock client (PCRAFT_MOCK_JNPM=true)")
		return NewService(NewMockClient(), log)
	}
	token := os.Getenv(envToken)
	if token == "" {
		log.Info("jnpm: PCRAFT_JNPM_TOKEN not set; assignee resolution disabled")
		return NewService(nil, log)
	}
	client := NewHTTPClient(os.Getenv(envBaseURL), token)
	return NewService(client, log)
}
