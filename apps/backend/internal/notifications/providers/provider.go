package providers

import (
	"context"
)

type Message struct {
	EventType     string
	Title         string
	Body          string
	TaskID        string
	TaskSessionID string
	UserID        string
	Config        map[string]interface{}
	// TargetEmail is the resolved recipient for email-addressed providers
	// (e.g. Lark). Populated by the service layer before dispatch; ignored by
	// providers that don't address by email.
	TargetEmail string
}

type Provider interface {
	Available() bool
	Validate(config map[string]interface{}) error
	Send(ctx context.Context, message Message) error
}
