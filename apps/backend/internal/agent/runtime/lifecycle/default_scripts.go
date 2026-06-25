package lifecycle

// DefaultPrepareScript returns the default prepare script for a given executor type string.
func DefaultPrepareScript(executorType string) string {
	switch executorType {
	case "local":
		return defaultLocalPrepareScript
	case "worktree":
		return defaultWorktreePrepareScript
	default:
		return ""
	}
}

// KandevBranchCheckoutPostlude returns a kandev-managed shell snippet that
// guarantees the session's feature branch is checked out inside the
// workspace, no matter what the user's stored prepare_script does.
//
// Why a postlude instead of just relying on the default script: profiles
// created in the UI snapshot the *then-current* default into their stored
// prepare_script field. When kandev's default is updated to add a new
// kandev-managed step (like the worktree-branch checkout), older profiles
// silently miss it forever. Making the checkout an invariant — appended
// after the user's script — keeps the contract regardless of which default
// the user happens to have stored.
//
// The snippet is wrapped in a subshell + `|| true` so any failure (e.g. the
// user's script never produced /workspace, or the branch is the same as the
// base) is benign and doesn't block agentctl from starting.
//
//nolint:dupword // two `fi` tokens close two distinct shell blocks.
func KandevBranchCheckoutPostlude() string {
	return `

# ---- kandev-managed: ensure session feature branch is checked out ----
# Appended automatically after the user's prepare script. Idempotent and
# non-destructive: prefer an existing local branch (which may carry unpushed
# work after a container resume), then fall through to a fresh tracking
# branch off origin, and only as a last resort create the branch off HEAD.
# The previous "git checkout -B feature origin/feature" form was destructive
# for the resume case — overwriting local commits with the remote tip.
(
  if [ -d "{{workspace.path}}/.git" ] \
     && [ -n "{{worktree.branch}}" ] \
     && [ "{{worktree.branch}}" != "{{repository.branch}}" ]; then
    cd "{{workspace.path}}" || exit 0
    if git rev-parse --verify "{{worktree.branch}}" >/dev/null 2>&1; then
      git checkout "{{worktree.branch}}"
    elif git fetch --depth=1 origin "{{worktree.branch}}" 2>/dev/null; then
      git checkout -b "{{worktree.branch}}" "origin/{{worktree.branch}}"
    else
      git checkout -b "{{worktree.branch}}"
    fi
  fi
) || true
`
}

const defaultLocalPrepareScript = `#!/bin/bash
# Prepare local environment
# Runs before launching the local agent runtime.
# The script executes with working directory set to {{workspace.path}}.
# Use {{repository.path}} when you need the canonical repository root path.

# ---- Repository setup (if configured) ----
{{repository.setup_script}}
`

const defaultWorktreePrepareScript = `#!/bin/bash
# Prepare worktree environment
# Runs after the worktree has already been created/reused by Kandev.
# The script executes with working directory set to {{worktree.path}}.
# Use {{repository.path}} if you need to run commands in the main repository.

# ---- Repository setup (if configured) ----
{{repository.setup_script}}
`