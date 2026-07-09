// Executor types match models.ExecutorType in apps/backend/internal/task/models/models.go.

export type CleanupSummary = {
  lines: string[];
};

type KnownExecutor = "local";

const SINGLE: Record<KnownExecutor, string> = {
  local: "The agent ran directly in your repo — your files, branch, and folder are not touched.",
};

const GENERIC_LINE = "Any running agent sessions will be stopped.";

function normalize(executorType: string | null | undefined): KnownExecutor | null {
  if (!executorType) return null;
  const key = executorType.toLowerCase();
  if (key in SINGLE) return key as KnownExecutor;
  return null;
}

/** Single-task variant. */
export function getCleanupSummary(executorType: string | null | undefined): CleanupSummary {
  const known = normalize(executorType);
  if (!known) return { lines: [GENERIC_LINE] };
  return { lines: [SINGLE[known], GENERIC_LINE] };
}

const PLURAL: Partial<Record<KnownExecutor, (n: number) => string>> = {
  local: (n) => `${n} local ${pl(n, "task", "tasks")} — your repo folders won't be touched.`,
};

function pl(n: number, one: string, many: string): string {
  return n === 1 ? one : many;
}

/** Bulk variant — groups N tasks by executor type and emits one line per group. */
export function getBulkCleanupSummary(
  executorTypes: Array<string | null | undefined>,
): CleanupSummary {
  if (executorTypes.length === 0) return { lines: [GENERIC_LINE] };

  const counts = new Map<KnownExecutor, number>();
  for (const t of executorTypes) {
    const known = normalize(t);
    if (!known) continue;
    counts.set(known, (counts.get(known) ?? 0) + 1);
  }

  const order: KnownExecutor[] = ["local"];

  const lines: string[] = [];
  for (const key of order) {
    const count = counts.get(key);
    if (!count) continue;
    const fmt = PLURAL[key];
    if (fmt) lines.push(fmt(count));
  }
  lines.push(GENERIC_LINE);
  return { lines };
}
