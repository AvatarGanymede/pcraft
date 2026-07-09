package names

import "testing"

func TestTool(t *testing.T) {
	if got := Tool("list_tasks"); got != "list_tasks_pcraft" {
		t.Fatalf("Tool() = %q", got)
	}
}

func TestIsReservedServer(t *testing.T) {
	if !IsReservedServer("pcraft") || !IsReservedServer("kandev") {
		t.Fatal("expected pcraft and legacy kandev to be reserved")
	}
	if IsReservedServer("github") {
		t.Fatal("github should not be reserved")
	}
}
