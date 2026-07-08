package registry

// RoutableProviderIDs is the canonical, single-source-of-truth list of
// provider IDs eligible for office provider routing. pcraft ships a single
// agent (Claude Code), so this list has exactly one entry. It lives in the
// registry package so internal/office/routing's catalogue (KnownProviders)
// can reference it without an import cycle.
//
// The ID MUST match the claude-acp agent's ID() value (claude_acp.go) so
// registry lookups against the enabled real agent succeed.
var RoutableProviderIDs = []string{
	"claude-acp",
}
