import { describe, it, expect } from "vitest";
import type { Executor } from "@/lib/types/http";
import { computeExecutorHint } from "./task-create-dialog-options";

function exec(id: string, type: Executor["type"]): Executor {
  return { id, type, name: id } as Executor;
}

const DOCKER =
  "A Docker container will be created from the selected base branch and checked out on a task branch.";
const LOCAL = "The agent will run directly on the repository.";

describe("computeExecutorHint", () => {
  const executors = [
    exec("loc", "local"),
    exec("docker", "local_docker"),
    exec("remote-docker", "remote_docker"),
  ];

  it("explains that Docker profiles create an isolated task branch", () => {
    expect(computeExecutorHint(executors, "docker", 1)).toBe(DOCKER);
    expect(computeExecutorHint(executors, "remote-docker", 1)).toBe(DOCKER);
  });

  it("returns the local hint regardless of repoCount", () => {
    expect(computeExecutorHint(executors, "loc", 1)).toBe(LOCAL);
    expect(computeExecutorHint(executors, "loc", 5)).toBe(LOCAL);
  });

  it("returns null for an unknown executor id", () => {
    expect(computeExecutorHint(executors, "nope", 1)).toBeNull();
  });

  it("returns null for an unrecognised executor type", () => {
    const odd = [exec("x", "remote" as Executor["type"])];
    expect(computeExecutorHint(odd, "x", 1)).toBeNull();
  });
});
