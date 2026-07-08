import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { afterEach, beforeEach, describe, it } from "node:test";
import assert from "node:assert/strict";

import shim from "./native-shim.js";

const { binaryName, platformPackage, resolveRuntime, validateRuntime } = shim;

function createBundle(dir) {
  const launcher = process.platform === "win32" ? "pcraft.exe" : "pcraft";
  const agentctl = process.platform === "win32" ? "agentctl.exe" : "agentctl";
  fs.mkdirSync(path.join(dir, "bin"), { recursive: true });
  fs.writeFileSync(path.join(dir, "bin", launcher), "fake");
  fs.writeFileSync(path.join(dir, "bin", agentctl), "fake");
}

describe("native npm shim", () => {
  let tmpDir;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "pcraft-native-shim-"));
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  it("maps supported platforms to runtime packages", () => {
    assert.equal(platformPackage("linux", "x64"), "@beilin/runtime-linux-x64");
    assert.equal(platformPackage("darwin", "arm64"), "@beilin/runtime-darwin-arm64");
    assert.equal(platformPackage("win32", "x64"), "@beilin/runtime-win32-x64");
  });

  it("uses exe suffix on Windows only", () => {
    assert.equal(binaryName("pcraft", "win32"), "pcraft.exe");
    assert.equal(binaryName("pcraft", "linux"), "pcraft");
  });

  it("resolves PCRAFT_BUNDLE_DIR before npm packages", () => {
    createBundle(tmpDir);

    const runtime = resolveRuntime({ PCRAFT_BUNDLE_DIR: tmpDir }, () => {
      throw new Error("should not resolve npm package");
    });

    assert.equal(runtime.bundleDir, tmpDir);
    assert.equal(runtime.executable, path.join(tmpDir, "bin", process.platform === "win32" ? "pcraft.exe" : "pcraft"));
  });

  it("resolves the installed runtime package", () => {
    createBundle(tmpDir);
    const pkgJSON = path.join(tmpDir, "package.json");
    fs.writeFileSync(pkgJSON, "{}");

    const runtime = resolveRuntime({}, () => pkgJSON);

    assert.equal(runtime.bundleDir, tmpDir);
  });

  it("rejects bundles without the native pcraft binary", () => {
    fs.mkdirSync(path.join(tmpDir, "bin"), { recursive: true });
    fs.writeFileSync(path.join(tmpDir, "bin", "agentctl"), "fake");

    assert.throws(() => validateRuntime(tmpDir), /pcraft native binary not found/);
  });
});
