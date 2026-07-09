package providers

import (
	"context"
	"fmt"
	"strings"
)

// LarkSender is the minimal Feishu/Lark bot surface the provider needs.
// Satisfied by *lark.Notifier.
type LarkSender interface {
	Enabled() bool
	NotifyByEmail(ctx context.Context, email, title, body string) error
}

// LarkProvider delivers notifications as Lark bot direct messages. The
// recipient email is resolved upstream (JNPM assignee or admin fallback) and
// arrives on Message.TargetEmail.
type LarkProvider struct {
	sender LarkSender
	// baseURL is the externally reachable pcraft origin (e.g.
	// http://10.0.0.5:38429, no trailing slash). Used to build a task
	// deep-link appended to the message. May be empty (link omitted).
	baseURL string
}

// NewLarkProvider builds a Lark provider around a sender. sender may be nil
// (Lark unconfigured); Available() then reports false. baseURL is the origin
// used to build task deep-links; pass "" to omit the link.
func NewLarkProvider(sender LarkSender, baseURL string) *LarkProvider {
	return &LarkProvider{
		sender:  sender,
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
	}
}

func (p *LarkProvider) Available() bool {
	return p.sender != nil && p.sender.Enabled()
}

func (p *LarkProvider) Validate(_ map[string]interface{}) error {
	return nil
}

func (p *LarkProvider) Send(ctx context.Context, message Message) error {
	if p.sender == nil || !p.sender.Enabled() {
		return fmt.Errorf("lark notifier not configured")
	}
	email := strings.TrimSpace(message.TargetEmail)
	if email == "" {
		return fmt.Errorf("lark: no recipient email resolved")
	}
	return p.sender.NotifyByEmail(ctx, email, message.Title, p.appendTaskLink(message.Body, message.TaskID))
}

// appendTaskLink appends the task deep-link to the message body when both a
// base URL and a task id are available. Returns body unchanged otherwise.
func (p *LarkProvider) appendTaskLink(body, taskID string) string {
	taskID = strings.TrimSpace(taskID)
	if p.baseURL == "" || taskID == "" {
		return body
	}
	link := p.baseURL + "/t/" + taskID
	if strings.TrimSpace(body) == "" {
		return link
	}
	return body + "\n\n" + link
}
