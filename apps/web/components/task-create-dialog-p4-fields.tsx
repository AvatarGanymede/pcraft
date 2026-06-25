"use client";

import { useEffect, useState } from "react";
import { Input } from "@pcraft/ui/input";
import { Label } from "@pcraft/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@pcraft/ui/select";
import { listP4Workspaces, type P4Workspace } from "@/lib/api/domains/p4-api";

export type P4TaskFormValues = {
  p4WorkspaceId: string;
  panelId: string;
  requirement: string;
  prefabPath: string;
};

type P4TaskCreateFieldsProps = {
  values: P4TaskFormValues;
  onChange: (patch: Partial<P4TaskFormValues>) => void;
  disabled?: boolean;
};

export function P4WorkspaceSelect({ values, onChange, disabled }: P4TaskCreateFieldsProps) {
  const [workspaces, setWorkspaces] = useState<P4Workspace[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    void listP4Workspaces()
      .then((resp) => {
        if (!cancelled) setWorkspaces(resp.workspaces ?? []);
      })
      .catch(() => {
        if (!cancelled) setWorkspaces([]);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="grid gap-1.5">
      <Label htmlFor="p4-workspace">P4 Workspace</Label>
      <Select
        value={values.p4WorkspaceId}
        onValueChange={(v) => onChange({ p4WorkspaceId: v })}
        disabled={disabled || loading}
      >
        <SelectTrigger id="p4-workspace" className="cursor-pointer">
          <SelectValue placeholder={loading ? "加载 P4 clients…" : "选择 P4 client"} />
        </SelectTrigger>
        <SelectContent>
          {workspaces.map((ws) => (
            <SelectItem key={ws.id} value={ws.id} className="cursor-pointer">
              {ws.name || ws.p4client}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

export function P4TaskDetailFields({ values, onChange, disabled }: P4TaskCreateFieldsProps) {
  return (
    <>
      <div className="grid gap-1.5">
        <Label htmlFor="panel-id">panelId</Label>
        <Input
          id="panel-id"
          value={values.panelId}
          onChange={(e) => onChange({ panelId: e.target.value })}
          disabled={disabled}
          placeholder="例如 MainMenuPanel"
        />
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="requirement">需求描述</Label>
        <Input
          id="requirement"
          value={values.requirement}
          onChange={(e) => onChange({ requirement: e.target.value })}
          disabled={disabled}
          placeholder="简要描述本次 GUI 开发需求"
        />
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="prefab-path">Prefab 路径（可选）</Label>
        <Input
          id="prefab-path"
          value={values.prefabPath}
          onChange={(e) => onChange({ prefabPath: e.target.value })}
          disabled={disabled}
          placeholder="Assets/..."
        />
      </div>
    </>
  );
}

export function P4TaskCreateFields({ values, onChange, disabled }: P4TaskCreateFieldsProps) {
  const [workspaces, setWorkspaces] = useState<P4Workspace[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    void listP4Workspaces()
      .then((resp) => {
        if (!cancelled) setWorkspaces(resp.workspaces ?? []);
      })
      .catch(() => {
        if (!cancelled) setWorkspaces([]);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="grid gap-3" data-testid="p4-task-create-fields">
      <div className="grid gap-1.5">
        <Label htmlFor="p4-workspace">P4 Workspace</Label>
        <Select
          value={values.p4WorkspaceId}
          onValueChange={(v) => onChange({ p4WorkspaceId: v })}
          disabled={disabled || loading}
        >
          <SelectTrigger id="p4-workspace" className="cursor-pointer">
            <SelectValue placeholder={loading ? "加载 P4 clients…" : "选择 P4 client"} />
          </SelectTrigger>
          <SelectContent>
            {workspaces.map((ws) => (
              <SelectItem key={ws.id} value={ws.id} className="cursor-pointer">
                {ws.name || ws.p4client}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="panel-id">panelId</Label>
        <Input
          id="panel-id"
          value={values.panelId}
          onChange={(e) => onChange({ panelId: e.target.value })}
          disabled={disabled}
          placeholder="例如 MainMenuPanel"
        />
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="requirement">需求描述</Label>
        <Input
          id="requirement"
          value={values.requirement}
          onChange={(e) => onChange({ requirement: e.target.value })}
          disabled={disabled}
          placeholder="简要描述本次 GUI 开发需求"
        />
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="prefab-path">Prefab 路径（可选）</Label>
        <Input
          id="prefab-path"
          value={values.prefabPath}
          onChange={(e) => onChange({ prefabPath: e.target.value })}
          disabled={disabled}
          placeholder="Assets/..."
        />
      </div>
    </div>
  );
}

export function isP4TaskFormValid(values: P4TaskFormValues, title: string): boolean {
  return (
    title.trim() !== "" &&
    values.p4WorkspaceId.trim() !== "" &&
    values.panelId.trim() !== "" &&
    values.requirement.trim() !== ""
  );
}
