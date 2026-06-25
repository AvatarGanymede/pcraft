package process

import (
	"testing"

	"github.com/AvatarGanymede/pcraft/internal/agentctl/server/adapter"
	"github.com/AvatarGanymede/pcraft/internal/agentctl/types"
)

func TestExtractFilePaths(t *testing.T) {
	req := &adapter.PermissionRequest{
		ActionType: string(types.ActionTypeFileWrite),
		ActionDetails: map[string]interface{}{
			"path": "//depot/main/foo.cs",
		},
	}
	got := extractFilePaths(req)
	if len(got) != 1 || got[0] != "//depot/main/foo.cs" {
		t.Fatalf("unexpected paths: %v", got)
	}
}

func TestShouldInterceptP4Checkout(t *testing.T) {
	if !shouldInterceptP4Checkout(&adapter.PermissionRequest{
		ActionType: string(types.ActionTypeFileWrite),
		ActionDetails: map[string]interface{}{"path": "a.cs"},
	}) {
		t.Fatal("expected intercept")
	}
	if shouldInterceptP4Checkout(&adapter.PermissionRequest{ActionType: "command"}) {
		t.Fatal("expected no intercept for command")
	}
}
