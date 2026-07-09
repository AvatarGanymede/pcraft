// Package lark is a minimal Feishu/Lark bot client used to deliver pcraft
// notifications as direct messages. It authenticates as a custom app
// (tenant_access_token) and sends plain-text messages addressed by email.
//
// Configuration is env-driven (PCRAFT_LARK_BASE_DOMAIN, PCRAFT_LARK_APP_ID,
// PCRAFT_LARK_APP_SECRET). See plan/notification-jnpm-lark-plan.md.
package lark

import "fmt"

// DefaultBaseDomain is the Feishu open-platform base used when
// PCRAFT_LARK_BASE_DOMAIN is unset. Use https://open.larksuite.com for the
// international (Lark) tenant.
const DefaultBaseDomain = "https://open.feishu.cn"

// APIError is returned for non-2xx HTTP responses or non-zero Feishu API codes.
type APIError struct {
	StatusCode int
	Code       int
	Message    string
}

func (e *APIError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("lark api: code %d: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("lark api: status %d: %s", e.StatusCode, e.Message)
}
