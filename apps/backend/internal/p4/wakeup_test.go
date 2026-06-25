package p4

import (
	"testing"

	v1 "github.com/AvatarGanymede/pcraft/pkg/api/v1"
)

func TestResumePrompt(t *testing.T) {
	got := ResumePrompt("", map[string]interface{}{
		"panel_id":    "MainPanel",
		"requirement": "Add login button",
		"prefab":      "Assets/UI/Login.prefab",
	})
	want := "/dev-gui-plugin:run MainPanel, Add login button, resume=true, prefab=Assets/UI/Login.prefab"
	if got != want {
		t.Fatalf("ResumePrompt = %q, want %q", got, want)
	}
}

func TestNormalizeState(t *testing.T) {
	if got := NormalizeState(v1.TaskStateReview); got != v1.TaskStateInProgress {
		t.Fatalf("REVIEW -> %q", got)
	}
	if got := NormalizeState(v1.TaskStateCompleted); got != v1.TaskStateDone {
		t.Fatalf("COMPLETED -> %q", got)
	}
}

func TestFilterWakeupCandidates(t *testing.T) {
	all := []BacklogCandidate{
		{ID: "a", WorkspaceID: "ws1"},
		{ID: "b", WorkspaceID: "ws2"},
		{ID: "c", WorkspaceID: "ws1"},
	}
	got := FilterWakeupCandidates(all, "ws1")
	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(got))
	}
}
