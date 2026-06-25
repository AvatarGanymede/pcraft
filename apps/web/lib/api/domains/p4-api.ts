import { fetchJson, type ApiRequestOptions } from "../client";

export type P4Workspace = {
  id: string;
  name: string;
  p4client: string;
  p4user?: string;
  p4stream?: string;
  root_path?: string;
};

export async function listP4Workspaces(p4user?: string, options?: ApiRequestOptions) {
  const qs = p4user ? `?p4user=${encodeURIComponent(p4user)}` : "";
  return fetchJson<{ workspaces: P4Workspace[]; total: number }>(`/api/v1/p4/workspaces${qs}`, options);
}

export async function fetchTaskP4Opened(taskId: string, options?: ApiRequestOptions) {
  return fetchJson<{ changelist: string; files: string[]; total: number }>(
    `/api/v1/tasks/${taskId}/p4/opened`,
    options,
  );
}

export async function closeTaskAfterP4Submit(
  taskId: string,
  payload?: { p4_changelist?: string },
  options?: ApiRequestOptions,
) {
  return fetchJson<{ id: string; state: string }>(`/api/v1/tasks/${taskId}/close`, {
    ...options,
    init: {
      method: "POST",
      body: JSON.stringify(payload ?? {}),
      ...(options?.init ?? {}),
    },
  });
}

export async function updateTaskGuiPhase(
  taskId: string,
  payload: { phase: string; status?: string },
  options?: ApiRequestOptions,
) {
  return fetchJson(`/api/v1/tasks/${taskId}/gui-phase`, {
    ...options,
    init: {
      method: "PATCH",
      body: JSON.stringify(payload),
      ...(options?.init ?? {}),
    },
  });
}
