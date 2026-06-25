package branchslug

import (
	"crypto/rand"
	"strings"
	"unicode"
)

const branchSuffixAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// SmallSuffix returns a random alphanumeric suffix up to maxLen characters.
func SmallSuffix(maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if maxLen > 3 {
		maxLen = 3
	}
	buf := make([]byte, maxLen)
	if _, err := rand.Read(buf); err != nil {
		return strings.Repeat("x", maxLen)
	}
	for i := range buf {
		buf[i] = branchSuffixAlphabet[int(buf[i])%len(branchSuffixAlphabet)]
	}
	return string(buf)
}

// SanitizeBranchSlug converts a git branch name into a filesystem-safe slug.
func SanitizeBranchSlug(branch string) string {
	if branch == "" {
		return ""
	}
	var sb strings.Builder
	sb.Grow(len(branch))
	for _, r := range branch {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			sb.WriteRune(r)
		case r == '_', r == '.', r == '-':
			sb.WriteRune(r)
		default:
			sb.WriteRune('-')
		}
	}
	return strings.Trim(sb.String(), "-")
}

func sanitizeForBranch(name string, maxLen int) string {
	slug := SanitizeBranchSlug(name)
	if maxLen > 0 && len(slug) > maxLen {
		slug = slug[:maxLen]
	}
	return strings.Trim(slug, "-")
}

// SemanticWorktreeName generates a semantic worktree directory name from a task title.
func SemanticWorktreeName(taskTitle, suffix string) string {
	semanticName := sanitizeForBranch(taskTitle, 20)
	if semanticName == "" {
		return suffix
	}
	return semanticName + "_" + suffix
}
