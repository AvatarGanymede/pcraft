package p4

import "context"

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

func (s *Service) ListWorkspaces(ctx context.Context, p4user string) ([]Workspace, error) {
	clients, err := s.client.ListClients(ctx, p4user)
	if err != nil {
		return nil, err
	}
	workspaces := make([]Workspace, 0, len(clients))
	for _, client := range clients {
		workspaces = append(workspaces, Workspace{
			ID:       client,
			Name:     client,
			P4Client: client,
			P4User:   p4user,
		})
	}
	return workspaces, nil
}
