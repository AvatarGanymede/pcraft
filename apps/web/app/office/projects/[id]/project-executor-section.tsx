"use client";

import { useState, useCallback } from "react";
import { Button } from "@pcraft/ui/button";
import { Label } from "@pcraft/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@pcraft/ui/select";
import { IconDeviceFloppy } from "@tabler/icons-react";
import { toast } from "sonner";
import { updateProject } from "@/lib/api/domains/office-api";
import { useAppStore } from "@/components/state-provider";
import type { Project } from "@/lib/state/slices/office/types";

type ProjectExecutorSectionProps = {
  project: Project;
};

function ExecutorTypeSelect({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  return (
    <div className="space-y-1">
      <Label className="text-xs">Type</Label>
      <Select value={value || "inherit"} onValueChange={(v) => onChange(v === "inherit" ? "" : v)}>
        <SelectTrigger className="cursor-pointer">
          <SelectValue placeholder="Inherit from workspace" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="inherit" className="cursor-pointer">
            Inherit from workspace
          </SelectItem>
          <SelectItem value="local_pc" className="cursor-pointer">
            Local (standalone)
          </SelectItem>
        </SelectContent>
      </Select>
    </div>
  );
}

export function ProjectExecutorSection({ project }: ProjectExecutorSectionProps) {
  const updateProjectStore = useAppStore((s) => s.updateProject);
  const config = project.executorConfig ?? {};

  const [executorType, setExecutorType] = useState((config.type as string) ?? "");
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);

  const handleSave = useCallback(async () => {
    setSaving(true);
    try {
      const newConfig: Record<string, unknown> = {};
      if (executorType) newConfig.type = executorType;
      await updateProject(project.id, { executorConfig: newConfig });
      updateProjectStore(project.id, { executorConfig: newConfig });
      setDirty(false);
      toast.success("Executor configuration saved");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save executor configuration");
    } finally {
      setSaving(false);
    }
  }, [executorType, project.id, updateProjectStore]);

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-semibold">Executor Configuration</h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            How agent sessions run for this project.
          </p>
        </div>
        {dirty && (
          <Button
            size="sm"
            variant="outline"
            onClick={handleSave}
            disabled={saving}
            className="cursor-pointer"
          >
            <IconDeviceFloppy className="h-3.5 w-3.5 mr-1" />
            Save
          </Button>
        )}
      </div>

      <div className="space-y-3">
        <ExecutorTypeSelect
          value={executorType}
          onChange={(v) => {
            setExecutorType(v);
            setDirty(true);
          }}
        />
      </div>
    </div>
  );
}
