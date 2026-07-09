package lark

import (
	"context"
	"strconv"
	"sync"
)

// SentMessage records a message delivered through the mock client.
type SentMessage struct {
	Email string
	Text  string
}

// MockClient is an in-memory Client for E2E and unit tests. It records every
// message so tests can assert on delivery.
type MockClient struct {
	mu   sync.Mutex
	sent []SentMessage
	seq  int
}

// NewMockClient returns an empty mock client.
func NewMockClient() *MockClient {
	return &MockClient{}
}

// SendTextByEmail records the message and returns a synthetic message id.
func (m *MockClient) SendTextByEmail(_ context.Context, email, text string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	m.sent = append(m.sent, SentMessage{Email: email, Text: text})
	return "om_mock_" + strconv.Itoa(m.seq), nil
}

// Sent returns a copy of all recorded messages.
func (m *MockClient) Sent() []SentMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]SentMessage, len(m.sent))
	copy(out, m.sent)
	return out
}
