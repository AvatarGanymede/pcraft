package jnpm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AvatarGanymede/pcraft/internal/common/logger"
)

func testLogger(_ *testing.T) *logger.Logger {
	return logger.Default()
}

func TestParseIssueID(t *testing.T) {
	cases := []struct {
		in    string
		want  int
		wantK bool
	}{
		{"#755621", 755621, true},
		{"755621", 755621, true},
		{"  #755621 ", 755621, true},
		{"JNPM-42", 42, true},
		{"", 0, false},
		{"#", 0, false},
		{"abc", 0, false},
	}
	for _, tc := range cases {
		got, ok := parseIssueID(tc.in)
		if ok != tc.wantK || (ok && got != tc.want) {
			t.Errorf("parseIssueID(%q) = (%d, %v), want (%d, %v)", tc.in, got, ok, tc.want, tc.wantK)
		}
	}
}

func TestResolveAssigneeEmail_Mock(t *testing.T) {
	mock := NewMockClient()
	mock.SetIssue(&IssueDetail{
		IssueID:  755621,
		Title:    "Fix crash",
		Assignee: Assignee{Name: "Alice", Email: "alice@example.com"},
	})
	svc := NewService(mock, testLogger(t))

	email, name, err := svc.ResolveAssigneeEmail(context.Background(), "#755621")
	if err != nil {
		t.Fatalf("ResolveAssigneeEmail: %v", err)
	}
	if email != "alice@example.com" || name != "Alice" {
		t.Fatalf("got (%q, %q), want (alice@example.com, Alice)", email, name)
	}
}

func TestResolveAssigneeEmail_Disabled(t *testing.T) {
	svc := NewService(nil, testLogger(t))
	if svc.Enabled() {
		t.Fatal("expected disabled service")
	}
	if _, _, err := svc.ResolveAssigneeEmail(context.Background(), "#1"); err == nil {
		t.Fatal("expected error from disabled service")
	}
}

func TestHTTPClient_GetIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("PRIVATE-TOKEN"); got != "tok" {
			t.Errorf("PRIVATE-TOKEN = %q, want tok", got)
		}
		if r.URL.Path != "/v1/open-api/projects/issues/755621" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"payload": map[string]any{
				"title": "Fix crash",
				"assignee": map[string]any{
					"email":    "bob@example.com",
					"fullName": "Bob",
					"username": "bob",
				},
			},
		})
	}))
	defer srv.Close()

	client := NewHTTPClient(srv.URL, "tok")
	issue, err := client.GetIssue(context.Background(), 755621)
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if issue.Assignee.Email != "bob@example.com" || issue.Assignee.Name != "Bob" || issue.Title != "Fix crash" {
		t.Fatalf("unexpected issue: %+v", issue)
	}
}

func TestHTTPClient_GetIssue_FallbackOwner(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"payload": map[string]any{
				"owner": map[string]any{"email": "owner@example.com", "name": "Owner"},
			},
		})
	}))
	defer srv.Close()

	client := NewHTTPClient(srv.URL, "tok")
	issue, err := client.GetIssue(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if issue.Assignee.Email != "owner@example.com" {
		t.Fatalf("expected owner fallback, got %+v", issue.Assignee)
	}
}
