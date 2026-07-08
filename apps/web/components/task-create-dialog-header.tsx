"use client";

import { DialogTitle } from "@pcraft/ui/dialog";

export type DialogHeaderContentProps = {
  isCreateMode: boolean;
  isEditMode: boolean;
  sessionRepoName?: string;
  initialTitle?: string;
};

/**
 * Header for the task-create dialog. In create/edit mode the header is
 * intentionally minimal — the dialog itself signals "create" and the
 * repo + branch chips and task-name input live in the body. Session mode
 * shows a breadcrumb (repo / task / new session).
 */
export function DialogHeaderContent({
  isCreateMode,
  isEditMode,
  sessionRepoName,
  initialTitle,
}: DialogHeaderContentProps) {
  if (isCreateMode || isEditMode) {
    return (
      <div className="flex flex-col gap-0.5">
        <DialogTitle className="text-base font-semibold">
          {isEditMode ? "Edit task" : "New task"}
        </DialogTitle>
        <p className="text-xs text-muted-foreground">
          {isEditMode
            ? "Update the task's GUI development configuration"
            : "Fill in the details below to create a new GUI development task"}
        </p>
      </div>
    );
  }
  return (
    <DialogTitle asChild>
      <div className="flex items-center gap-1 min-w-0 text-sm font-medium">
        {sessionRepoName && (
          <>
            <span className="truncate text-muted-foreground">{sessionRepoName}</span>
            <span className="text-muted-foreground mx-0.5">/</span>
          </>
        )}
        <span className="truncate">{initialTitle || "Task"}</span>
        <span className="text-muted-foreground mx-0.5">/</span>
        <span className="text-muted-foreground whitespace-nowrap">new session</span>
      </div>
    </DialogTitle>
  );
}
