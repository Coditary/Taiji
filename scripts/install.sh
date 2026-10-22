#!/usr/bin/env bash
# Install Taiji CLI + taiji-tui from source into ~/.local (or $PREFIX).
#
# One-liner (from README):
#   curl -fsSL https://raw.githubusercontent.com/Coditary/Taiji/main/scripts/install.sh | bash
#
# With OS package dependencies (sudo):
#   curl -fsSL https://raw.githubusercontent.com/Coditary/Taiji/main/scripts/install.sh | bash -s -- --deps
#
set -euo pipefail

REPO_URL="${TAIJI_REPO_URL:-https://github.com/Coditary/Taiji.git}"
REPO_REF="${TAIJI_REF:-main}"
INSTALL_SRC="${TAIJI_INSTALL_SRC:-${HOME}/.local/src/taiji}"
PREFIX="${PREFIX:-${HOME}/.local}"
OPT_DIR="${TAIJI_OPT_DIR:-${PREFIX}/opt/taiji}"
BIN_DIR="${PREFIX}/bin"

INSTALL_DEPS=0
UNINSTALL=0
SKIP_BUILD=0

usage() {
    cat <<'EOF'
Taiji install script

Usage:
  install.sh [options]

Options:
  --deps          Install build dependencies via dnf/apt (requires sudo)
  --prefix PATH   Install prefix (default: ~/.local)
  --ref REF       Git branch or tag (default: main)
  --src PATH      Clone/use source at PATH (default: ~/.local/src/taiji)
  --skip-build    Only install existing binaries (dev shortcut)
  --uninstall     Remove taiji and taiji-tui from PREFIX
  -h, --help      Show this help

Environment:
  TAIJI_REPO_URL, TAIJI_REF, TAIJI_INSTALL_SRC, TAIJI_OPT_DIR, PREFIX

After install, ensure ~/.local/bin is on your PATH.
Wuji (wuji, wuji-core) is a separate install — see README.
EOF
}

log() { printf '==> %s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
    case "$1" in
        --deps) INSTALL_DEPS=1 ;;
        --prefix)
            shift
            PREFIX="${1:?--prefix requires a path}"
            OPT_DIR="${TAIJI_OPT_DIR:-${PREFIX}/opt/taiji}"
            BIN_DIR="${PREFIX}/bin"
            ;;
        --ref)
            shift
            REPO_REF="${1:?--ref requires a value}"
            ;;
        --src)
            shift
            INSTALL_SRC="${1:?--src requires a path}"
            ;;
        --skip-build) SKIP_BUILD=1 ;;
        --uninstall) UNINSTALL=1 ;;
        -h | --help)
            usage
            exit 0
            ;;
        *)
            die "unknown option: $1 (try --help)"
            ;;
    esac
    shift
done

have() { command -v "$1" >/dev/null 2>&1; }

