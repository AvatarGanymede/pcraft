---
name: release
description: pcraft release & versioning conventions — single SemVer across npm, GitHub release, and Docker. Use when cutting a release, debugging release artifacts, or answering questions about version channels.
---

# Release & Versioning

pcraft uses a **single SemVer** `X.Y.Z` shared across all distribution channels.

## Version targets

- `apps/cli/package.json` version → `X.Y.Z`
- npm main package: `pcraft@X.Y.Z`
- npm runtime packages: `@beilin/runtime-{platform}@X.Y.Z` (5 platforms; declared as `optionalDependencies` in main package)
- Git tag: `vX.Y.Z` (three-part; legacy `vM.m` tags normalize to `M.m.0`)
- GitHub release: `vX.Y.Z` with platform tarballs `pcraft-{platform}.tar.gz` + `.sha256`
- Docker: `ghcr.io/avatarganymede/pcraft:{version,tag,latest,universal}`

## Release flow

Entirely in CI via `.github/workflows/release.yml`, triggered by a maintainer from the GitHub Actions UI:

1. Maintainer clicks "Run workflow" → picks `bump` (patch/minor/major) → optional `dry_run`.
2. `prepare` job bumps version + regenerates CHANGELOG, opens release PR, squash-merges, tags `vX.Y.Z`.
3. `build-web` + `build-bundles` (5 platforms) build the release artifacts.
4. Docker jobs build and promote multi-arch base + universal images.
5. `publish-release` creates the GitHub release with platform tarballs + sha256 + auto-generated notes.
6. `publish-npm` publishes 5 `@beilin/runtime-*` packages + main `pcraft` package to npmjs.

There is no local release script — the entire flow runs in GHA.

## Runtime resolution

The npm shim (`apps/cli/bin/native-shim.js`) locates the bundled runtime via:

1. `PCRAFT_BUNDLE_DIR` env var (used by tests and explicit overrides).
2. Installed `@beilin/runtime-{platform}` npm package via `require.resolve()`.

The native Go launcher (`apps/backend/cmd/pcraft`) is spawned by the npm shim for `run` / `start` / `service` commands.

## npm Trusted Publishers

Before the first release, configure Trusted Publishers on npmjs.com for all 6 packages:
- `pcraft`
- `@beilin/runtime-linux-x64`, `@beilin/runtime-linux-arm64`
- `@beilin/runtime-darwin-x64`, `@beilin/runtime-darwin-arm64`
- `@beilin/runtime-win32-x64`

Each package needs this GitHub Actions workflow (`release.yml`) registered as its trusted publisher.
