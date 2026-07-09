// Package jnpm is a thin read-only client for the JNPM Open API. Its only job
// in pcraft is to resolve a ticket's assignee (name + email) from a JNPM issue
// number so the notification layer can route Lark messages to that person.
//
// Configuration is env-driven (PCRAFT_JNPM_BASE_URL, PCRAFT_JNPM_TOKEN); there
// is no per-workspace config or secret store, unlike the Jira/Linear
// integrations. See plan/notification-jnpm-lark-plan.md.
package jnpm

import "fmt"

// DefaultBaseURL is the JNPM Open API base used when PCRAFT_JNPM_BASE_URL is
// unset. Matches the reference ai_summarize deployment.
const DefaultBaseURL = "https://jn-p-api.bytedance.net/jnpm/"

// Assignee is the resolved owner of a JNPM issue. Any field may be empty when
// the upstream record omits it.
type Assignee struct {
	Name     string
	Username string
	Email    string
}

// IssueDetail is the subset of a JNPM issue pcraft consumes.
type IssueDetail struct {
	IssueID  int
	Title    string
	Assignee Assignee
}

// APIError is returned for non-2xx JNPM responses so callers can distinguish
// transport failures from HTTP-level errors.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("jnpm api: status %d: %s", e.StatusCode, e.Message)
}
