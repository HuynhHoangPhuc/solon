#!/usr/bin/env bash
# ensure-binary.sh — Install or update the sl binary from GitHub Releases.
# Exit 0 = sl binary ready + PATH injected. Exit 1 = failed.
# Also injects INSTALL_DIR into PATH via CLAUDE_ENV_FILE so skills can call `sl` directly.
set -euo pipefail

INSTALL_DIR="${HOME}/.solon/bin"
REPO="HuynhHoangPhuc/solon"

# Inject INSTALL_DIR into session PATH via Claude Code env file
_inject_path() {
  local env_file="${CLAUDE_ENV_FILE:-}"
  if [ -n "${env_file}" ]; then
    # Escape $PATH so it expands at execution time, not write time
    echo "export PATH=\"${INSTALL_DIR}:\${PATH}\"" >> "${env_file}"
  fi
}

# Called on every successful exit — ensures PATH is set each session
_success() {
  _inject_path
  exit 0
}

# Detect OS + arch
case "$(uname -s)" in
  Linux*)                OS="linux";   EXT="" ;;
  Darwin*)               OS="darwin";  EXT="" ;;
  MINGW*|MSYS*|CYGWIN*) OS="windows"; EXT=".exe" ;;
  *) echo "[solon] Unsupported OS" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64|amd64)  ARCH="x64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "[solon] Unsupported arch" >&2; exit 1 ;;
esac

SL_BIN="${INSTALL_DIR}/sl${EXT}"
mkdir -p "${INSTALL_DIR}"

# --- Step 1: Check if binary exists and get local version ---
LOCAL_VERSION=""
if [ -x "${SL_BIN}" ]; then
  RAW_VER=$("${SL_BIN}" --version 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' || true)
  if [ -n "${RAW_VER}" ]; then
    LOCAL_VERSION="v${RAW_VER}"
  fi
fi

# --- Step 2: Get latest release tag from GitHub ---
LATEST_TAG=""
if command -v curl &>/dev/null; then
  LATEST_TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
    | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/' || true)
elif command -v wget &>/dev/null; then
  LATEST_TAG=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
    | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/' || true)
fi

# --- Step 3: Decide whether to download ---

# Binary exists and matches latest → ready
if [ -x "${SL_BIN}" ] && [ -n "${LOCAL_VERSION}" ] && [ "${LOCAL_VERSION}" = "${LATEST_TAG}" ]; then
  _success
fi

# Binary exists but can't reach GitHub → keep current, verify it works
if [ -x "${SL_BIN}" ] && [ -z "${LATEST_TAG}" ]; then
  "${SL_BIN}" --version >/dev/null 2>&1 || { echo "[solon] sl binary exists but is corrupted" >&2; exit 1; }
  _success
fi

# No binary and no GitHub access → fatal
if [ ! -x "${SL_BIN}" ] && [ -z "${LATEST_TAG}" ]; then
  echo "[solon] sl binary not found and cannot reach GitHub to download" >&2
  exit 1
fi

# --- Step 4: Download ---
TARGET="${LATEST_TAG}"
if [ -n "${LOCAL_VERSION}" ]; then
  echo "[solon] Updating sl: ${LOCAL_VERSION} -> ${TARGET}" >&2
else
  echo "[solon] Installing sl ${TARGET}..." >&2
fi

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

_verify_checksum() {
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
    echo "[solon] Checksum mismatch for sl binary" >&2
    rm -f "${bin}"
    return 1
  fi
}

RELEASE_NAME="sl-${OS}-${ARCH}${EXT}"
URL="https://github.com/${REPO}/releases/download/${TARGET}/${RELEASE_NAME}"
TMP="${SL_BIN}.tmp"

if ! _download "${URL}" "${TMP}"; then
  rm -f "${TMP}"
  # Download failed — fall back to existing binary if available
  if [ -x "${SL_BIN}" ]; then
    echo "[solon] Download failed, keeping existing sl ${LOCAL_VERSION}" >&2
    _success
  fi
  echo "[solon] Failed to download sl from ${URL}" >&2
  exit 1
fi
_verify_checksum "${TMP}" "${URL}" || { echo "[solon] Checksum verification failed" >&2; exit 1; }
mv "${TMP}" "${SL_BIN}"
if [ "${OS}" = "windows" ]; then
  powershell.exe -NoProfile -Command "Unblock-File '$(cygpath -w "${SL_BIN}")'" 2>/dev/null || true
else
  chmod +x "${SL_BIN}"
fi

# --- Step 5: Verify binary works ---
"${SL_BIN}" --version >/dev/null 2>&1 || { echo "[solon] Installed sl binary failed verification" >&2; rm -f "${SL_BIN}"; exit 1; }

echo "[solon] sl ${TARGET} ready" >&2
_success
