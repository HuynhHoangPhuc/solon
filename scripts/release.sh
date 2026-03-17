#!/usr/bin/env bash
# release.sh — Build sl binary and create a GitHub release
# Usage: ./scripts/release.sh [--dry-run]
#
# Reads version from marketplace.json, builds sl for all platforms,
# then creates a GitHub release with all artifacts.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MARKETPLACE_JSON="${ROOT}/.claude-plugin/marketplace.json"
DRY_RUN=false

[ "${1:-}" = "--dry-run" ] && DRY_RUN=true

# --- Read version ---
VERSION=$(grep '"version"' "${MARKETPLACE_JSON}" | head -1 | sed 's/.*"version": *"\([^"]*\)".*/\1/')
if [ -z "${VERSION}" ]; then
  echo "Error: could not read version from ${MARKETPLACE_JSON}" >&2
  exit 1
fi
TAG="v${VERSION}"
echo "Building release ${TAG}..."

DIST="${ROOT}/dist"
rm -rf "${DIST}"
mkdir -p "${DIST}"

# --- Build sl (Rust) ---
echo "Building sl..."
cd "${ROOT}"
# Ensure Cargo.toml version matches
CARGO_VERSION=$(grep '^version' Cargo.toml | head -1 | sed 's/.*"\(.*\)".*/\1/')
if [ "${CARGO_VERSION}" != "${VERSION}" ]; then
  echo "Updating Cargo.toml version: ${CARGO_VERSION} -> ${VERSION}"
  sed -i '' "s/^version = \".*\"/version = \"${VERSION}\"/" Cargo.toml
fi

TARGETS=(
  "aarch64-apple-darwin:darwin-arm64"
  "x86_64-apple-darwin:darwin-x64"
  "x86_64-unknown-linux-gnu:linux-x64"
  "aarch64-unknown-linux-gnu:linux-arm64"
)

for entry in "${TARGETS[@]}"; do
  target="${entry%%:*}"
  suffix="${entry##*:}"
  echo "  sl-${suffix}..."
  if cargo build --release --target "${target}" 2>/dev/null; then
    cp "target/${target}/release/sl" "${DIST}/sl-${suffix}"
  else
    echo "  Skipping sl-${suffix} (cross-compile target not installed)"
  fi
done

# Windows
if cargo build --release --target x86_64-pc-windows-gnu 2>/dev/null; then
  cp "target/x86_64-pc-windows-gnu/release/sl.exe" "${DIST}/sl-windows-x64.exe"
else
  echo "  Skipping sl-windows-x64 (cross-compile target not installed)"
fi

# --- Generate checksums ---
echo "Generating checksums..."
cd "${DIST}"
for f in *; do
  [ -f "$f" ] || continue
  if command -v sha256sum &>/dev/null; then
    sha256sum "$f" > "$f.sha256"
  elif command -v shasum &>/dev/null; then
    shasum -a 256 "$f" > "$f.sha256"
  fi
done

echo ""
echo "Artifacts in ${DIST}/:"
ls -lh "${DIST}/"

# --- Create GitHub release ---
if [ "${DRY_RUN}" = true ]; then
  echo ""
  echo "[dry-run] Would create release ${TAG} with artifacts above"
  exit 0
fi

echo ""
echo "Creating GitHub release ${TAG}..."
cd "${ROOT}"
gh release create "${TAG}" "${DIST}"/* \
  --title "solon ${TAG}" \
  --generate-notes

echo "Release ${TAG} created successfully."
