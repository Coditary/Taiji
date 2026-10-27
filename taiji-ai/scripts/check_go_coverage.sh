#!/usr/bin/env bash
# Fail if Go statement coverage for COVER_PKGS is below COVER_MIN (default 80).
#
# Usage (from taiji-ai/):
#   ./scripts/check_go_coverage.sh
#   COVER_MIN=75 COVER_PKGS='./internal/agent/...' ./scripts/check_go_coverage.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

COVER_MIN="${COVER_MIN:-80}"
COVER_PKGS="${COVER_PKGS:-./internal/...}"
PROFILE="${COVER_PROFILE:-${ROOT}/.cache/go-cover.out}"

mkdir -p "$(dirname "${PROFILE}")"

if ! go test "${COVER_PKGS}" -count=1 -coverprofile="${PROFILE}" >/tmp/taiji-cover-test.log 2>&1; then
    cat /tmp/taiji-cover-test.log
    exit 1
fi

total_line="$(go tool cover -func="${PROFILE}" | awk '/^total:/ {print $3}')"
pct="${total_line%\%}"
if ! awk -v p="${pct}" -v min="${COVER_MIN}" 'BEGIN { exit (p+0 >= min+0) ? 0 : 1 }'; then
    printf 'coverage %s is below minimum %s%% (packages: %s)\n' "${pct}%" "${COVER_MIN}" "${COVER_PKGS}" >&2
    exit 1
fi
printf 'coverage %s (minimum %s%%) on %s\n' "${pct}%" "${COVER_MIN}" "${COVER_PKGS}"
