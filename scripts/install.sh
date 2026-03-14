#!/usr/bin/env bash
# install.sh — Download and install sl + solon-hooks binaries from GitHub Releases
set -euo pipefail

REPO="HuynhHoangPhuc/solon"
INSTALL_DIR="${HOME}/.solon/bin"

# Detect OS
case "$(uname -s)" in
  Linux*)             OS="linux";   EXT="" ;;
  Darwin*)            OS="darwin";  EXT="" ;;
  MINGW*|MSYS*|CYGWIN*) OS="windows"; EXT=".exe" ;;
  *)       echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

# Detect architecture
case "$(uname -m)" in
  x86_64|amd64) ARCH="x64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

mkdir -p "${INSTALL_DIR}"

# Download helper
_download() {
  local url="$1" dest="$2"
  if command -v curl &>/dev/null; then
    curl -fsSL -o "${dest}" "${url}"
  elif command -v wget &>/dev/null; then
    wget -q -O "${dest}" "${url}"
  else
    echo "Error: curl or wget is required." >&2
    exit 1
  fi
}

# Verify + install one binary
_install_binary() {
  local name="$1"
  local release_name="${name}-${OS}-${ARCH}${EXT}"
  local url="https://github.com/${REPO}/releases/latest/download/${release_name}"
  local dest="${INSTALL_DIR}/${name}${EXT}"
  local tmp="${dest}.tmp"

  echo "Installing ${name} for ${OS}-${ARCH}..."
  _download "${url}" "${tmp}"

  # Verify SHA256 checksum if available
  if command -v sha256sum &>/dev/null || command -v shasum &>/dev/null; then
    local chk="${tmp}.sha256"
    _download "${url}.sha256" "${chk}" 2>/dev/null || true
    if [ -f "${chk}" ] && [ -s "${chk}" ]; then
      echo "Verifying checksum..."
      local expected actual
      expected=$(awk '{print $1}' "${chk}")
      if command -v sha256sum &>/dev/null; then
        actual=$(sha256sum "${tmp}" | awk '{print $1}')
      else
        actual=$(shasum -a 256 "${tmp}" | awk '{print $1}')
      fi
      rm -f "${chk}"
      if [ "${expected}" != "${actual}" ]; then
        echo "Checksum verification failed for ${name}!" >&2
        rm -f "${tmp}"
        exit 1
      fi
      echo "Checksum OK."
    fi
  fi

  mv "${tmp}" "${dest}"
  [ "${OS}" != "windows" ] && chmod +x "${dest}"
  echo "Installed: ${dest}"
}

_install_binary "sl"
_install_binary "solon-hooks"

echo ""
echo "Version: $(\"${INSTALL_DIR}/sl${EXT}\" --version 2>/dev/null || echo 'unknown')"

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
