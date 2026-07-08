package p4

import (
	"context"
	"testing"

	"github.com/AvatarGanymede/pcraft/internal/common/logger"
)

func testLogger(t *testing.T) *logger.Logger {
	t.Helper()
	log, err := logger.NewLogger(logger.LoggingConfig{Level: "error", Format: "console"})
	if err != nil {
		t.Fatalf("create test logger: %v", err)
	}
	return log
}

// userTrackingClient records the p4user passed to ListClients so we can assert
// the service falls back to CurrentUser when no user is supplied.
type userTrackingClient struct {
	*MockClient
	currentUser   string
	lastListUser  string
	currentUserErr error
}

func (c *userTrackingClient) ListClients(ctx context.Context, p4user string) ([]string, error) {
	c.lastListUser = p4user
	return c.MockClient.ListClients(ctx, p4user)
}

func (c *userTrackingClient) CurrentUser(_ context.Context) (string, error) {
	return c.currentUser, c.currentUserErr
}

func TestListWorkspaces_DefaultsToCurrentUser(t *testing.T) {
	mock := NewMockClient()
	mock.Clients = []string{"ws-a", "ws-b"}
	client := &userTrackingClient{MockClient: mock, currentUser: "beilin"}
	svc := NewService(client, testLogger(t))

	workspaces, err := svc.ListWorkspaces(context.Background(), "")
	if err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}
	if client.lastListUser != "beilin" {
		t.Fatalf("expected ListClients called with current user %q, got %q", "beilin", client.lastListUser)
	}
	if len(workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(workspaces))
	}
	if workspaces[0].P4User != "beilin" {
		t.Fatalf("expected workspace P4User %q, got %q", "beilin", workspaces[0].P4User)
	}
	if workspaces[0].ID != "ws-a" || workspaces[0].P4Client != "ws-a" {
		t.Fatalf("unexpected workspace fields: %+v", workspaces[0])
	}
}

func TestListWorkspaces_ExplicitUserOverridesDefault(t *testing.T) {
	mock := NewMockClient()
	client := &userTrackingClient{MockClient: mock, currentUser: "beilin"}
	svc := NewService(client, testLogger(t))

	if _, err := svc.ListWorkspaces(context.Background(), "alice"); err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}
	if client.lastListUser != "alice" {
		t.Fatalf("expected explicit user %q to be used, got %q", "alice", client.lastListUser)
	}
}
