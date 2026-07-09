package lark

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/AvatarGanymede/pcraft/internal/common/logger"
)

// Notifier delivers pcraft notifications through the Lark bot. It is the
// small surface the notification provider depends on.
type Notifier struct {
	client Client
	log    *logger.Logger
}

// NewNotifier builds a notifier. client may be nil when Lark is unconfigured;
// Enabled() then returns false and NotifyByEmail errors.
func NewNotifier(client Client, log *logger.Logger) *Notifier {
	return &Notifier{client: client, log: log}
}

// Enabled reports whether a real client is wired (app credentials configured).
func (n *Notifier) Enabled() bool {
	return n != nil && n.client != nil
}

// NotifyByEmail sends a plain-text message combining title + body to the user
// identified by email.
func (n *Notifier) NotifyByEmail(ctx context.Context, email, title, body string) error {
	if !n.Enabled() {
		return fmt.Errorf("lark: not configured")
	}
	if strings.TrimSpace(email) == "" {
		return fmt.Errorf("lark: empty recipient email")
	}
	text := composeText(title, body)
	msgID, err := n.client.SendTextByEmail(ctx, email, text)
	if err != nil {
		return err
	}
	if n.log != nil {
		n.log.Info("lark: notification sent", zap.String("email", email), zap.String("message_id", msgID))
	}
	return nil
}

// composeText joins the title and body into a single plain-text message. When
// either is empty the other is used verbatim.
func composeText(title, body string) string {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	switch {
	case title == "":
		return body
	case body == "":
		return title
	default:
		return title + "\n" + body
	}
}
