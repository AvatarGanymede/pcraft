package p4

import (
	"context"
	"testing"
)

func TestEnsureChangelist_UsesP4Create(t *testing.T) {
	client := NewMockClient()
	svc := NewService(client, testLogger(t), NewMemoryLockStore())
	cl := svc.EnsureChangelist(context.Background(), "task-1")
	if cl != "1000" {
		t.Fatalf("changelist = %q, want 1000", cl)
	}
	if len(client.CreatedDescribe) != 1 || client.CreatedDescribe[0] != "pcraft task task-1" {
		t.Fatalf("unexpected describe: %v", client.CreatedDescribe)
	}
	cl2 := svc.EnsureChangelist(context.Background(), "task-1")
	if cl2 != cl {
		t.Fatalf("expected cached changelist %q, got %q", cl, cl2)
	}
}
