package providers

import (
	"context"
	"errors"
	"testing"
)

type fakeLarkSender struct {
	enabled   bool
	lastEmail string
	lastTitle string
	lastBody  string
	err       error
	calls     int
}

func (f *fakeLarkSender) Enabled() bool { return f.enabled }
func (f *fakeLarkSender) NotifyByEmail(_ context.Context, email, title, body string) error {
	f.calls++
	f.lastEmail, f.lastTitle, f.lastBody = email, title, body
	return f.err
}

func TestLarkProvider_Available(t *testing.T) {
	if NewLarkProvider(nil, "").Available() {
		t.Error("nil sender should be unavailable")
	}
	if NewLarkProvider(&fakeLarkSender{enabled: false}, "").Available() {
		t.Error("disabled sender should be unavailable")
	}
	if !NewLarkProvider(&fakeLarkSender{enabled: true}, "").Available() {
		t.Error("enabled sender should be available")
	}
}

func TestLarkProvider_Send(t *testing.T) {
	sender := &fakeLarkSender{enabled: true}
	p := NewLarkProvider(sender, "")
	err := p.Send(context.Background(), Message{
		Title:       "Title",
		Body:        "Body",
		TargetEmail: "u@example.com",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if sender.calls != 1 || sender.lastEmail != "u@example.com" || sender.lastTitle != "Title" || sender.lastBody != "Body" {
		t.Fatalf("unexpected send: %+v", sender)
	}
}

func TestLarkProvider_Send_AppendsTaskLink(t *testing.T) {
	sender := &fakeLarkSender{enabled: true}
	p := NewLarkProvider(sender, "http://10.0.0.5:38429/")
	err := p.Send(context.Background(), Message{
		Title:       "Title",
		Body:        "Body",
		TaskID:      "abc-123",
		TargetEmail: "u@example.com",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	want := "Body\n\nhttp://10.0.0.5:38429/t/abc-123"
	if sender.lastBody != want {
		t.Fatalf("body = %q, want %q", sender.lastBody, want)
	}
}

func TestLarkProvider_Send_LinkOnlyBodyWhenEmpty(t *testing.T) {
	sender := &fakeLarkSender{enabled: true}
	p := NewLarkProvider(sender, "http://10.0.0.5:38429")
	if err := p.Send(context.Background(), Message{TaskID: "t1", TargetEmail: "u@example.com"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if sender.lastBody != "http://10.0.0.5:38429/t/t1" {
		t.Fatalf("body = %q", sender.lastBody)
	}
}

func TestLarkProvider_Send_NoLinkWithoutBaseURL(t *testing.T) {
	sender := &fakeLarkSender{enabled: true}
	p := NewLarkProvider(sender, "")
	if err := p.Send(context.Background(), Message{Body: "Body", TaskID: "t1", TargetEmail: "u@example.com"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if sender.lastBody != "Body" {
		t.Fatalf("body = %q, want unchanged", sender.lastBody)
	}
}

func TestLarkProvider_Send_NoRecipient(t *testing.T) {
	sender := &fakeLarkSender{enabled: true}
	p := NewLarkProvider(sender, "")
	if err := p.Send(context.Background(), Message{Title: "t", TargetEmail: "  "}); err == nil {
		t.Fatal("expected error for empty recipient")
	}
	if sender.calls != 0 {
		t.Fatalf("sender should not be called, got %d", sender.calls)
	}
}

func TestLarkProvider_Send_Disabled(t *testing.T) {
	p := NewLarkProvider(&fakeLarkSender{enabled: false}, "")
	if err := p.Send(context.Background(), Message{TargetEmail: "u@example.com"}); err == nil {
		t.Fatal("expected error when sender disabled")
	}
}

func TestLarkProvider_Send_Propagates(t *testing.T) {
	sender := &fakeLarkSender{enabled: true, err: errors.New("boom")}
	p := NewLarkProvider(sender, "")
	if err := p.Send(context.Background(), Message{TargetEmail: "u@example.com"}); err == nil {
		t.Fatal("expected sender error to propagate")
	}
}