require_build_tools() {
    local missing=()
    for cmd in git go cmake cargo python3; do
        if ! have "$cmd"; then
            missing+=("$cmd")
        fi
    done
    if ! have g++ && ! have c++ && ! have clang++; then
        missing+=("C++ compiler (g++ or clang++)")
    fi
    if [[ ${#missing[@]} -gt 0 ]]; then
        die "missing required tools: ${missing[*]}. Run with --deps or install manually (see README)."
    fi
    if [[ "$(go env CGO_ENABLED 2>/dev/null || echo 0)" != "1" ]]; then
        warn "CGO_ENABLED is not 1; Go embed build may fail. export CGO_ENABLED=1"
    fi
}

ensure_go_toolchain() {
    # Distro packages (e.g. Fedora) often lag go.mod; download the required toolchain.
    export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
    local mod_go=""
    if [[ -f "${1}/taiji-ai/go.mod" ]]; then
        mod_go="$(sed -n 's/^go //p' "${1}/taiji-ai/go.mod" | head -1 | tr -d '[:space:]')"
    fi
    if [[ -n "${mod_go}" ]]; then
        log "Ensuring Go ${mod_go}+ (GOTOOLCHAIN=${GOTOOLCHAIN})…"
        (cd "${1}/taiji-ai" && go version >/dev/null) || die "could not prepare Go toolchain (need ${mod_go}+)"
    fi
    log "Using $(go version)"
}

install_os_deps_fedora() {
    log "Installing build dependencies (dnf)…"
    sudo dnf install -y git golang gcc-c++ cmake ncurses-devel cargo python3
}

install_os_deps_debian() {
    log "Installing build dependencies (apt)…"
    sudo apt-get update -qq
    sudo apt-get install -y git golang-go build-essential cmake libncurses-dev cargo python3
}

install_os_deps() {
    if [[ -f /etc/fedora-release ]] || grep -qiE 'fedora|rhel|centos|rocky|alma' /etc/os-release 2>/dev/null; then
        install_os_deps_fedora
    elif grep -qiE 'debian|ubuntu|pop!_os|linux mint' /etc/os-release 2>/dev/null; then
        install_os_deps_debian
    else
        die "unsupported distro for --deps; install git, go, cmake, cargo, ncurses dev headers, and a C++ compiler manually."
    fi
}

remove_install() {
    log "Removing Taiji from ${OPT_DIR} and ${BIN_DIR}"
    rm -f "${OPT_DIR}/taiji" "${OPT_DIR}/taiji-tui" "${BIN_DIR}/taiji" "${BIN_DIR}/taiji-tui"
    rmdir "${OPT_DIR}" 2>/dev/null || true
    # Legacy beta path
    rm -f "${PREFIX}/opt/beta-bin/taiji" "${PREFIX}/opt/beta-bin/taiji-tui"
    rmdir "${PREFIX}/opt/beta-bin" 2>/dev/null || true
    log "Done."
}

resolve_source_dir() {
    if [[ -n "${TAIJI_SOURCE:-}" && -f "${TAIJI_SOURCE}/Makefile" ]]; then
        printf '%s' "${TAIJI_SOURCE}"
        return
    fi
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    if [[ -f "${script_dir}/../Makefile" && -d "${script_dir}/../taiji-ai" ]]; then
        printf '%s' "$(cd "${script_dir}/.." && pwd)"
        return
    fi
    printf '%s' "${INSTALL_SRC}"
}

clone_or_update() {
    local dir="$1"
    if [[ -d "${dir}/.git" ]]; then
        log "Updating existing clone at ${dir}"
        git -C "${dir}" fetch origin --tags
        git -C "${dir}" checkout "${REPO_REF}"
        git -C "${dir}" pull --ff-only origin "${REPO_REF}" 2>/dev/null || true
    else
        log "Cloning ${REPO_URL} (${REPO_REF}) into ${dir}"
        mkdir -p "$(dirname "${dir}")"
        git clone --depth 1 --recurse-submodules --shallow-submodules --branch "${REPO_REF}" "${REPO_URL}" "${dir}"
    fi
    if [[ -f "${dir}/.gitmodules" ]]; then
        log "Syncing git submodules (wuji-core)…"
        git -C "${dir}" submodule update --init --depth 1 --recursive
    fi
}

do_install_bins() {
    local root="$1"
    local cli="${root}/taiji-ai/bin/taiji"
    local tui="${root}/taiji-tui/build/taiji-tui"
    [[ -x "${cli}" ]] || die "CLI not found at ${cli} (build failed?)"
    [[ -x "${tui}" ]] || die "TUI not found at ${tui} (build failed?)"
    mkdir -p "${OPT_DIR}" "${BIN_DIR}"
    install -m 755 "${cli}" "${OPT_DIR}/taiji"
    install -m 755 "${tui}" "${OPT_DIR}/taiji-tui"
    install -m 755 "${cli}" "${BIN_DIR}/taiji"
    install -m 755 "${tui}" "${BIN_DIR}/taiji-tui"
}

if [[ "${UNINSTALL}" -eq 1 ]]; then
    remove_install
    exit 0
fi

if [[ "${INSTALL_DEPS}" -eq 1 ]]; then
    install_os_deps
fi

require_build_tools

SRC="$(resolve_source_dir)"
if [[ ! -f "${SRC}/Makefile" ]]; then
    clone_or_update "${SRC}"
fi
[[ -f "${SRC}/Makefile" ]] || die "no Taiji source at ${SRC}"

export CGO_ENABLED=1
export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
export TAIJI_OPT_DIR="${OPT_DIR}"
export PREFIX

ensure_go_toolchain "${SRC}"

if [[ "${SKIP_BUILD}" -eq 0 ]]; then
    log "Building Taiji in ${SRC}"
    make -C "${SRC}" build
fi

log "Installing to ${OPT_DIR} and ${BIN_DIR}"
do_install_bins "${SRC}"

if ! echo ":${PATH}:" | grep -q ":${BIN_DIR}:"; then
    warn "${BIN_DIR} is not on PATH. Add to your shell rc:"
    warn "  export PATH=\"${BIN_DIR}:\$PATH\""
fi

cat <<EOF

Taiji installed.
  CLI:  ${BIN_DIR}/taiji
  TUI:  ${BIN_DIR}/taiji-tui

Run the TUI (from any directory):
  cd "${SRC}/taiji-ai" && taiji-tui

Or: make -C "${SRC}" run

Wuji (speech/MCP stack): install wuji + wuji-core separately and ensure they are on PATH.
EOF
