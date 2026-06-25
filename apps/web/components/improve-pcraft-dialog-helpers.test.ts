import { describe, it, expect, vi, beforeEach } from "vitest";

import { buildImprovePcraftDescription } from "./improve-pcraft-dialog-helpers";
import type { ImprovePcraftBootstrapResponse } from "@/lib/api/domains/improve-pcraft-api";

const uploadFrontendLog = vi.fn();
vi.mock("@/lib/api/domains/improve-pcraft-api", async (orig) => {
  const actual = await orig<typeof import("@/lib/api/domains/improve-pcraft-api")>();
  return { ...actual, uploadFrontendLog: (...args: unknown[]) => uploadFrontendLog(...args) };
});

vi.mock("@/lib/logger/buffer", () => ({
  snapshotLogs: () => [
    { timestamp: "2026-04-29T10:00:00Z", level: "info", source: "console", message: "hi" },
  ],
}));

const bootstrap: ImprovePcraftBootstrapResponse = {
  repository_id: "r1",
  workflow_id: "w1",
  branch: "main",
  bundle_dir: "/tmp/pcraft-improve-abc",
  bundle_files: {
    metadata: "/tmp/pcraft-improve-abc/metadata.json",
    backend_log: "/tmp/pcraft-improve-abc/backend.log",
    frontend_log: "/tmp/pcraft-improve-abc/frontend.log",
  },
  github_login: "octocat",
  has_write_access: false,
  fork_status: "unknown",
};

describe("buildImprovePcraftDescription", () => {
  beforeEach(() => {
    uploadFrontendLog.mockReset();
    uploadFrontendLog.mockResolvedValue({ path: bootstrap.bundle_files.frontend_log });
  });

  it("returns description unchanged when bootstrap is null", async () => {
    const out = await buildImprovePcraftDescription("desc", null, true);
    expect(out).toBe("desc");
    expect(uploadFrontendLog).not.toHaveBeenCalled();
  });

  it("returns description unchanged when captureLogs is false", async () => {
    const out = await buildImprovePcraftDescription("desc", bootstrap, false);
    expect(out).toBe("desc");
    expect(uploadFrontendLog).not.toHaveBeenCalled();
  });

  it("appends bundle file paths and uploads frontend log when captureLogs=true", async () => {
    const out = await buildImprovePcraftDescription("Original prompt", bootstrap, true);
    expect(out).toContain("Original prompt");
    expect(out).toContain("Context bundle for the agent:");
    expect(out).toContain(bootstrap.bundle_files.metadata);
    expect(out).toContain(bootstrap.bundle_files.backend_log);
    expect(out).toContain(bootstrap.bundle_files.frontend_log);
    expect(uploadFrontendLog).toHaveBeenCalledWith(
      bootstrap.bundle_dir,
      expect.arrayContaining([expect.objectContaining({ message: "hi" })]),
    );
  });

  it("does not abort when frontend log upload fails", async () => {
    uploadFrontendLog.mockRejectedValueOnce(new Error("network down"));
    await expect(buildImprovePcraftDescription("desc", bootstrap, true)).resolves.toContain(
      "Context bundle for the agent:",
    );
  });

  it("omits the frontend_log path when upload fails", async () => {
    uploadFrontendLog.mockRejectedValueOnce(new Error("network down"));
    const out = await buildImprovePcraftDescription("desc", bootstrap, true);
    expect(out).toContain(bootstrap.bundle_files.metadata);
    expect(out).toContain(bootstrap.bundle_files.backend_log);
    expect(out).not.toContain(bootstrap.bundle_files.frontend_log);
  });
});
