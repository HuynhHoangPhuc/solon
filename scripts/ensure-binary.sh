#!/usr/bin/env bash
# ensure-binary.sh — Idempotent: download sl + solon-hooks to ~/.solon/bin/ on first run
# Fast no-op when both binaries already exist

# Defense-in-depth: remove any recursive solon/ nesting in plugin cache.
# Root cause fixed in marketplace.json (source: "./" prevents re-cloning).
if [ -n "${CLAUDE_PLUGIN_ROOT:-}" ] && [ -d "${CLAUDE_PLUGIN_ROOT}/solon" ]; then
  rm -rf "${CLAUDE_PLUGIN_ROOT}/solon"
fi

INSTALL_DIR="${HOME}/.solon/bin"
REPO="HuynhHoangPhuc/solon"

# Detect OS
case "$(uname -s)" in
  Linux*)             OS="linux";   EXT="" ;;
  Darwin*)            OS="darwin";  EXT="" ;;
  MINGW*|MSYS*|CYGWIN*) OS="windows"; EXT=".exe" ;;
  *) exit 0 ;;  # Unknown OS — skip silently
esac

# Detect arch
case "$(uname -m)" in
  x86_64|amd64) ARCH="x64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) exit 0 ;;  # Unknown arch — skip silently
esac

SL_BIN="${INSTALL_DIR}/sl${EXT}"
HOOKS_BIN="${INSTALL_DIR}/solon-hooks${EXT}"
SC_BIN="${INSTALL_DIR}/sc${EXT}"

# Fast no-op: all binaries exist
if [ -x "${SL_BIN}" ] && [ -x "${HOOKS_BIN}" ] && [ -x "${SC_BIN}" ]; then
  exit 0
fi

mkdir -p "${INSTALL_DIR}"

# Download helper: uses curl or wget
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

# Verify checksum helper (skips silently if no checksum tool available)
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

# Install a binary if missing
_install_binary() {
  local name="$1" dest="$2"
  [ -x "${dest}" ] && return 0

  local release_name="${name}-${OS}-${ARCH}${EXT}"
  local url="https://github.com/${REPO}/releases/latest/download/${release_name}"
  local tmp="${dest}.tmp"

  _download "${url}" "${tmp}" || { rm -f "${tmp}"; return 1; }
  _verify "${tmp}" "${url}" || return 1
  mv "${tmp}" "${dest}"
  [ "${OS}" != "windows" ] && chmod +x "${dest}"
}

failed=0
_install_binary "sl" "${SL_BIN}"             || failed=1
_install_binary "solon-hooks" "${HOOKS_BIN}" || failed=1
_install_binary "sc" "${SC_BIN}"             || failed=1

exit ${failed}
