import { describe, expect, it } from "vitest";

import {
  buildOrderedWorkflows,
  getSwimlaneEmptyMessage,
  isWorkflowSnapshotPending,
} from "./swimlane-display";

describe("buildOrderedWorkflows", () => {
  const workflows = [
    { id: "wf-a", name: "Alpha" },
    { id: "wf-b", name: "Beta" },
  ];
  const snapshots = {
    "wf-a": { workflowName: "Alpha board" },
  };

  it("keeps the filtered workflow visible while its snapshot is still loading", () => {
    expect(buildOrderedWorkflows("wf-b", workflows, snapshots)).toEqual([
      { id: "wf-b", name: "Beta" },
    ]);
  });

  it("prefers snapshot metadata when the filtered workflow is hydrated", () => {
    expect(buildOrderedWorkflows("wf-a", workflows, snapshots)).toEqual([
      { id: "wf-a", name: "Alpha board" },
    ]);
  });
});

describe("isWorkflowSnapshotPending", () => {
  it("reports pending when a filtered workflow has metadata but no snapshot yet", () => {
    expect(
      isWorkflowSnapshotPending(
        "wf-b",
        {},
        [{ id: "wf-b", name: "Beta" }],
        false,
      ),
    ).toBe(true);
  });
});

describe("getSwimlaneEmptyMessage", () => {
  it("shows loading instead of an empty board while the filtered snapshot is pending", () => {
    const message = getSwimlaneEmptyMessage({
      isLoading: false,
      snapshots: {},
      orderedWorkflows: [{ id: "wf-b", name: "Beta" }],
      workflowFilter: "wf-b",
      getFilteredTasks: () => [],
    });

    expect(message).toBe("Loading...");
  });
});
