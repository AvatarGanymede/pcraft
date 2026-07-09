package lark

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AvatarGanymede/pcraft/internal/common/logger"
)

func TestHTTPClient_SendTextByEmail(t *testing.T) {
	var tokenCalls, sendCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/tenant_access_token/internal"):
			tokenCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0, "msg": "ok", "tenant_access_token": "t-123", "expire": 7200,
			})
		case strings.HasSuffix(r.URL.Path, "/im/v1/messages"):
			sendCalls++
			if got := r.Header.Get("Authorization"); got != "Bearer t-123" {
				t.Errorf("Authorization = %q, want Bearer t-123", got)
			}
			if got := r.URL.Query().Get("receive_id_type"); got != "email" {
				t.Errorf("receive_id_type = %q, want email", got)
			}
			raw, _ := io.ReadAll(r.Body)
			var body map[string]string
			_ = json.Unmarshal(raw, &body)
			if body["receive_id"] != "u@example.com" || body["msg_type"] != "text" {
				t.Errorf("unexpected body: %v", body)
			}
			if !strings.Contains(body["content"], `"text"`) {
				t.Errorf("content missing text field: %q", body["content"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0, "msg": "ok", "data": map[string]any{"message_id": "om_1"},
			})
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := NewHTTPClient(srv.URL, "app", "secret")
	id, err := client.SendTextByEmail(context.Background(), "u@example.com", "hello")
	if err != nil {
		t.Fatalf("SendTextByEmail: %v", err)
	}
	if id != "om_1" {
		t.Fatalf("message id = %q, want om_1", id)
	}
	// Second send should reuse the cached token.
	if _, err := client.SendTextByEmail(context.Background(), "u@example.com", "again"); err != nil {
		t.Fatalf("second send: %v", err)
	}
	if tokenCalls != 1 {
		t.Fatalf("tokenCalls = %d, want 1 (token should be cached)", tokenCalls)
	}
	if sendCalls != 2 {
		t.Fatalf("sendCalls = %d, want 2", sendCalls)
	}
}

func TestNotifier_Disabled(t *testing.T) {
	n := NewNotifier(nil, logger.Default())
	if n.Enabled() {
		t.Fatal("expected disabled notifier")
	}
	if err := n.NotifyByEmail(context.Background(), "u@example.com", "t", "b"); err == nil {
		t.Fatal("expected error from disabled notifier")
	}
}

func TestNotifier_NotifyByEmail(t *testing.T) {
	mock := NewMockClient()
	n := NewNotifier(mock, logger.Default())
	if err := n.NotifyByEmail(context.Background(), "u@example.com", "Title", "Body"); err != nil {
		t.Fatalf("NotifyByEmail: %v", err)
	}
	sent := mock.Sent()
	if len(sent) != 1 || sent[0].Email != "u@example.com" || sent[0].Text != "Title\nBody" {
		t.Fatalf("unexpected sent: %+v", sent)
	}
}

func TestComposeText(t *testing.T) {
	if got := composeText("", "b"); got != "b" {
		t.Errorf("composeText empty title = %q", got)
	}
	if got := composeText("t", ""); got != "t" {
		t.Errorf("composeText empty body = %q", got)
	}
	if got := composeText("t", "b"); got != "t\nb" {
		t.Errorf("composeText = %q", got)
	}
}
