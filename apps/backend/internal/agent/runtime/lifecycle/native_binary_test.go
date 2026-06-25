package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AvatarGanymede/pcraft/internal/agent/agents"
	"github.com/AvatarGanymede/pcraft/internal/agentruntime"
)

// TestPreferNativeBinary_NonNativeAgent verifies that an agent which does not
// implement NativeBinaryAgent never prefers a native binary, regardless of
// runtime or metadata.
func TestPreferNativeBinary_NonNativeAgent(t *testing.T) {
	m := &Manager{}
	ag := agents.NewClaudeACP() // does not implement NativeBinaryAgent
	meta := map[string]interface{}{MetadataKeyNativeBinary: "claude"}
	if m.preferNativeBinary(ag, agentruntime.RuntimeStandalone, meta) {
		t.Error("non-native agent should never prefer a native binary")
	}
}

// TestPreferNativeBinary_StandaloneLooksUpPath verifies the standalone branch
// probes the backend host's PATH (which is the execution environment for
// local_pc / worktree). A binary on PATH prefers native; an empty PATH does not.
func TestPreferNativeBinary_StandaloneLooksUpPath(t *testing.T) {
	m := &Manager{}
	ag := agents.NewCopilotACP()

	dir := t.TempDir()
	bin := filepath.Join(dir, "copilot")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write fake copilot: %v", err)
	}

	t.Setenv("PATH", dir)
	if !m.preferNativeBinary(ag, agentruntime.RuntimeStandalone, nil) {
		t.Error("standalone with copilot on PATH should prefer native binary")
	}

	t.Setenv("PATH", t.TempDir()) // empty dir, no copilot
	if m.preferNativeBinary(ag, agentruntime.RuntimeStandalone, nil) {
		t.Error("standalone without copilot on PATH should fall back to npx")
	}
}
