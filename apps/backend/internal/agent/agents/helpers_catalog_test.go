package agents

import "testing"

func TestCatalogPermissionSettings_IncludesAgentctlAutoApprove(t *testing.T) {
	catalog := CatalogPermissionSettings(NewClaudeACP())
	setting, ok := catalog[PermissionKeyAutoApprove]
	if !ok {
		t.Fatal("catalog missing auto_approve")
	}
	if setting.ApplyMethod != PermissionApplyMethodAgentctlAutoApprove {
		t.Fatalf("ApplyMethod = %q, want %q", setting.ApplyMethod, PermissionApplyMethodAgentctlAutoApprove)
	}
	if setting.Default {
		t.Fatal("auto_approve must default to false")
	}
}

func TestCatalogPermissionSettings_MergesCodexCLIFlags(t *testing.T) {
	// Removed — CodexACP agent has been deleted.
}
