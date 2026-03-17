#!/usr/bin/env bash
# ensure-binary.sh — Install or update the sl binary from GitHub Releases.
# Flow: check binary exists → get local version via --version → fetch latest
# tag from GitHub → download only if missing or outdated. Fast no-op when
# binary exists and version matches.

INSTALL_DIR="${HOME}/.solon/bin"
REPO="HuynhHoangPhuc/solon"
SL_BIN=""

# Detect OS + arch
case "$(uname -s)" in
  Linux*)                OS="linux";   EXT="" ;;
  Darwin*)               OS="darwin";  EXT="" ;;
  MINGW*|MSYS*|CYGWIN*) OS="windows"; EXT=".exe" ;;
  *) exit 0 ;;
esac
case "$(uname -m)" in
  x86_64|amd64)  ARCH="x64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) exit 0 ;;
esac

SL_BIN="${INSTALL_DIR}/sl${EXT}"
mkdir -p "${INSTALL_DIR}"

# --- Step 1: Check if binary exists and get local version ---
LOCAL_VERSION=""
if [ -x "${SL_BIN}" ]; then
  # Extract version from sl --version output (e.g., "sl 0.4.1" → "v0.4.1")
  RAW_VER=$("${SL_BIN}" --version 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')
  if [ -n "${RAW_VER}" ]; then
    LOCAL_VERSION="v${RAW_VER}"
  fi
fi

# --- Step 2: Get latest release tag from GitHub ---
LATEST_TAG=""
if command -v curl &>/dev/null; then
  LATEST_TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
    | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
elif command -v wget &>/dev/null; then
  LATEST_TAG=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
    | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
fi

# --- Step 3: Decide whether to download ---

# Binary exists and matches latest → nothing to do
if [ -x "${SL_BIN}" ] && [ -n "${LOCAL_VERSION}" ] && [ "${LOCAL_VERSION}" = "${LATEST_TAG}" ]; then
  exit 0
fi

# Binary exists but can't determine latest (network error) → keep current
if [ -x "${SL_BIN}" ] && [ -z "${LATEST_TAG}" ]; then
  exit 0
fi

# No binary and no latest tag → can't install
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
    return 1
  fi
}

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
    echo "[solon] Checksum mismatch" >&2
    rm -f "${bin}"
    return 1
  fi
}

RELEASE_NAME="sl-${OS}-${ARCH}${EXT}"
URL="https://github.com/${REPO}/releases/download/${TARGET}/${RELEASE_NAME}"
TMP="${SL_BIN}.tmp"

_download "${URL}" "${TMP}" || { rm -f "${TMP}"; exit 1; }
_verify "${TMP}" "${URL}" || exit 1
mv "${TMP}" "${SL_BIN}"
[ "${OS}" != "windows" ] && chmod +x "${SL_BIN}"
echo "[solon] Installed sl ${TARGET}" >&2
exit 0
