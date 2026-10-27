#!/usr/bin/env bash
# Build Taiji inside Docker from a plain source tree — no Wuji CLI installed.
#
# Usage (from repo root):
#   ./scripts/test-docker-build-without-wuji.sh
#
# Requires: docker, initialized wuji-core submodule (compile-time Go dependency).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${TAIJI_DOCKER_BUILD_IMAGE:-taiji-build-without-wuji}"
DOCKERFILE="${ROOT}/docker/build-without-wuji/Dockerfile"

log() { printf '==> %s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

if ! command -v docker >/dev/null 2>&1; then
    die "docker is not installed or not on PATH"
fi

if [[ ! -f "${ROOT}/wuji-core/go.mod" ]]; then
    die "wuji-core submodule missing at ${ROOT}/wuji-core — run: git submodule update --init --depth 1 wuji-core"
fi

STAGE="$(mktemp -d)"
trap 'rm -rf "${STAGE}"' EXIT

log "Staging source tree (no .git, no local Wuji clone)…"
rsync -a \
    --exclude '.git/' \
    --exclude 'wuji-ai/' \
    --exclude '.wuji/' \
    --exclude '.taiji/' \
    --exclude 'taiji-tui/build/' \
    --exclude 'taiji-tui/build-test/' \
    --exclude 'taiji-ai/bin/' \
    --exclude 'taiji-ai/globwalk/target/' \
    --exclude 'taiji-ai/taiji-tools/target/' \
    --exclude 'taiji-tui/fuzzy/target/' \
    --exclude 'taiji-tui/highlight/target/' \
    "${ROOT}/" "${STAGE}/taiji/"

[[ -f "${STAGE}/taiji/Makefile" ]] || die "staged tree is incomplete (Makefile missing)"

log "Building Docker image ${IMAGE}…"
docker build -t "${IMAGE}" -f "${DOCKERFILE}" "${ROOT}/docker/build-without-wuji"

TARBALL="${STAGE}/taiji-src.tar.gz"
tar -C "${STAGE}" -czf "${TARBALL}" taiji

log "Running install/build inside container (PATH has no wuji)…"
# Tarball into the container avoids bind-mount SELinux issues on Fedora hosts.
docker run --rm -i \
    -e TAIJI_SOURCE=/work/taiji \
    -e PREFIX=/opt/taiji-prefix \
    -e TAIJI_OPT_DIR=/opt/taiji-prefix/opt/taiji \
    "${IMAGE}" \
    bash -ec '
        set -euo pipefail
        mkdir -p /work
        tar -xzf - -C /work
        if command -v wuji >/dev/null 2>&1; then
            echo "error: wuji must not be on PATH in this test image" >&2
            exit 1
        fi
        if command -v wuji-core >/dev/null 2>&1; then
            echo "error: wuji-core CLI must not be on PATH in this test image" >&2
            exit 1
        fi
        export CGO_ENABLED=1 GOTOOLCHAIN=auto
        bash /work/taiji/scripts/install.sh
        test -x /opt/taiji-prefix/bin/taiji
        test -x /opt/taiji-prefix/bin/taiji-tui
        /opt/taiji-prefix/bin/taiji --help >/dev/null
        echo "ok: taiji and taiji-tui built without Wuji CLI on PATH"
    ' < "${TARBALL}"

log "Docker build-without-wuji test passed."
