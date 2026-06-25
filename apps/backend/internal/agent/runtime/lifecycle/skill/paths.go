package skill

import (
	"path/filepath"
	"regexp"
	"strings"
)

// validSlugRe matches slugs that are safe for use in shell commands
// and on-disk paths. Anything outside this set is dropped during
// delivery to avoid path-traversal or shell-quoting hazards.
var validSlugRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// isValidSlug reports whether the given slug is safe.
func isValidSlug(s string) bool { return s != "" && validSlugRe.MatchString(s) }

// isValidPathComponent reports whether the given filename is a single
// safe path component (no separators, no traversal). Used when writing
// instruction files where the filename comes from upstream data.
func isValidPathComponent(s string) bool {
	if s == "" {
		return false
	}
	if strings.ContainsAny(s, "/\\") {
		return false
	}
	if strings.Contains(s, "..") {
		return false
	}
	return true
}

// instructionsDirHost returns the on-host directory where a manifest's
// instruction files are written.
func instructionsDirHost(kandevBase, workspaceSlug, agentID string) string {
	return filepath.Join(kandevBase, "runtime", workspaceSlug, "instructions", agentID)
}
