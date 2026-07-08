import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { useDefaultSelectionsEffect } from "./task-create-dialog-effects";
import { STORAGE_KEYS } from "@/lib/settings/constants";
import type { DialogFormState, StoreSelections } from "@/components/task-create-dialog-types";

beforeEach(() => {
  localStorage.clear();
});

const PROFILE_DOCKER = "profile-docker";
const PROFILE_LOCAL = "profile-local";

type DefaultSelFake = Pick<
  DialogFormState,
  | "agentProfileId"
  | "workflowAgentProfileId"
  | "selectedWorkflowId"
  | "executorId"
  | "executorProfileId"
  | "setAgentProfileId"
  | "setExecutorId"
  | "setExecutorProfileId"
  | "noRepository"
  | "repositories"
  | "remoteRepos"
  | "useRemote"
>;

function makeDefaultSelFs(overrides: Partial<DefaultSelFake> = {}): DialogFormState {
  return {
    agentProfileId: "",
    workflowAgentProfileId: "",
    selectedWorkflowId: null,
    executorId: "exec-1",
    executorProfileId: "profile-1",
    setAgentProfileId: vi.fn(),
    setExecutorId: vi.fn(),
    setExecutorProfileId: vi.fn(),
    noRepository: false,
    repositories: [],
    remoteRepos: [],
    useRemote: false,
    ...overrides,
  } as unknown as DialogFormState;
}

function makeSel(overrides: Partial<StoreSelections> = {}): StoreSelections {
  return {
    agentProfiles: [],
    compatibleAgentProfiles: [],
    authLoaded: true,
    executors: [],
    workspaceDefaults: null,
    ...overrides,
  };
}

function localExecutor(): StoreSelections["executors"][number] {
  return {
    id: "exec-local",
    type: "local",
    profiles: [{ id: PROFILE_LOCAL, executor_type: "local" }],
  } as unknown as StoreSelections["executors"][number];
}

function dockerExecutor(): StoreSelections["executors"][number] {
  return {
    id: "exec-docker",
    type: "local_docker",
    profiles: [{ id: PROFILE_DOCKER, executor_type: "local_docker" }],
  } as unknown as StoreSelections["executors"][number];
}

describe("useDefaultSelectionsEffect - executor profile defaults", () => {
  it("defaults to the local executor and its profile when nothing was saved", async () => {
    const fs = makeDefaultSelFs({ executorId: "", executorProfileId: "" });
    const local = localExecutor();
    const docker = dockerExecutor();
    const sel = makeSel({ executors: [docker, local] });

    renderHook(() => useDefaultSelectionsEffect(fs, true, sel, []));

    await waitFor(() => expect(fs.setExecutorId).toHaveBeenCalledWith(local.id));
    await waitFor(() => expect(fs.setExecutorProfileId).toHaveBeenCalledWith(PROFILE_LOCAL));
  });

  it("falls back to the first executor's profile when there is no local executor", async () => {
    const fs = makeDefaultSelFs({ executorId: "", executorProfileId: "" });
    const docker = dockerExecutor();
    const sel = makeSel({ executors: [docker] });

    renderHook(() => useDefaultSelectionsEffect(fs, true, sel, []));

    await waitFor(() => expect(fs.setExecutorId).toHaveBeenCalledWith(docker.id));
    await waitFor(() => expect(fs.setExecutorProfileId).toHaveBeenCalledWith(PROFILE_DOCKER));
  });

  it("honors the last-used executor profile from localStorage", async () => {
    localStorage.setItem(STORAGE_KEYS.LAST_EXECUTOR_PROFILE_ID, JSON.stringify(PROFILE_DOCKER));
    const fs = makeDefaultSelFs({ executorId: "", executorProfileId: "" });
    const local = localExecutor();
    const docker = dockerExecutor();
    const sel = makeSel({ executors: [local, docker] });

    renderHook(() => useDefaultSelectionsEffect(fs, true, sel, []));

    await waitFor(() => expect(fs.setExecutorProfileId).toHaveBeenCalledWith(PROFILE_DOCKER));
  });

  it("derives executorId from an already-selected executor profile", async () => {
    const fs = makeDefaultSelFs({ executorId: "", executorProfileId: PROFILE_LOCAL });
    const local = localExecutor();
    const sel = makeSel({ executors: [local] });

    renderHook(() => useDefaultSelectionsEffect(fs, true, sel, []));

    await waitFor(() => expect(fs.setExecutorId).toHaveBeenCalledWith(local.id));
  });
});
