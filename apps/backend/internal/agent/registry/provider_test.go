package registry

import (
	"testing"
)

// TestProvide_MockAgentModes pins the PCRAFT_MOCK_AGENT behavior:
//   - unset / arbitrary → claude-acp loaded, mock-agent disabled
//   - "true"            → claude-acp loaded, mock-agent enabled
//   - "only"            → only mock-agent registered and enabled
func TestProvide_MockAgentModes(t *testing.T) {
	tests := []struct {
		name            string
		envValue        string
		wantMockEnabled bool
		wantOnlyMock    bool // only mock-agent registered (no real agents)
	}{
		{
			name:            "unset: claude-acp loaded, mock disabled",
			envValue:        "",
			wantMockEnabled: false,
			wantOnlyMock:    false,
		},
		{
			name:            "true: claude-acp loaded, mock enabled",
			envValue:        "true",
			wantMockEnabled: true,
			wantOnlyMock:    false,
		},
		{
			name:            "only: only mock-agent registered and enabled",
			envValue:        "only",
			wantMockEnabled: true,
			wantOnlyMock:    true,
		},
		{
			name:            "arbitrary value: treated as unset",
			envValue:        "false",
			wantMockEnabled: false,
			wantOnlyMock:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PCRAFT_MOCK_AGENT", tt.envValue)

			log := newTestLogger()
			reg, cleanup, err := Provide(log)
			if err != nil {
				t.Fatalf("Provide() error: %v", err)
			}
			defer cleanup() //nolint:errcheck

			// mock-agent is always registered; only its enabled state varies.
			mock, hasMock := reg.Get("mock-agent")
			if !hasMock {
				t.Fatal("mock-agent should always be registered")
			}
			if mock.Enabled() != tt.wantMockEnabled {
				t.Errorf("mock-agent Enabled() = %v, want %v", mock.Enabled(), tt.wantMockEnabled)
			}

			all := reg.List()
			if tt.wantOnlyMock {
				if len(all) != 1 {
					t.Errorf("only mode: expected 1 agent, got %d", len(all))
				}
				if reg.Exists("claude-acp") {
					t.Error("only mode: claude-acp should NOT be registered")
				}
				return
			}
			// Non-only mode: the single real agent claude-acp is loaded.
			if !reg.Exists("claude-acp") {
				t.Error("expected default agent 'claude-acp' to be loaded")
			}
		})
	}
}
