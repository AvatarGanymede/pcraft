type WorkflowListItem = { id: string; name: string };

type SnapshotMeta = { workflowName: string };

export function buildOrderedWorkflows(
  workflowFilter: string | null | undefined,
  workflows: WorkflowListItem[],
  snapshots: Record<string, SnapshotMeta>,
): WorkflowListItem[] {
  if (workflowFilter) {
    const snapshot = snapshots[workflowFilter];
    if (snapshot) {
      return [{ id: workflowFilter, name: snapshot.workflowName }];
    }
    const workflow = workflows.find((wf) => wf.id === workflowFilter);
    if (workflow) {
      return [{ id: workflowFilter, name: workflow.name }];
    }
    return [];
  }
  return workflows
    .filter((wf) => snapshots[wf.id])
    .map((wf) => ({
      id: wf.id,
      name: snapshots[wf.id]?.workflowName ?? wf.name,
    }));
}

export function isWorkflowSnapshotPending(
  workflowFilter: string | null | undefined,
  snapshots: Record<string, unknown>,
  orderedWorkflows: WorkflowListItem[],
  isLoading: boolean,
): boolean {
  if (isLoading) return true;
  if (!workflowFilter) return false;
  return orderedWorkflows.some((wf) => wf.id === workflowFilter && !snapshots[workflowFilter]);
}

export function getSwimlaneEmptyMessage({
  isLoading,
  snapshots,
  orderedWorkflows,
  workflowFilter,
  getFilteredTasks,
}: {
  isLoading: boolean;
  snapshots: Record<string, unknown>;
  orderedWorkflows: WorkflowListItem[];
  workflowFilter: string | null | undefined;
  getFilteredTasks: (id: string) => unknown[];
}): string | null {
  if (isWorkflowSnapshotPending(workflowFilter, snapshots, orderedWorkflows, isLoading)) {
    return "Loading...";
  }
  if (orderedWorkflows.length === 0) return "No workflows available yet.";
  const visible = workflowFilter
    ? orderedWorkflows
    : orderedWorkflows.filter((wf) => getFilteredTasks(wf.id).length > 0);
  if (visible.length === 0) return "No tasks yet";
  return null;
}
