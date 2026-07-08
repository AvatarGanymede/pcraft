import type { WorkspaceState } from "@/lib/state/slices/workspace/types";
import type { ListWorkspacesResponse } from "@/lib/types/http";

export const ACTIVE_WORKSPACE_COOKIE = "kandev-active-workspace";
export const LEGACY_OFFICE_ACTIVE_WORKSPACE_COOKIE = "office-active-workspace";

type WorkspaceItem = ListWorkspacesResponse["workspaces"][number];

export function mapWorkspaceItem(ws: WorkspaceItem): WorkspaceState["items"][number] {
  return {
    id: ws.id,
    name: ws.name,
    description: ws.description ?? null,
    owner_id: ws.owner_id,
    default_environment_id: ws.default_environment_id ?? null,
    default_agent_profile_id: ws.default_agent_profile_id ?? null,
    default_config_agent_profile_id: ws.default_config_agent_profile_id ?? null,
    office_workflow_id: ws.office_workflow_id ?? null,
    task_form_config: ws.task_form_config ?? null,
    p4_client: ws.p4_client ?? "",
    p4_root: ws.p4_root ?? "",
    p4_stream: ws.p4_stream ?? "",
    created_at: ws.created_at,
    updated_at: ws.updated_at,
  };
}

export function readCookie(name: string): string | null {
  if (typeof document === "undefined") return null;
  const encodedName = `${encodeURIComponent(name)}=`;
  const entries = document.cookie
    .split(";")
    .map((part) => part.trim())
    .filter((part) => part.startsWith(encodedName));
  const entry = entries[entries.length - 1];
  return entry ? decodeURIComponent(entry.slice(encodedName.length)) : null;
}

export function readActiveWorkspaceCookie(): string | null {
  return readCookie(ACTIVE_WORKSPACE_COOKIE) || readCookie(LEGACY_OFFICE_ACTIVE_WORKSPACE_COOKIE);
}
