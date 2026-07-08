"use client";

import { useEffect, useState } from "react";
import Link from "@/components/routing/app-link";
import { useRouter } from "@/lib/routing/client-router";
import { IconLayoutColumns, IconTrash } from "@tabler/icons-react";
import { Button } from "@pcraft/ui/button";
import { Input } from "@pcraft/ui/input";
import { Label } from "@pcraft/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@pcraft/ui/card";
import { Separator } from "@pcraft/ui/separator";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@pcraft/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@pcraft/ui/dialog";
import { updateWorkspaceAction, deleteWorkspaceAction } from "@/app/actions/workspaces";
import { listP4Workspaces, type P4Workspace } from "@/lib/api/domains/p4-api";
import type { AgentProfileOption, WorkspaceState } from "@/lib/state/slices";

type Workspace = WorkspaceState["items"][number];
import { useRequest } from "@/lib/http/use-request";
import { useToast } from "@/components/toast-provider";
import { useAppStore } from "@/components/state-provider";
import { UnsavedChangesBadge, UnsavedSaveButton } from "@/components/settings/unsaved-indicator";
import { WorkspaceTaskFormCard } from "@/components/settings/workspace-task-form-card";

type WorkspaceEditClientProps = {
  workspaceId: string;
};

export function WorkspaceEditClient({ workspaceId }: WorkspaceEditClientProps) {
  const workspace = useAppStore(
    (state) => state.workspaces.items.find((item: Workspace) => item.id === workspaceId) ?? null,
  );

  if (!workspace) {
    return (
      <div>
        <Card>
          <CardContent className="py-12 text-center">
            <p className="text-muted-foreground">Workspace not found</p>
            <Button className="mt-4" asChild>
              <Link href="/settings/workspace">Back to Workspaces</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return <WorkspaceEditForm key={workspace.id} workspace={workspace} />;
}

type WorkspaceEditFormProps = {
  workspace: Workspace;
};

type SelectFieldProps = {
  label: string;
  value: string;
  onChange: (v: string) => void;
  options: { id: string; name: string }[];
  emptyLabel: string;
  emptyValue: string;
};

function SelectField({
  label,
  value,
  onChange,
  options,
  emptyLabel,
  emptyValue,
}: SelectFieldProps) {
  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      <Select value={value || "none"} onValueChange={(v) => onChange(v === "none" ? "" : v)}>
        <SelectTrigger className="w-full">
          <SelectValue placeholder={`Select ${label.toLowerCase()}`} />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="none">No default</SelectItem>
          {options.map((opt) => (
            <SelectItem key={opt.id} value={opt.id}>
              {opt.name}
            </SelectItem>
          ))}
          {options.length === 0 && (
            <SelectItem value={emptyValue} disabled>
              {emptyLabel}
            </SelectItem>
          )}
        </SelectContent>
      </Select>
    </div>
  );
}

type WorkspaceSettingsCardProps = {
  isDirty: boolean;
  workspaceNameDraft: string;
  onNameChange: (value: string) => void;
  defaultAgentProfileId: string;
  onAgentProfileChange: (value: string) => void;
  agentProfiles: AgentProfileOption[];
  isLoading: boolean;
  saveStatus: "idle" | "loading" | "success" | "error";
  onSave: () => void;
};

function WorkspaceSettingsCard({
  isDirty,
  workspaceNameDraft,
  onNameChange,
  defaultAgentProfileId,
  onAgentProfileChange,
  agentProfiles,
  isLoading,
  saveStatus,
  onSave,
}: WorkspaceSettingsCardProps) {
  const profileOptions = agentProfiles.map((p: AgentProfileOption) => ({
    id: p.id,
    name: p.label,
  }));
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <span>Workspace Settings</span>
          {isDirty && <UnsavedChangesBadge />}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="workspace-name">Name</Label>
            <Input
              id="workspace-name"
              value={workspaceNameDraft}
              onChange={(e) => onNameChange(e.target.value)}
            />
          </div>
          <SelectField
            label="Default Agent Profile"
            value={defaultAgentProfileId}
            onChange={onAgentProfileChange}
            options={profileOptions}
            emptyLabel="No agent profiles available"
            emptyValue="empty-agent-profiles"
          />
          <div className="flex justify-end pt-2">
            <UnsavedSaveButton
              isDirty={isDirty}
              isLoading={isLoading}
              status={saveStatus}
              onClick={onSave}
            />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

type P4WorkspaceCardProps = {
  isDirty: boolean;
  p4Client: string;
  onP4ClientChange: (value: string) => void;
  p4Root: string;
  p4Stream: string;
  isLoading: boolean;
  saveStatus: "idle" | "loading" | "success" | "error";
  onSave: () => void;
};

// Binds this workspace 1:1 to a local P4 client (workspace). The dropdown is
// populated from the developer's local `p4 clients`; on save the backend
// resolves the client's Root/Stream and a task created in this workspace uses
// that Root as its working directory.
function P4WorkspaceCard({
  isDirty,
  p4Client,
  onP4ClientChange,
  p4Root,
  p4Stream,
  isLoading,
  saveStatus,
  onSave,
}: P4WorkspaceCardProps) {
  const [clients, setClients] = useState<P4Workspace[]>([]);
  const [clientsLoading, setClientsLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setClientsLoading(true);
    void listP4Workspaces()
      .then((resp) => {
        if (!cancelled) setClients(resp.workspaces ?? []);
      })
      .catch(() => {
        if (!cancelled) setClients([]);
      })
      .finally(() => {
        if (!cancelled) setClientsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  // The saved client may not appear in the freshly-listed clients (e.g. p4 not
  // reachable this session); keep it selectable so we don't silently drop it.
  const options = clients.map((c) => ({ id: c.p4client || c.id, name: c.name || c.p4client }));
  if (p4Client && !options.some((o) => o.id === p4Client)) {
    options.unshift({ id: p4Client, name: p4Client });
  }

  // Show the selected client's Root/Stream immediately (before saving) by
  // reading them from the freshly-listed clients. Fall back to the saved
  // values (from the workspace binding) when the client isn't in the list yet
  // — e.g. the saved selection while the list is still loading.
  const selected = clients.find((c) => (c.p4client || c.id) === p4Client);
  const displayRoot = selected?.root_path ?? p4Root;
  const displayStream = selected?.p4stream ?? p4Stream;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <span>P4 Workspace</span>
          {isDirty && <UnsavedChangesBadge />}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>P4 Client</Label>
            <Select
              value={p4Client || "none"}
              onValueChange={(v) => onP4ClientChange(v === "none" ? "" : v)}
              disabled={clientsLoading}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder={clientsLoading ? "Loading P4 clients…" : "Select a P4 client"} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">Not bound</SelectItem>
                {options.map((opt) => (
                  <SelectItem key={opt.id} value={opt.id}>
                    {opt.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              Tasks in this workspace run in the selected client's root directory.
            </p>
          </div>
          {p4Client ? (
            <div className="grid gap-1 text-xs text-muted-foreground">
              <div>
                <span className="font-medium text-foreground">Root:</span>{" "}
                <span className="font-mono">{displayRoot || "—"}</span>
              </div>
              <div>
                <span className="font-medium text-foreground">Stream:</span>{" "}
                <span className="font-mono">{displayStream || "—"}</span>
              </div>
            </div>
          ) : null}
          <div className="flex justify-end pt-2">
            <UnsavedSaveButton
              isDirty={isDirty}
              isLoading={isLoading}
              status={saveStatus}
              onClick={onSave}
            />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

type WorkspaceLinksCardProps = {
  workspaceId: string;
};

function WorkspaceLinksCard({ workspaceId }: WorkspaceLinksCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Workspace Links</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid gap-3">
          <Button asChild variant="outline" className="justify-start gap-2">
            <Link href={`/settings/workspace/${workspaceId}/workflows`}>
              <IconLayoutColumns className="h-4 w-4" />
              Workflows
            </Link>
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

type DeleteWorkspaceCardProps = {
  workspaceName: string;
  deleteDialogOpen: boolean;
  setDeleteDialogOpen: (open: boolean) => void;
  deleteConfirmText: string;
  setDeleteConfirmText: (text: string) => void;
  onDelete: () => void;
};

function DeleteWorkspaceCard({
  workspaceName,
  deleteDialogOpen,
  setDeleteDialogOpen,
  deleteConfirmText,
  setDeleteConfirmText,
  onDelete,
}: DeleteWorkspaceCardProps) {
  return (
    <>
      <Card className="border-destructive">
        <CardHeader>
          <CardTitle className="text-destructive">Delete Workspace</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Delete this workspace</p>
              <p className="text-xs text-muted-foreground">This action cannot be undone.</p>
            </div>
            <Button
              variant="destructive"
              onClick={() => setDeleteDialogOpen(true)}
              className="cursor-pointer"
              data-testid="workspace-settings-delete-button"
            >
              <IconTrash className="h-4 w-4 mr-2" />
              Delete
            </Button>
          </div>
        </CardContent>
      </Card>

      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Workspace</DialogTitle>
            <DialogDescription>
              Type the workspace name <span className="font-medium">{workspaceName}</span> to
              confirm deletion. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="confirm-delete">Confirm Delete</Label>
            <Input
              id="confirm-delete"
              value={deleteConfirmText}
              onChange={(event) => setDeleteConfirmText(event.target.value)}
              placeholder={workspaceName}
              autoComplete="off"
              data-testid="workspace-settings-delete-confirm-input"
            />
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteDialogOpen(false)}
              className="cursor-pointer"
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={onDelete}
              disabled={deleteConfirmText !== workspaceName}
              className="cursor-pointer"
              data-testid="workspace-settings-delete-confirm-button"
            >
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

type SavedState = {
  name: string;
  agentProfileId: string;
  p4Client: string;
};

function buildWorkspaceUpdates(
  draft: { name: string; agentProfileId: string; p4Client: string },
  saved: SavedState,
): Record<string, string | undefined> {
  const updates: Record<string, string | undefined> = {};
  if (draft.name.trim() !== saved.name) updates.name = draft.name.trim();
  if (draft.agentProfileId !== saved.agentProfileId)
    updates.default_agent_profile_id = draft.agentProfileId;
  if (draft.p4Client !== saved.p4Client) updates.p4_client = draft.p4Client;
  return updates;
}

type WorkspaceDraftState = {
  workspaceNameDraft: string;
  defaultAgentProfileId: string;
  p4Client: string;
};

type SaveRequestLike = {
  run: (id: string, updates: Record<string, string | undefined>) => Promise<Workspace>;
};

type WorkspaceSaveHandlerOptions = {
  currentWorkspace: Workspace;
  draft: WorkspaceDraftState;
  savedState: SavedState;
  isDirty: boolean;
  setSavedState: (s: SavedState) => void;
  setCurrentWorkspace: (fn: (prev: Workspace) => Workspace) => void;
  workspaces: Workspace[];
  setWorkspaces: (items: Workspace[]) => void;
  saveWorkspaceRequest: SaveRequestLike;
  toast: ReturnType<typeof useToast>["toast"];
};

function buildSaveHandler({
  currentWorkspace,
  draft,
  savedState,
  isDirty,
  setSavedState,
  setCurrentWorkspace,
  workspaces,
  setWorkspaces,
  saveWorkspaceRequest,
  toast,
}: WorkspaceSaveHandlerOptions) {
  return async () => {
    if (!isDirty) return;
    try {
      const updates = buildWorkspaceUpdates(
        {
          name: draft.workspaceNameDraft,
          agentProfileId: draft.defaultAgentProfileId,
          p4Client: draft.p4Client,
        },
        savedState,
      );
      const updated = await saveWorkspaceRequest.run(currentWorkspace.id, updates);
      setCurrentWorkspace((prev) => ({ ...prev, ...updated }));
      setSavedState({
        name: updated.name ?? draft.workspaceNameDraft.trim(),
        agentProfileId: updated.default_agent_profile_id ?? "",
        p4Client: updated.p4_client ?? "",
      });
      setWorkspaces(
        workspaces.map((ws: Workspace) =>
          ws.id === updated.id
            ? {
                ...ws,
                name: updated.name,
                default_environment_id: updated.default_environment_id ?? null,
                default_agent_profile_id: updated.default_agent_profile_id ?? null,
                p4_client: updated.p4_client ?? "",
                p4_root: updated.p4_root ?? "",
                p4_stream: updated.p4_stream ?? "",
              }
            : ws,
        ),
      );
    } catch (error) {
      toast({
        title: "Failed to save workspace",
        description: error instanceof Error ? error.message : "Request failed",
        variant: "error",
      });
    }
  };
}

function useWorkspaceEditForm(workspace: Workspace) {
  const router = useRouter();
  const { toast } = useToast();
  const [currentWorkspace, setCurrentWorkspace] = useState<Workspace>(workspace);
  const [workspaceNameDraft, setWorkspaceNameDraft] = useState(workspace.name ?? "");
  const [defaultAgentProfileId, setDefaultAgentProfileId] = useState(
    workspace.default_agent_profile_id ?? "",
  );
  const [p4Client, setP4Client] = useState(workspace.p4_client ?? "");
  const [savedState, setSavedState] = useState<SavedState>({
    name: workspace.name ?? "",
    agentProfileId: workspace.default_agent_profile_id ?? "",
    p4Client: workspace.p4_client ?? "",
  });
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteConfirmText, setDeleteConfirmText] = useState("");

  const agentProfiles = useAppStore((state) => state.agentProfiles.items);
  const workspaces = useAppStore((state) => state.workspaces.items);
  const setWorkspaces = useAppStore((state) => state.setWorkspaces);

  const saveWorkspaceRequest = useRequest(updateWorkspaceAction);
  const deleteWorkspaceRequest = useRequest(deleteWorkspaceAction);

  const isDirty =
    workspaceNameDraft.trim() !== savedState.name ||
    defaultAgentProfileId !== savedState.agentProfileId ||
    p4Client !== savedState.p4Client;

  const handleSave = buildSaveHandler({
    currentWorkspace,
    draft: { workspaceNameDraft, defaultAgentProfileId, p4Client },
    savedState,
    isDirty,
    setSavedState,
    setCurrentWorkspace,
    workspaces,
    setWorkspaces,
    saveWorkspaceRequest,
    toast,
  });

  const handleDeleteWorkspace = async () => {
    if (deleteConfirmText !== currentWorkspace.name) return;
    try {
      await deleteWorkspaceRequest.run(currentWorkspace.id, currentWorkspace.name);
      setWorkspaces(workspaces.filter((ws: Workspace) => ws.id !== currentWorkspace.id));
      router.push("/settings/workspace");
    } catch (error) {
      toast({
        title: "Failed to delete workspace",
        description: error instanceof Error ? error.message : "Request failed",
        variant: "error",
      });
    }
  };

  // Clears pre-fill so Cancel-then-reopen can't silently bypass the re-type requirement.
  const handleDeleteDialogOpenChange = (open: boolean) => {
    setDeleteDialogOpen(open);
    if (!open) setDeleteConfirmText("");
  };

  return {
    currentWorkspace,
    workspaceNameDraft,
    setWorkspaceNameDraft,
    defaultAgentProfileId,
    setDefaultAgentProfileId,
    p4Client,
    setP4Client,
    deleteDialogOpen,
    setDeleteDialogOpen: handleDeleteDialogOpenChange,
    deleteConfirmText,
    setDeleteConfirmText,
    agentProfiles,
    isDirty,
    saveWorkspaceRequest,
    handleSave,
    handleDeleteWorkspace,
  };
}

function WorkspaceEditForm({ workspace }: WorkspaceEditFormProps) {
  const {
    currentWorkspace,
    workspaceNameDraft,
    setWorkspaceNameDraft,
    defaultAgentProfileId,
    setDefaultAgentProfileId,
    p4Client,
    setP4Client,
    deleteDialogOpen,
    setDeleteDialogOpen,
    deleteConfirmText,
    setDeleteConfirmText,
    agentProfiles,
    isDirty,
    saveWorkspaceRequest,
    handleSave,
    handleDeleteWorkspace,
  } = useWorkspaceEditForm(workspace);

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-2xl font-bold">{currentWorkspace.name}</h2>
        <p className="text-sm text-muted-foreground mt-1">
          Manage workspace details, its P4 workspace, and workflows.
        </p>
      </div>
      <Separator />
      <WorkspaceSettingsCard
        isDirty={isDirty}
        workspaceNameDraft={workspaceNameDraft}
        onNameChange={setWorkspaceNameDraft}
        defaultAgentProfileId={defaultAgentProfileId}
        onAgentProfileChange={setDefaultAgentProfileId}
        agentProfiles={agentProfiles}
        isLoading={saveWorkspaceRequest.isLoading}
        saveStatus={saveWorkspaceRequest.status}
        onSave={handleSave}
      />
      <P4WorkspaceCard
        isDirty={isDirty}
        p4Client={p4Client}
        onP4ClientChange={setP4Client}
        p4Root={currentWorkspace.p4_root ?? ""}
        p4Stream={currentWorkspace.p4_stream ?? ""}
        isLoading={saveWorkspaceRequest.isLoading}
        saveStatus={saveWorkspaceRequest.status}
        onSave={handleSave}
      />
      <WorkspaceTaskFormCard
        workspaceId={currentWorkspace.id}
        initialConfig={currentWorkspace.task_form_config ?? null}
      />
      <WorkspaceLinksCard workspaceId={currentWorkspace.id} />
      <Separator />
      <DeleteWorkspaceCard
        workspaceName={currentWorkspace.name}
        deleteDialogOpen={deleteDialogOpen}
        setDeleteDialogOpen={setDeleteDialogOpen}
        deleteConfirmText={deleteConfirmText}
        setDeleteConfirmText={setDeleteConfirmText}
        onDelete={handleDeleteWorkspace}
      />
    </div>
  );
}
