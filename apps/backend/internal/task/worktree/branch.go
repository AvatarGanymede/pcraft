package worktree

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/AvatarGanymede/pcraft/internal/utility/branchslug"
)

// DefaultBranchPrefix is used when no repository-specific prefix is provided.
const DefaultBranchPrefix = "feature/"

// ValidateBranchPrefix ensures a prefix contains only safe branch characters.
func ValidateBranchPrefix(prefix string) error {
	trimmed := strings.TrimSpace(prefix)
	if trimmed == "" {
		return nil
	}
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '/' || r == '-' || r == '_' || r == '.' {
			continue
		}
		return fmt.Errorf("invalid branch prefix")
	}
	if strings.Contains(trimmed, "..") || strings.Contains(trimmed, "@{") {
		return fmt.Errorf("invalid branch prefix")
	}
	return nil
}

// SanitizeBranchSlug converts a git branch name into a filesystem-safe slug.
func SanitizeBranchSlug(branch string) string {
	return branchslug.SanitizeBranchSlug(branch)
}

// SmallSuffix returns a random alphanumeric suffix up to maxLen characters.
func SmallSuffix(maxLen int) string {
	return branchslug.SmallSuffix(maxLen)
}

// SemanticWorktreeName generates a semantic worktree directory name from a task title.
func SemanticWorktreeName(taskTitle, suffix string) string {
	return branchslug.SemanticWorktreeName(taskTitle, suffix)
}
