package p4

import (
	"context"
	"strings"

	"go.uber.org/zap"
)

type Workspace struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	P4Port      string `json:"p4port"`
	P4User      string `json:"p4user"`
	P4Client    string `json:"p4client"`
	P4Stream    string `json:"p4stream"`
	RootPath    string `json:"root_path"`
	IsDefault   bool   `json:"is_default"`
}

// ResolveClientRoot returns the local Root and Stream for a named p4 client,
// resolved via `p4 client -o`. Used when a pcraft workspace binds a client so
// the root can be cached and later used as a task's working directory.
func (s *Service) ResolveClientRoot(ctx context.Context, clientName string) (root, stream string, err error) {
	return s.client.GetClientRoot(ctx, clientName)
}

// ListWorkspaces returns the p4 clients (workspaces) for a user. When p4user is
// empty we default to the locally configured P4USER so the task-create dialog
// shows only the current developer's workspaces instead of every client on the
// server (which can number in the thousands on shared depots).
func (s *Service) ListWorkspaces(ctx context.Context, p4user string) ([]Workspace, error) {
	p4user = strings.TrimSpace(p4user)
	if p4user == "" {
		if current, err := s.client.CurrentUser(ctx); err == nil {
			p4user = strings.TrimSpace(current)
		}
	}
	clients, err := s.client.ListClients(ctx, p4user)
	if err != nil {
		return nil, err
	}
	workspaces := make([]Workspace, 0, len(clients))
	for _, client := range clients {
		ws := Workspace{
			ID:       client,
			Name:     client,
			P4Client: client,
			P4User:   p4user,
		}
		// Best-effort: resolve Root/Stream so the settings UI can show them
		// immediately on selection. A failure for one client (e.g. spec
		// unreadable) leaves its Root/Stream empty rather than dropping the
		// whole list.
		if root, stream, rerr := s.client.GetClientRoot(ctx, client); rerr == nil {
			ws.RootPath = root
			ws.P4Stream = stream
		} else if s.log != nil {
			s.log.Warn("p4 resolve client root failed; leaving empty",
				zap.String("client", client), zap.Error(rerr))
		}
		workspaces = append(workspaces, ws)
	}
	return workspaces, nil
}
