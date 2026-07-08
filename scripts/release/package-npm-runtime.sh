#!/usr/bin/env bash
# Generate per-platform npm runtime packages from built dist/release-assets.
#
# Each platform bundle (pcraft-{platform}.tar.gz) is repackaged as an npm package
# containing the native bin/ directory. These are the @beilin/runtime-* packages
# that the main pcraft npm package declares as optionalDependencies.
#
# Usage:
#   package-npm-runtime.sh <version> <release-assets-dir> <output-dir>
#
# Arguments:
#   version            SemVer string (e.g. 0.17.0)
#   release-assets-dir Directory containing pcraft-*.tar.gz files
#   output-dir         Directory where per-platform npm packages are written
#
# Output (one directory per platform ready for npm publish):
#   <output-dir>/@beilin/runtime-linux-x64/
#   <output-dir>/@beilin/runtime-linux-arm64/
#   <output-dir>/@beilin/runtime-darwin-x64/
#   <output-dir>/@beilin/runtime-darwin-arm64/
#   <output-dir>/@beilin/runtime-win32-x64/
set -euo pipefail

VERSION="${1:?Usage: $0 <version> <release-assets-dir> <output-dir>}"
ASSETS_DIR="${2:?Usage: $0 <version> <release-assets-dir> <output-dir>}"
OUT_DIR="${3:?Usage: $0 <version> <release-assets-dir> <output-dir>}"

# Maps platform dir name → npm package name + npm os/cpu fields
declare -A PLATFORM_TO_PACKAGE=(
  ["linux-x64"]="@beilin/runtime-linux-x64"
  ["linux-arm64"]="@beilin/runtime-linux-arm64"
  ["macos-x64"]="@beilin/runtime-darwin-x64"
  ["macos-arm64"]="@beilin/runtime-darwin-arm64"
  ["windows-x64"]="@beilin/runtime-win32-x64"
)

declare -A PLATFORM_TO_OS=(
  ["linux-x64"]='["linux"]'
  ["linux-arm64"]='["linux"]'
  ["macos-x64"]='["darwin"]'
  ["macos-arm64"]='["darwin"]'
  ["windows-x64"]='["win32"]'
)

declare -A PLATFORM_TO_CPU=(
  ["linux-x64"]='["x64"]'
  ["linux-arm64"]='["arm64"]'
  ["macos-x64"]='["x64"]'
  ["macos-arm64"]='["arm64"]'
  ["windows-x64"]='["x64"]'
)

echo "Packaging npm runtime packages for version $VERSION..."
echo "  assets dir: $ASSETS_DIR"
echo "  output dir: $OUT_DIR"

for platform in linux-x64 linux-arm64 macos-x64 macos-arm64 windows-x64; do
  archive="$ASSETS_DIR/pcraft-${platform}.tar.gz"
  if [[ ! -f "$archive" ]]; then
    echo "Error: missing archive $archive" >&2
    exit 1
  fi

  package_name="${PLATFORM_TO_PACKAGE[$platform]}"
  scope_dir="${package_name%%/*}"
  pkg_dir="${package_name##*/}"
  pkg_out="$OUT_DIR/${scope_dir}/${pkg_dir}"

  rm -rf "$pkg_out"
  mkdir -p "$pkg_out"

  local_tmp="$pkg_out/.extract_tmp"
  mkdir -p "$local_tmp"
  tar -xzf "$archive" -C "$local_tmp"

  bundle_root="$local_tmp/pcraft"
  if [[ ! -d "$bundle_root" ]]; then
    echo "Error: expected pcraft/ directory in $archive" >&2
    exit 1
  fi

  cp -R "$bundle_root/bin" "$pkg_out/bin"
  rm -rf "$local_tmp"

  os_field="${PLATFORM_TO_OS[$platform]}"
  cpu_field="${PLATFORM_TO_CPU[$platform]}"

  cat > "$pkg_out/package.json" <<EOF
{
  "name": "$package_name",
  "version": "$VERSION",
  "description": "pcraft runtime bundle for $platform",
  "license": "AGPL-3.0-only",
  "repository": {
    "type": "git",
    "url": "git+https://github.com/AvatarGanymede/pcraft.git"
  },
  "homepage": "https://github.com/AvatarGanymede/pcraft",
  "os": $os_field,
  "cpu": $cpu_field,
  "files": [
    "bin"
  ]
}
EOF

  echo "  packaged $package_name@$VERSION"
done

echo "Done. Runtime packages written to $OUT_DIR"
