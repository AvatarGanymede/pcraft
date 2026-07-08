#!/usr/bin/env bash
# Publish the main @beilin/pcraft npm package + all @beilin/runtime-* optional packages.
#
# Authentication: Trusted Publishers (OIDC). Each of the 6 packages must have
# this workflow configured as its trusted publisher on npmjs.com. The npm CLI
# auto-detects OIDC credentials from GitHub Actions and exchanges them for a
# short-lived publish token. No NPM_TOKEN secret is needed.
#
# Prerequisites:
#   - GitHub release assets for <tag> must already exist (verified before publishing).
#   - Running inside GitHub Actions with `id-token: write` permission set on
#     the publish-npm job.
#
# Usage:
#   publish-npm.sh <version> <tag>
#
# Arguments:
#   version  SemVer string (e.g. 0.17.0)
#   tag      Git tag (e.g. v0.17.0) — used to verify GitHub release assets exist
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MAIN_PACKAGE="$(node -e "console.log(require('${ROOT_DIR}/apps/cli/package.json').name)")"

VERSION="${1:?Usage: $0 <version> <tag>}"
TAG="${2:?Usage: $0 <version> <tag>}"

bold()  { printf '\033[1m%s\033[0m' "$*"; }
green() { printf '\033[32m%s\033[0m' "$*"; }
red()   { printf '\033[31m%s\033[0m' "$*"; }
yellow(){ printf '\033[33m%s\033[0m' "$*"; }

log()    { echo "  >> $*"; }
log_ok() { echo "  $(green "ok") $*"; }

package_already_published() {
  local pkg="$1"
  npm view "${pkg}@${VERSION}" version --silent >/dev/null 2>&1
}

record_already_published() {
  local pkg="$1"
  echo "  $(yellow "skip") $pkg@$VERSION already published (treated as idempotent success)" >&2
  ALREADY_PUBLISHED+=("$pkg")
}

die() {
  echo "$(red "Error:") $*" >&2
  exit 1
}

REQUIRED_PLATFORMS=(linux-x64 linux-arm64 macos-x64 macos-arm64 windows-x64)

log "Verifying GitHub release assets exist for $TAG..."
for platform in "${REQUIRED_PLATFORMS[@]}"; do
  asset="pcraft-${platform}.tar.gz"
  if ! gh release view "$TAG" --json assets --jq ".assets[].name" 2>/dev/null | grep -q "^${asset}$"; then
    die "GitHub release asset missing: $asset in release $TAG. Run release workflow first."
  fi
done
log_ok "All 5 platform assets present in GitHub release $TAG"

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT
ASSETS_DIR="$WORK_DIR/assets"
mkdir -p "$ASSETS_DIR"

log "Downloading release assets for $TAG..."
for platform in "${REQUIRED_PLATFORMS[@]}"; do
  asset="pcraft-${platform}.tar.gz"
  log "  downloading $asset..."
  gh release download "$TAG" --pattern "$asset" --dir "$ASSETS_DIR"
done
log_ok "Assets downloaded to $ASSETS_DIR"

NPM_PKG_DIR="$WORK_DIR/npm-packages"
bash "$ROOT_DIR/scripts/release/package-npm-runtime.sh" "$VERSION" "$ASSETS_DIR" "$NPM_PKG_DIR"

RUNTIME_PACKAGES=(
  "@beilin/runtime-linux-x64"
  "@beilin/runtime-linux-arm64"
  "@beilin/runtime-darwin-x64"
  "@beilin/runtime-darwin-arm64"
  "@beilin/runtime-win32-x64"
)

echo
echo "$(bold "Publishing @beilin/runtime-* packages...")"
FAILED_PACKAGES=()
ALREADY_PUBLISHED=()

for pkg in "${RUNTIME_PACKAGES[@]}"; do
  scope="${pkg%%/*}"
  name="${pkg##*/}"
  pkg_dir="$NPM_PKG_DIR/${scope}/${name}"

  if [[ ! -d "$pkg_dir" ]]; then
    echo "  $(red "missing") $pkg_dir (package directory was not generated)" >&2
    FAILED_PACKAGES+=("$pkg")
    continue
  fi

  if package_already_published "$pkg"; then
    record_already_published "$pkg"
    continue
  fi

  log "Publishing $pkg@$VERSION..."
  if output="$(cd "$pkg_dir" && npm publish --access public --provenance 2>&1)"; then
    log_ok "$pkg@$VERSION published"
  elif echo "$output" | grep -qE "EPUBLISHCONFLICT|cannot publish over the previously published versions|You cannot publish over"; then
    record_already_published "$pkg"
  else
    echo "  $(red "FAIL") Failed to publish $pkg@$VERSION:" >&2
    echo "$output" | sed 's/^/      /' >&2
    FAILED_PACKAGES+=("$pkg")
  fi
done

if [[ "${#FAILED_PACKAGES[@]}" -gt 0 ]]; then
  echo
  echo "$(red "Error:") The following runtime packages failed to publish:" >&2
  for pkg in "${FAILED_PACKAGES[@]}"; do
    echo "  - $pkg" >&2
  done
  echo >&2
  echo "Refusing to publish main ${MAIN_PACKAGE}@$VERSION. Fix the runtime publish failures" >&2
  echo "and re-run this script (already-published runtime packages will be skipped)." >&2
  exit 1
fi

log "Pinning optionalDependencies to $VERSION before publishing main package..."
node -e "
  const fs = require('fs');
  const path = '$ROOT_DIR/apps/cli/package.json';
  const pkg = JSON.parse(fs.readFileSync(path, 'utf8'));
  if (pkg.optionalDependencies) {
    for (const k of Object.keys(pkg.optionalDependencies)) {
      pkg.optionalDependencies[k] = '$VERSION';
    }
  }
  fs.writeFileSync(path, JSON.stringify(pkg, null, 2) + '\n');
"
log_ok "optionalDependencies pinned to $VERSION"

echo
echo "$(bold "Publishing ${MAIN_PACKAGE}@$VERSION...")"
if package_already_published "$MAIN_PACKAGE"; then
  record_already_published "$MAIN_PACKAGE"
elif main_output="$(cd "$ROOT_DIR/apps/cli" && npm publish --access public --provenance 2>&1)"; then
  log_ok "${MAIN_PACKAGE}@$VERSION published"
elif echo "$main_output" | grep -qE "EPUBLISHCONFLICT|cannot publish over the previously published versions|You cannot publish over"; then
  record_already_published "$MAIN_PACKAGE"
else
  echo "  $(red "FAIL") Failed to publish ${MAIN_PACKAGE}@$VERSION:" >&2
  echo "$main_output" | sed 's/^/      /' >&2
  exit 1
fi

echo
echo "$(green "$(bold "All npm packages published successfully!")")"
if [[ "${#ALREADY_PUBLISHED[@]}" -gt 0 ]]; then
  echo "  $(yellow "note") The following npm packages were already published at $VERSION:"
  for pkg in "${ALREADY_PUBLISHED[@]}"; do
    echo "    - $pkg"
  done
fi
