#!/usr/bin/env bash
# ensure-binary.sh — Install or update the sl binary.
# Compares plugin.json version against installed version; updates when outdated.
# Fast no-op when binary exists and version matches. No network call for checks.

# Defense-in-depth: remove any recursive solon/ nesting in plugin cache.
if [ -n "${CLAUDE_PLUGIN_ROOT:-}" ] && [ -d "${CLAUDE_PLUGIN_ROOT}/solon" ]; then
  rm -rf "${CLAUDE_PLUGIN_ROOT}/solon"
fi

INSTALL_DIR="${HOME}/.solon/bin"
REPO="HuynhHoangPhuc/solon"
VERSION_FILE="${INSTALL_DIR}/.version"

# Read expected version from plugin.json (no network needed)
PLUGIN_JSON="${CLAUDE_PLUGIN_ROOT:-.}/.claude-plugin/plugin.json"
if [ -f "${PLUGIN_JSON}" ]; then
  EXPECTED_VERSION=$(grep '"version"' "${PLUGIN_JSON}" | head -1 | sed 's/.*"version": *"\([^"]*\)".*/\1/')
else
  # Fallback: try parent directory structure
  SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
  PLUGIN_JSON="${SCRIPT_DIR}/../.claude-plugin/plugin.json"
  if [ -f "${PLUGIN_JSON}" ]; then
    EXPECTED_VERSION=$(grep '"version"' "${PLUGIN_JSON}" | head -1 | sed 's/.*"version": *"\([^"]*\)".*/\1/')
  fi
fi

# Prefix with v if needed (release tags use v-prefix)
if [ -n "${EXPECTED_VERSION}" ] && [ "${EXPECTED_VERSION#v}" = "${EXPECTED_VERSION}" ]; then
  EXPECTED_VERSION="v${EXPECTED_VERSION}"
fi

# Detect OS
case "$(uname -s)" in
  Linux*)             OS="linux";   EXT="" ;;
  Darwin*)            OS="darwin";  EXT="" ;;
  MINGW*|MSYS*|CYGWIN*) OS="windows"; EXT=".exe" ;;
  *) exit 0 ;;
esac

# Detect arch
case "$(uname -m)" in
  x86_64|amd64) ARCH="x64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) exit 0 ;;
esac

SL_BIN="${INSTALL_DIR}/sl${EXT}"

mkdir -p "${INSTALL_DIR}"

# Download helper
_download() {
  local url="$1" dest="$2"
  if command -v curl &>/dev/null; then
    curl -fsSL -o "${dest}" "${url}" 2>/dev/null
  elif command -v wget &>/dev/null; then
    wget -q -O "${dest}" "${url}" 2>/dev/null
  else
    echo "[solon] Error: curl or wget required" >&2
    return 1
  fi
}

# Verify checksum (skips silently if unavailable)
_verify() {
  local bin="$1" url="$2"
  local chk="${bin}.sha256"
  _download "${url}.sha256" "${chk}" 2>/dev/null || return 0
  [ -s "${chk}" ] || { rm -f "${chk}"; return 0; }
  local expected actual
  expected=$(awk '{print $1}' "${chk}")
  if command -v sha256sum &>/dev/null; then
    actual=$(sha256sum "${bin}" | awk '{print $1}')
  elif command -v shasum &>/dev/null; then
    actual=$(shasum -a 256 "${bin}" | awk '{print $1}')
  else
    rm -f "${chk}"; return 0
  fi
  rm -f "${chk}"
  if [ "${expected}" != "${actual}" ]; then
    echo "[solon] Checksum mismatch for $(basename "${bin}")" >&2
    rm -f "${bin}"
    return 1
  fi
}

# Download and install a single binary from a specific release tag
_install_binary() {
  local name="$1" dest="$2" tag="$3"
  local release_name="${name}-${OS}-${ARCH}${EXT}"
  local url="https://github.com/${REPO}/releases/download/${tag}/${release_name}"
  local tmp="${dest}.tmp"

  _download "${url}" "${tmp}" || { rm -f "${tmp}"; return 1; }
  _verify "${tmp}" "${url}" || return 1
  mv "${tmp}" "${dest}"
  [ "${OS}" != "windows" ] && chmod +x "${dest}"
}

# --- Main logic ---

LOCAL_VERSION=""
[ -f "${VERSION_FILE}" ] && LOCAL_VERSION=$(cat "${VERSION_FILE}" 2>/dev/null)

ALL_EXIST=true
[ -x "${SL_BIN}" ] || ALL_EXIST=false

# Determine target version: plugin.json (local) or GitHub API (fallback)
TARGET="${EXPECTED_VERSION}"
if [ -z "${TARGET}" ]; then
  if command -v curl &>/dev/null; then
    TARGET=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
      | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
  elif command -v wget &>/dev/null; then
    TARGET=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
      | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
  fi
fi

if [ -z "${TARGET}" ]; then
  [ "${ALL_EXIST}" = true ] && exit 0
  echo "[solon] Error: could not determine target version" >&2
  exit 1
fi

# Fast no-op: binary exists and version matches
if [ "${ALL_EXIST}" = true ] && [ "${LOCAL_VERSION}" = "${TARGET}" ]; then
  exit 0
fi

if [ "${ALL_EXIST}" = true ] && [ -n "${LOCAL_VERSION}" ]; then
  echo "[solon] Updating sl binary: ${LOCAL_VERSION} -> ${TARGET}" >&2
fi

failed=0
_install_binary "sl" "${SL_BIN}" "${TARGET}" || failed=1

if [ "${failed}" -eq 0 ]; then
  echo "${TARGET}" > "${VERSION_FILE}"
fi

exit ${failed}
