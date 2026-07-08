package p4

import (
	"context"
	"fmt"
)

type MockClient struct {
	Clients []string
	// RootByClient maps a client name to its (root, stream). Unknown clients
	// fall back to a synthesized temp-style root so tests/mock mode always
	// resolve a non-empty working directory.
	RootByClient map[string][2]string
}

func NewMockClient() *MockClient {
	return &MockClient{
		Clients: []string{"default-client"},
		RootByClient: map[string][2]string{
			"default-client": {"/tmp/pcraft-p4/default-client", "//depot/main"},
		},
	}
}

func (m *MockClient) ListClients(_ context.Context, _ string) ([]string, error) {
	return m.Clients, nil
}

func (m *MockClient) CurrentUser(_ context.Context) (string, error) {
	return "mock-user", nil
}

func (m *MockClient) GetClientRoot(_ context.Context, clientName string) (string, string, error) {
	if rs, ok := m.RootByClient[clientName]; ok {
		return rs[0], rs[1], nil
	}
	return fmt.Sprintf("/tmp/pcraft-p4/%s", clientName), "", nil
}
