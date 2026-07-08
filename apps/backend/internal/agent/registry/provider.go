package registry

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/AvatarGanymede/pcraft/internal/agent/agents"
	"github.com/AvatarGanymede/pcraft/internal/agent/runtime/routingerr"
	"github.com/AvatarGanymede/pcraft/internal/common/logger"
	"go.uber.org/zap"
)

// Provide creates and loads the agent registry.
//
// PCRAFT_MOCK_AGENT controls mock-agent availability:
//   - "only"  → E2E mode: only register mock agent, skip all others
//   - "true"  → Dev mode: load all agents AND enable mock agent
//   - unset   → Production: load all agents, mock agent disabled
func Provide(log *logger.Logger) (*Registry, func() error, error) {
	reg := NewRegistry(log)

	mockMode := os.Getenv("PCRAFT_MOCK_AGENT")
	if mockMode == "only" {
		// E2E mode: only register mock agent — skip agent discovery for all others
		_ = reg.Register(agents.NewMockAgent())
		configureMockAgent(reg, "mock-agent", log)
	} else {
		reg.LoadDefaults()
		if mockMode == "true" {
			// Dev mode: enable the base mock agent alongside the real agents.
			configureMockAgent(reg, "mock-agent", log)
		}
		registerRealProvidersProber(reg, log)
	}

	return reg, func() error { return nil }, nil
}

// registerRealProvidersProber wires the shared ACP probe to each
// routable real provider. The probe spawns the agent's CLI via
// BuildCommand, performs a JSON-RPC initialize, and tears down — no
// session ever starts. Called once at boot in non-mock mode so the
// HTTP /routing/retry endpoint can flip a degraded provider back to
// healthy without waiting for the next real launch.
func registerRealProvidersProber(reg *Registry, log *logger.Logger) {
	resolver := func(providerID string) ([]string, map[string]string, bool) {
		ag, ok := reg.Get(providerID)
		if !ok || !ag.Enabled() {
			return nil, nil, false
		}
		cmd := ag.BuildCommand(agents.CommandOptions{})
		if cmd.IsEmpty() {
			return nil, nil, false
		}
		var env map[string]string
		if rt := ag.Runtime(); rt != nil && len(rt.Env) > 0 {
			env = make(map[string]string, len(rt.Env))
			for k, v := range rt.Env {
				env[k] = v
			}
		}
		return cmd.Args(), env, true
	}
	probe := routingerr.NewACPProbe(resolver, log)
	for _, id := range RoutableProviderIDs {
		// Skip ids that are not actually loaded. The probe is harmless to
		// register against a missing agent — but registering only loaded
		// ones keeps the boot log noise minimal.
		if !reg.Exists(id) {
			continue
		}
		routingerr.RegisterProber(id, probe)
	}
}

// configureMockAgent enables and configures the mock agent binary path and capabilities.
// PCRAFT_MOCK_AGENT_MCP=false disables MCP support (defaults to enabled).
func configureMockAgent(reg *Registry, id string, log *logger.Logger) {
	ag, ok := reg.Get(id)
	if !ok {
		return
	}
	mock, ok := ag.(*agents.MockAgent)
	if !ok {
		return
	}
	mock.SetEnabled(true)
	if os.Getenv("PCRAFT_MOCK_AGENT_MCP") == "false" {
		mock.SetSupportsMCP(false)
	}
	// Resolve binary path: same directory as the running executable.
	// On Windows the binary is mock-agent.exe — exec.Command on Windows
	// only auto-appends .exe for PATH lookups, not absolute paths, so
	// the basename must include the extension here.
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	binaryName := "mock-agent"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(filepath.Dir(exePath), binaryName)
	mock.SetBinaryPath(binaryPath)
	log.Info("mock agent enabled",
		zap.String("id", id),
		zap.String("cmd", binaryPath),
		zap.Bool("supports_mcp", mock.SupportsMCPEnabled()))
}
