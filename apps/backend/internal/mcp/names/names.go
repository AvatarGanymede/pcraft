// Package names holds stable identifiers for the built-in Pcraft MCP server.
package names

const (
	// ServerName is the MCP server key injected into agent sessions
	// (e.g. mcp__pcraft__list_tasks_pcraft).
	ServerName = "pcraft"
	// ToolSuffix is appended to every first-party MCP tool (list_tasks_pcraft, …).
	ToolSuffix = "_pcraft"
	// LegacyServerName is the pre-rename server key kept for profile-config skip logic.
	LegacyServerName = "kandev"
)

// Tool returns the registered MCP tool name for a base action stem.
func Tool(stem string) string {
	return stem + ToolSuffix
}

// IsReservedServer reports whether name is the built-in Pcraft MCP server
// (current or legacy alias). Profile configs must not shadow it.
func IsReservedServer(name string) bool {
	return name == ServerName || name == LegacyServerName
}
