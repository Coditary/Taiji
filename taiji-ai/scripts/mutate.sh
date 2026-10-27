#!/usr/bin/env bash
# Run Go mutation testing with gremlins (https://gremlins.dev).
#
# Usage (from taiji-ai/):
#   ./scripts/mutate.sh                  # default package ./internal/provider
#   ./scripts/mutate.sh ./internal/agent
#   ./scripts/mutate.sh --dry-run ./internal/config
#   MUTATE_EXCLUDE='wuji\\.go$' ./scripts/mutate.sh ./internal/provider
#
# Full module (slow):
#   ./scripts/mutate.sh --module
#
# Changed code only (git diff vs HEAD):
#   ./scripts/mutate.sh --diff
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

GREMLINS_VERSION="${GREMLINS_VERSION:-v0.6.0}"
GREMLINS_BIN="${GREMLINS_BIN:-}"

log() { printf '==> %s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

ensure_gremlins() {
    if [[ -n "${GREMLINS_BIN}" && -x "${GREMLINS_BIN}" ]]; then
        return 0
    fi
    if command -v gremlins >/dev/null 2>&1; then
        GREMLINS_BIN="$(command -v gremlins)"
        return 0
    fi
    log "Installing gremlins ${GREMLINS_VERSION} (go install)…"
    GOTOOLCHAIN="${GOTOOLCHAIN:-auto}" go install "github.com/go-gremlins/gremlins/cmd/gremlins@${GREMLINS_VERSION}"
    GREMLINS_BIN="$(go env GOPATH)/bin/gremlins"
    [[ -x "${GREMLINS_BIN}" ]] || die "gremlins not found after install (GOPATH/bin)"
}

run_unleash() {
    local -a args=(unleash --config "${ROOT}/.gremlins.yaml")
    if [[ -n "${MUTATE_OUTPUT:-}" ]]; then
        args+=(-o "${MUTATE_OUTPUT}")
    fi
    if [[ -n "${MUTATE_EXCLUDE:-}" ]]; then
        local -a extra_excludes=()
        IFS=';' read -r -a extra_excludes <<< "${MUTATE_EXCLUDE}"
        local part
        for part in "${extra_excludes[@]}"; do
            [[ -z "${part}" ]] && continue
            args+=(-E "${part}")
        done
    fi
    args+=("$@")
    log "gremlins ${args[*]}"
    "${GREMLINS_BIN}" "${args[@]}"
}

ensure_gremlins
export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"

if [[ $# -eq 0 ]]; then
    set -- "./internal/provider"
fi

case "${1:-}" in
    --module)
        shift
        run_unleash . "$@"
        ;;
    --diff)
        shift
        diff_ref="${MUTATE_DIFF:-HEAD}"
        run_unleash --diff "${diff_ref}" . "$@"
        ;;
    --dry-run)
        shift
        if [[ $# -eq 0 ]]; then
            set -- "./internal/provider"
        fi
        run_unleash --dry-run "$@"
        ;;
    -h | --help)
        sed -n '2,18p' "$0"
        exit 0
        ;;
    *)
        run_unleash "$@"
        ;;
esac
