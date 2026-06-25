"use client";

import { memo, useCallback, useEffect, useState } from "react";
import { Button } from "@pcraft/ui/button";
import { closeTaskAfterP4Submit, fetchTaskP4Opened } from "@/lib/api/domains/p4-api";
import type { TaskState } from "@/lib/types/http";
import { useToast } from "@/components/toast-provider";

export type P4TaskRef = {
  id: string;
  p4_changelist?: string;
  state?: TaskState;
};

type P4ChangesPanelProps = {
  task: P4TaskRef | null | undefined;
  onClosed?: () => void;
};

export const P4ChangesPanel = memo(function P4ChangesPanel({ task, onClosed }: P4ChangesPanelProps) {
  const { toast } = useToast();
  const [files, setFiles] = useState<string[]>([]);
  const [changelist, setChangelist] = useState("");
  const [loading, setLoading] = useState(false);
  const [confirming, setConfirming] = useState(false);

  const refresh = useCallback(async () => {
    if (!task?.id) return;
    setLoading(true);
    try {
      const data = await fetchTaskP4Opened(task.id);
      setFiles(data.files ?? []);
      setChangelist(data.changelist || task.p4_changelist || "");
    } catch (err) {
      toast({ title: "P4 opened files", description: String(err), variant: "error" });
    } finally {
      setLoading(false);
    }
  }, [task?.id, task?.p4_changelist, toast]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  if (!task) {
    return null;
  }

  const canConfirmClose = task.state === "DONE";

  const handleConfirmSubmit = async () => {
    if (!task.id) return;
    setConfirming(true);
    try {
      await closeTaskAfterP4Submit(task.id, { p4_changelist: changelist || undefined });
      toast({ title: "Task closed", description: "P4 changelist verified and locks released." });
      onClosed?.();
    } catch (err) {
      toast({
        title: "Submit not verified",
        description: String(err),
        variant: "error",
      });
    } finally {
      setConfirming(false);
    }
  };

  return (
    <div className="flex h-full flex-col gap-3 p-3" data-testid="p4-changes-panel">
      <div className="flex items-center justify-between gap-2">
        <div>
          <div className="text-sm font-medium">P4 Changes</div>
          <div className="text-xs text-muted-foreground">
            Changelist {changelist || task.p4_changelist || "—"}
          </div>
        </div>
        <Button variant="outline" size="sm" className="cursor-pointer" onClick={() => void refresh()} disabled={loading}>
          Refresh
        </Button>
      </div>

      {task.state === "DONE" ? (
        <p className="text-xs text-muted-foreground">
          Pipeline complete. Submit changelist #{changelist || task.p4_changelist || "?"} in your P4 client, then confirm below.
        </p>
      ) : null}

      <ul className="min-h-0 flex-1 space-y-1 overflow-y-auto text-xs">
        {files.length === 0 ? (
          <li className="text-muted-foreground">{loading ? "Loading…" : "No pending files"}</li>
        ) : (
          files.map((f) => (
            <li key={f} className="truncate font-mono">
              {f}
            </li>
          ))
        )}
      </ul>

      {canConfirmClose ? (
        <Button
          className="cursor-pointer"
          onClick={() => void handleConfirmSubmit()}
          disabled={confirming}
          data-testid="p4-confirm-submitted"
        >
          {confirming ? "Verifying…" : "I've submitted in P4"}
        </Button>
      ) : null}
    </div>
  );
});
