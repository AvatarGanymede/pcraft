// Package executor defines the agent executor types shared across lifecycle and policy logic.
package executor

import (
	"github.com/AvatarGanymede/pcraft/internal/agentruntime"
	"github.com/AvatarGanymede/pcraft/internal/task/models"
)

// Name identifies the execution backend. It aliases agentruntime.Runtime
// so the executor and runtime layers share a single typed vocabulary
// without forcing every existing consumer to switch import paths.
type Name = agentruntime.Runtime

const (
	NameUnknown   Name = ""
	NameStandalone     = agentruntime.RuntimeStandalone
	NameLocal     Name = "local"
)

// ExecutorTypeToBackend maps an ExecutorType to its corresponding executor Name.
func ExecutorTypeToBackend(execType models.ExecutorType) Name {
	switch execType {
	case models.ExecutorTypeLocal, models.ExecutorTypeWorktree, models.ExecutorTypeMockRemote:
		return NameStandalone
	default:
		return NameStandalone
	}
}
