#!/usr/bin/env bash
# bctl installer — auto-detects OS/distro and installs the right package
set -euo pipefail

# Binaries are published to the dedicated releases repo. The source repo
# does not carry release-attached binary assets for every tag, and its
# /releases/latest endpoint can lag (e.g. when source-side releases are
# left in draft state). Querying the releases repo directly guarantees
# the install script gets the actual latest signed binary.
REPO="smichalabs/britivectl-releases"
BINARY="bctl"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# ── helpers ──────────────────────────────────────────────────────────────────

log()  { echo "==> $*"; }
die()  { echo "error: $*" >&2; exit 1; }

detect_os() {
    case "$(uname -s)" in
        Darwin) echo "darwin" ;;
        Linux)  echo "linux"  ;;
        *)      die "Unsupported OS: $(uname -s)" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64)          echo "x86_64" ;;
        arm64|aarch64)   echo "arm64"  ;;
        *)               die "Unsupported architecture: $(uname -m)" ;;
    esac
}

latest_version() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'
}

download() {
    curl -fsSL -o "$2" "$1"
}

verify_checksum() {
    local file="$1" checksums="$2" asset="$3"
    grep "${asset}" "${checksums}" | sha256sum --check --status \
        || die "Checksum verification failed for ${asset}"
}

# ── package manager install (Linux) ──────────────────────────────────────────

install_deb() {
    local url="$1" tmpdir="$2"
    local asset; asset="$(basename "${url}")"
    log "Downloading ${asset}..."
    download "${url}" "${tmpdir}/${asset}"
    log "Installing .deb package..."
    sudo dpkg -i "${tmpdir}/${asset}"
}

install_rpm() {
    local url="$1" tmpdir="$2"
    local asset; asset="$(basename "${url}")"
    log "Downloading ${asset}..."
    download "${url}" "${tmpdir}/${asset}"
    log "Installing .rpm package..."
    if command -v dnf &>/dev/null; then
        sudo dnf install -y "${tmpdir}/${asset}"
    else
        sudo rpm -i "${tmpdir}/${asset}"
    fi
}

install_tarball() {
    local url="$1" checksums_url="$2" tmpdir="$3" asset="$4"
    log "Downloading ${asset}..."
    download "${url}" "${tmpdir}/${asset}"
    download "${checksums_url}" "${tmpdir}/checksums.txt"
    log "Verifying checksum..."
    (cd "${tmpdir}" && verify_checksum "${asset}" "checksums.txt" "${asset}")
    log "Extracting..."
    tar -xzf "${tmpdir}/${asset}" -C "${tmpdir}"
    log "Installing to ${INSTALL_DIR}/${BINARY}..."
    install -m 755 "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
}

# ── main ─────────────────────────────────────────────────────────────────────

OS=$(detect_os)
ARCH=$(detect_arch)

log "Detecting latest bctl release..."
TAG=$(latest_version)
[ -n "${TAG}" ] || die "Could not determine latest release"
VERSION="${TAG#v}"

log "Installing bctl ${TAG}..."

TMPDIR=$(mktemp -d)
trap 'rm -rf "${TMPDIR}"' EXIT

BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
OS_TITLE="$(tr '[:lower:]' '[:upper:]' <<< "${OS:0:1}")${OS:1}"

if [[ "${OS}" == "darwin" ]]; then
    ASSET="bctl_${OS_TITLE}_${ARCH}.tar.gz"
    install_tarball "${BASE_URL}/${ASSET}" "${BASE_URL}/checksums.txt" "${TMPDIR}" "${ASSET}"

elif [[ "${OS}" == "linux" ]]; then
    # Prefer native package manager
    if command -v dpkg &>/dev/null; then
        # Debian / Ubuntu / WSL
        PKG_ARCH="${ARCH/x86_64/amd64}"
        ASSET="bctl_${VERSION}_linux_${PKG_ARCH}.deb"
        install_deb "${BASE_URL}/${ASSET}" "${TMPDIR}"
    elif command -v rpm &>/dev/null; then
        # RHEL / Fedora / CentOS
        PKG_ARCH="${ARCH/arm64/aarch64}"
        ASSET="bctl_${VERSION}_linux_${PKG_ARCH}.rpm"
        install_rpm "${BASE_URL}/${ASSET}" "${TMPDIR}"
    else
        # Fallback: tarball
        ASSET="bctl_${OS_TITLE}_${ARCH}.tar.gz"
        install_tarball "${BASE_URL}/${ASSET}" "${BASE_URL}/checksums.txt" "${TMPDIR}" "${ASSET}"
    fi
fi

echo ""
log "bctl ${TAG} installed successfully!"
echo "    Run 'bctl --help' to get started."
echo "    Run 'bctl init' to configure your tenant."
