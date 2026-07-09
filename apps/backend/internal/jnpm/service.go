package jnpm

import (
	"context"
	"fmt"
	"regexp"

	"go.uber.org/zap"

	"github.com/AvatarGanymede/pcraft/internal/common/logger"
)

// digitsRe extracts the first run of digits from a raw JNPM id like "#755621"
// or "JNPM-755621" or " 755621 ".
var digitsRe = regexp.MustCompile(`\d+`)

// Service resolves JNPM ticket assignees. It is a thin wrapper over Client so
// the notification layer can depend on a small, mockable surface.
type Service struct {
	client Client
	log    *logger.Logger
}

// NewService builds a service. client may be nil when JNPM is unconfigured; in
// that case Enabled() returns false and ResolveAssigneeEmail errors so callers
// fall back to the admin recipient.
func NewService(client Client, log *logger.Logger) *Service {
	return &Service{client: client, log: log}
}

// Enabled reports whether a real client is wired (token configured).
func (s *Service) Enabled() bool {
	return s != nil && s.client != nil
}

// ResolveAssigneeEmail parses rawJnpmID (may include a leading "#"), fetches
// the issue, and returns the assignee's email + display name. An empty email
// with a nil error means the ticket has no assignee email — the caller should
// fall back to the admin recipient.
func (s *Service) ResolveAssigneeEmail(ctx context.Context, rawJnpmID string) (email, name string, err error) {
	if !s.Enabled() {
		return "", "", fmt.Errorf("jnpm: not configured")
	}
	issueID, ok := parseIssueID(rawJnpmID)
	if !ok {
		return "", "", fmt.Errorf("jnpm: invalid issue id %q", rawJnpmID)
	}
	issue, err := s.client.GetIssue(ctx, issueID)
	if err != nil {
		return "", "", err
	}
	if issue.Assignee.Email == "" && s.log != nil {
		s.log.Warn("jnpm: issue has no assignee email", zap.Int("issue_id", issueID))
	}
	return issue.Assignee.Email, issue.Assignee.Name, nil
}

// parseIssueID extracts a numeric issue id from a raw string. Returns false
// when no digits are present.
func parseIssueID(raw string) (int, bool) {
	match := digitsRe.FindString(raw)
	if match == "" {
		return 0, false
	}
	var id int
	if _, err := fmt.Sscanf(match, "%d", &id); err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
