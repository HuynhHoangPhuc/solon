#!/usr/bin/env bash
# install.sh — Download and install the sl binary from GitHub Releases
set -euo pipefail

REPO="solon-dev/solon"
INSTALL_DIR="${HOME}/.solon/bin"
BINARY_NAME="sl"

# Detect OS
case "$(uname -s)" in
  Linux*)  OS="linux" ;;
  Darwin*) OS="darwin" ;;
  MINGW*|MSYS*|CYGWIN*) OS="windows"; BINARY_NAME="sl.exe" ;;
  *)       echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

# Detect architecture
case "$(uname -m)" in
  x86_64|amd64) ARCH="x64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

RELEASE_NAME="sl-${OS}-${ARCH}"
if [ "${OS}" = "windows" ]; then
  RELEASE_NAME="${RELEASE_NAME}.exe"
fi

DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${RELEASE_NAME}"
CHECKSUM_URL="${DOWNLOAD_URL}.sha256"

echo "Installing solon CLI (sl) for ${OS}-${ARCH}..."
echo "Download URL: ${DOWNLOAD_URL}"

# Create install directory
mkdir -p "${INSTALL_DIR}"

DEST="${INSTALL_DIR}/${BINARY_NAME}"
TMP_DEST="${DEST}.tmp"

# Download binary
if command -v curl &>/dev/null; then
  curl -fsSL -o "${TMP_DEST}" "${DOWNLOAD_URL}"
elif command -v wget &>/dev/null; then
  wget -q -O "${TMP_DEST}" "${DOWNLOAD_URL}"
else
  echo "Error: curl or wget is required to download the binary." >&2
  exit 1
fi

# Verify SHA256 checksum if available
if command -v sha256sum &>/dev/null || command -v shasum &>/dev/null; then
  CHECKSUM_FILE="${TMP_DEST}.sha256"
  if command -v curl &>/dev/null; then
    curl -fsSL -o "${CHECKSUM_FILE}" "${CHECKSUM_URL}" 2>/dev/null || true
  else
    wget -q -O "${CHECKSUM_FILE}" "${CHECKSUM_URL}" 2>/dev/null || true
  fi

  if [ -f "${CHECKSUM_FILE}" ] && [ -s "${CHECKSUM_FILE}" ]; then
    echo "Verifying checksum..."
    EXPECTED=$(cat "${CHECKSUM_FILE}" | awk '{print $1}')
    if command -v sha256sum &>/dev/null; then
      ACTUAL=$(sha256sum "${TMP_DEST}" | awk '{print $1}')
    else
      ACTUAL=$(shasum -a 256 "${TMP_DEST}" | awk '{print $1}')
    fi
    if [ "${EXPECTED}" != "${ACTUAL}" ]; then
      echo "Checksum verification failed!" >&2
      rm -f "${TMP_DEST}" "${CHECKSUM_FILE}"
      exit 1
    fi
    echo "Checksum OK."
    rm -f "${CHECKSUM_FILE}"
  fi
fi

# Install binary
mv "${TMP_DEST}" "${DEST}"
chmod +x "${DEST}"

echo "Installed: ${DEST}"
echo "Version: $("${DEST}" --version 2>/dev/null || echo 'unknown')"

# Add to PATH if not already there
SHELL_RC=""
if [ -n "${ZSH_VERSION:-}" ] || [ "$(basename "${SHELL:-}")" = "zsh" ]; then
  SHELL_RC="${HOME}/.zshrc"
elif [ "$(basename "${SHELL:-}")" = "bash" ]; then
  SHELL_RC="${HOME}/.bashrc"
fi

PATH_LINE="export PATH=\"\$PATH:${INSTALL_DIR}\""

if [ -n "${SHELL_RC}" ] && [ -f "${SHELL_RC}" ]; then
  if ! grep -qF "${INSTALL_DIR}" "${SHELL_RC}"; then
    echo "" >> "${SHELL_RC}"
    echo "# Added by solon installer" >> "${SHELL_RC}"
    echo "${PATH_LINE}" >> "${SHELL_RC}"
    echo "Added ${INSTALL_DIR} to PATH in ${SHELL_RC}"
    echo "Run: source ${SHELL_RC}"
  fi
fi

echo ""
echo "Installation complete! Run 'sl --version' to verify."
echo "If 'sl' is not found, add this to your shell config:"
echo "  ${PATH_LINE}"
