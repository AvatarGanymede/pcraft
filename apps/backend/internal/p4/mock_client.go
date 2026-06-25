package p4

import (
	"context"
	"fmt"
)

type MockClient struct {
	Clients         []string
	OpenedByCL      map[string][]string
	Submitted       map[string]bool
	NextChangelist  int
	CreatedDescribe []string
}

func NewMockClient() *MockClient {
	return &MockClient{
		Clients:        []string{"default-client"},
		OpenedByCL:     map[string][]string{},
		Submitted:      map[string]bool{},
		NextChangelist: 1000,
	}
}

func (m *MockClient) CreateChangelist(_ context.Context, description string) (string, error) {
	cl := fmt.Sprintf("%d", m.NextChangelist)
	m.NextChangelist++
	m.CreatedDescribe = append(m.CreatedDescribe, description)
	return cl, nil
}

func (m *MockClient) ListClients(_ context.Context, _ string) ([]string, error) {
	return m.Clients, nil
}

func (m *MockClient) CheckoutFiles(_ context.Context, changelist string, files []string) error {
	m.OpenedByCL[changelist] = append(m.OpenedByCL[changelist], files...)
	return nil
}

func (m *MockClient) RevertChangelist(_ context.Context, changelist string) error {
	delete(m.OpenedByCL, changelist)
	return nil
}

func (m *MockClient) OpenedFiles(_ context.Context, changelist string) ([]string, error) {
	return m.OpenedByCL[changelist], nil
}

func (m *MockClient) IsSubmitted(_ context.Context, changelist string) (bool, error) {
	return m.Submitted[changelist], nil
}
