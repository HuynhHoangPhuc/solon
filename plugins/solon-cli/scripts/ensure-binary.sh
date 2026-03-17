#!/usr/bin/env bash
# ensure-binary.sh — Install or update the sl binary from GitHub Releases.
# Exit 0 = sl binary ready + PATH injected. Exit 1 = failed.
# Safe for concurrent execution (multiple plugins may call this simultaneously).

INSTALL_DIR="${HOME}/.solon/bin"
REPO="HuynhHoangPhuc/solon"

# --- Helpers ---

_inject_path() {
  local env_file="${CLAUDE_ENV_FILE:-}"
  if [ -n "${env_file}" ]; then
    echo "export PATH=\"${INSTALL_DIR}:\${PATH}\"" >> "${env_file}"
  fi
}

_success() {
  _inject_path
  exit 0
}

_bin_exists() {
  [ -f "${SL_BIN}" ] && [ -s "${SL_BIN}" ]
}

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
  # Verify download produced a file (not a directory or empty)
  [ -f "${dest}" ] && [ -s "${dest}" ]
}

_verify_checksum() {
  local bin="$1" url="$2"
  local chk="${bin}.sha256"
  _download "${url}.sha256" "${chk}" 2>/dev/null || return 0
  [ -f "${chk}" ] && [ -s "${chk}" ] || { rm -f "${chk}" 2>/dev/null; return 0; }
  local expected actual
  expected=$(awk '{print $1}' "${chk}")
  if command -v sha256sum &>/dev/null; then
    actual=$(sha256sum "${bin}" | awk '{print $1}')
  elif command -v shasum &>/dev/null; then
    actual=$(shasum -a 256 "${bin}" | awk '{print $1}')
  else
    rm -f "${chk}" 2>/dev/null; return 0
  fi
  rm -f "${chk}" 2>/dev/null
  if [ "${expected}" != "${actual}" ]; then
    echo "[solon] Checksum mismatch" >&2
    rm -f "${bin}" 2>/dev/null
    return 1
  fi
}

# --- Detect OS + arch ---

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
mkdir -p "${INSTALL_DIR}" 2>/dev/null || true

# --- Step 1: Check if binary exists and get local version ---

LOCAL_VERSION=""
if _bin_exists; then
  RAW_VER=$("${SL_BIN}" --version 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' || true)
  [ -n "${RAW_VER}" ] && LOCAL_VERSION="v${RAW_VER}"
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

# Binary up-to-date → done
if _bin_exists && [ -n "${LOCAL_VERSION}" ] && [ "${LOCAL_VERSION}" = "${LATEST_TAG}" ]; then
  _success
fi

# Binary exists, can't reach GitHub → keep current
if _bin_exists && [ -z "${LATEST_TAG}" ]; then
  _success
fi

# No binary, no GitHub → fatal
if ! _bin_exists && [ -z "${LATEST_TAG}" ]; then
  echo "[solon] sl not found and cannot reach GitHub" >&2
  exit 1
fi

# --- Step 4: Download ---

TARGET="${LATEST_TAG}"
if [ -n "${LOCAL_VERSION}" ]; then
  echo "[solon] Updating sl: ${LOCAL_VERSION} -> ${TARGET}" >&2
else
  echo "[solon] Installing sl ${TARGET}..." >&2
fi

RELEASE_NAME="sl-${OS}-${ARCH}${EXT}"
URL="https://github.com/${REPO}/releases/download/${TARGET}/${RELEASE_NAME}"
# Use PID in temp name to avoid race conditions with concurrent hooks
TMP="${SL_BIN}.tmp.$$"

# Clean up any stale temp files (directories or files)
rm -rf "${SL_BIN}".tmp* 2>/dev/null || true

if ! _download "${URL}" "${TMP}"; then
  rm -rf "${TMP}" 2>/dev/null || true
  if _bin_exists; then
    echo "[solon] Download failed, keeping existing sl ${LOCAL_VERSION}" >&2
    _success
  fi
  echo "[solon] Failed to download sl from ${URL}" >&2
  exit 1
fi

# Verify checksum
if ! _verify_checksum "${TMP}" "${URL}"; then
  rm -rf "${TMP}" 2>/dev/null || true
  if _bin_exists; then
    echo "[solon] Checksum failed, keeping existing sl ${LOCAL_VERSION}" >&2
    _success
  fi
  echo "[solon] Checksum verification failed" >&2
  exit 1
fi

# Atomic move (overwrite existing)
mv -f "${TMP}" "${SL_BIN}" 2>/dev/null || {
  # mv failed — maybe another hook already updated, or permission issue
  rm -rf "${TMP}" 2>/dev/null || true
  if _bin_exists; then
    _success
  fi
  echo "[solon] Failed to install sl binary" >&2
  exit 1
}

# Set permissions
if [ "${OS}" = "windows" ]; then
  # Remove Windows SmartScreen block
  if command -v powershell.exe &>/dev/null; then
    powershell.exe -NoProfile -Command "Unblock-File '$(cygpath -w "${SL_BIN}" 2>/dev/null || echo "${SL_BIN}")'" 2>/dev/null || true
  fi
else
  chmod +x "${SL_BIN}" 2>/dev/null || true
fi

# --- Step 5: Verify binary works ---

if ! "${SL_BIN}" --version >/dev/null 2>&1; then
  echo "[solon] Installed sl binary failed verification" >&2
  rm -f "${SL_BIN}" 2>/dev/null || true
  exit 1
fi

echo "[solon] sl ${TARGET} ready" >&2
_success
