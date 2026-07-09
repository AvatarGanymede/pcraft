/**
 * Canonical executor type identifiers. Mirrors the backend's database
 * `executors.type` values. See `CLAUDE.md` -> Executor Types.
 *
 * Keep this union in sync with the backend:
 *   apps/backend/internal/task/models/models.go (ExecutorType constants)
 *
 * Only "local" and "mock_remote" are supported. All other types (docker,
 * sprites, worktree, SSH) have been removed.
 */
export type ExecutorType =
  | "local"
  | "local_pc"
  | "mock_remote";
