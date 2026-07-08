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
