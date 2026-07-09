package jnpm

import (
	"context"
	"sync"
)

// MockClient is an in-memory Client for E2E and unit tests. Seed issues with
// SetIssue; unknown ids return an APIError with status 404.
type MockClient struct {
	mu     sync.RWMutex
	issues map[int]*IssueDetail
}

// NewMockClient returns an empty mock client.
func NewMockClient() *MockClient {
	return &MockClient{issues: make(map[int]*IssueDetail)}
}

// SetIssue seeds (or overwrites) a mock issue.
func (m *MockClient) SetIssue(issue *IssueDetail) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.issues[issue.IssueID] = issue
}

// GetIssue returns a seeded issue or a 404 APIError.
func (m *MockClient) GetIssue(_ context.Context, issueID int) (*IssueDetail, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if issue, ok := m.issues[issueID]; ok {
		return issue, nil
	}
	return nil, &APIError{StatusCode: 404, Message: "issue not found"}
}
