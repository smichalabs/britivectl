#!/usr/bin/env bash
# bctl installer — downloads the latest release from GitHub and installs to /usr/local/bin
set -euo pipefail

REPO="smichalabs/britivectl"
BINARY="bctl"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

detect_os() {
    case "$(uname -s)" in
        Darwin) echo "Darwin" ;;
        Linux)  echo "Linux"  ;;
        *)      echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64)  echo "x86_64" ;;
        arm64)   echo "arm64"  ;;
        aarch64) echo "arm64"  ;;
        *)       echo "Unsupported arch: $(uname -m)" >&2; exit 1 ;;
    esac
}

OS=$(detect_os)
ARCH=$(detect_arch)

echo "Detecting latest bctl release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')

if [[ -z "$LATEST" ]]; then
    echo "Error: could not determine latest release" >&2
    exit 1
fi

VERSION="${LATEST#v}"
ASSET="${BINARY}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${ASSET}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/${LATEST}/checksums.txt"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading bctl ${LATEST} (${OS}/${ARCH})..."
curl -fsSL -o "${TMPDIR}/${ASSET}" "${URL}"
curl -fsSL -o "${TMPDIR}/checksums.txt" "${CHECKSUM_URL}"

echo "Verifying checksum..."
cd "${TMPDIR}"
grep "${ASSET}" checksums.txt | sha256sum --check --status || {
    echo "Checksum verification failed" >&2
    exit 1
}

echo "Extracting..."
tar -xzf "${ASSET}"

echo "Installing to ${INSTALL_DIR}/${BINARY}..."
install -m 755 "${BINARY}" "${INSTALL_DIR}/${BINARY}"

echo ""
echo "bctl ${LATEST} installed successfully!"
echo "Run 'bctl --help' to get started."
