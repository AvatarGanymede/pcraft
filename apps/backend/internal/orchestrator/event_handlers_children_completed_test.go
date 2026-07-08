package orchestrator

import (
	"testing"
	"time"
)

func TestLockChildCompletionOperationKeepsEntryUntilWaitersExit(t *testing.T) {
	svc := &Service{}
	unlockFirst := svc.lockChildCompletionOperation("op")

	secondAcquired := make(chan struct{})
	releaseSecond := make(chan struct{})
	done := make(chan struct{})
	go func() {
		unlockSecond := svc.lockChildCompletionOperation("op")
		close(secondAcquired)
		<-releaseSecond
		unlockSecond()
		close(done)
	}()

	waitForChildCompletionLockRefs(t, svc, "op", 2)
	unlockFirst()
	select {
	case <-secondAcquired:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second lock holder")
	}
	waitForChildCompletionLockRefs(t, svc, "op", 1)
	close(releaseSecond)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second lock release")
	}

	svc.childCompletionLocksMu.Lock()
	_, exists := svc.childCompletionLocks["op"]
	svc.childCompletionLocksMu.Unlock()
	if exists {
		t.Fatal("expected lock entry to be deleted after all holders exit")
	}
}

func waitForChildCompletionLockRefs(t *testing.T, svc *Service, operationID string, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		svc.childCompletionLocksMu.Lock()
		got := 0
		if entry := svc.childCompletionLocks[operationID]; entry != nil {
			got = entry.refs
		}
		svc.childCompletionLocksMu.Unlock()
		if got == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for lock refs %d", want)
}
