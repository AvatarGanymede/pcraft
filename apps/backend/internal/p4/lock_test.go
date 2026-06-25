package p4

import (
	"context"
	"testing"

	"github.com/AvatarGanymede/pcraft/internal/common/logger"
)

func testLogger(t *testing.T) *logger.Logger {
	t.Helper()
	log, err := logger.NewLogger(logger.LoggingConfig{Level: "error", Format: "json"})
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	return log
}

func TestCheckout_AllowsWhenUnlocked(t *testing.T) {
	svc := NewService(NewMockClient(), testLogger(t), NewMemoryLockStore())
	result, err := svc.Checkout(context.Background(), "task-a", []string{"//depot/main/foo.cs"})
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	if !result.Allowed {
		t.Fatal("expected checkout allowed")
	}
}

func TestCheckout_DeniesWhenLocked(t *testing.T) {
	svc := NewService(NewMockClient(), testLogger(t), NewMemoryLockStore())
	ctx := context.Background()
	if _, err := svc.Checkout(ctx, "task-a", []string{"//depot/main/foo.cs"}); err != nil {
		t.Fatalf("first checkout: %v", err)
	}
	result, err := svc.Checkout(ctx, "task-b", []string{"//depot/main/foo.cs"})
	if err != ErrFileLocked {
		t.Fatalf("expected ErrFileLocked, got %v", err)
	}
	if result == nil || result.Allowed || result.ConflictTaskID != "task-a" {
		t.Fatalf("unexpected conflict result: %+v", result)
	}
}

func TestConflictRevertReleasesLocks(t *testing.T) {
	client := NewMockClient()
	svc := NewService(client, testLogger(t), NewMemoryLockStore())
	ctx := context.Background()
	clA := svc.EnsureChangelist(ctx, "task-a")
	svc.BindChangelist("task-a", clA)
	if _, err := svc.Checkout(ctx, "task-a", []string{"//depot/main/shared.cs"}); err != nil {
		t.Fatalf("task-a checkout: %v", err)
	}
	clB := svc.EnsureChangelist(ctx, "task-b")
	svc.BindChangelist("task-b", clB)
	if _, err := svc.Checkout(ctx, "task-b", []string{"//depot/main/other.cs"}); err != nil {
		t.Fatalf("task-b checkout: %v", err)
	}
	result, err := svc.Checkout(ctx, "task-b", []string{"//depot/main/shared.cs"})
	if err != ErrFileLocked || result.ConflictTaskID != "task-a" {
		t.Fatalf("expected conflict with task-a, got %+v err=%v", result, err)
	}
	if err := svc.RevertChangelist(ctx, clB); err != nil {
		t.Fatalf("revert: %v", err)
	}
	svc.ReleaseByTask("task-b")
	owner, ok, err := svc.store.GetOwner(ctx, "//depot/main/shared.cs")
	if err != nil || !ok || owner != "task-a" {
		t.Fatalf("task-a lock should remain, got owner=%q ok=%v err=%v", owner, ok, err)
	}
}

func TestConfirmSubmittedAndRelease(t *testing.T) {
	client := NewMockClient()
	client.Submitted["12345"] = true
	svc := NewService(client, testLogger(t), NewMemoryLockStore())
	svc.taskChangelist["task-a"] = "12345"
	ctx := context.Background()
	if _, err := svc.Checkout(ctx, "task-a", []string{"//depot/main/foo.cs"}); err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if err := svc.ConfirmSubmittedAndRelease(ctx, "task-a", "12345"); err != nil {
		t.Fatalf("ConfirmSubmittedAndRelease: %v", err)
	}
	if cl := svc.GetChangelist("task-a"); cl != "" {
		t.Fatalf("expected changelist cleared, got %q", cl)
	}
}
