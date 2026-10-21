#!/usr/bin/env python3
"""Validate taiji-tui locale JSON against the English master catalog."""
from __future__ import annotations

import argparse
import sys
from pathlib import Path

from locale_lib import LocaleManifest, validate_locales, write_i18n_status, write_key_catalog


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "locales_dir",
        nargs="?",
        default=None,
        help="path to locales/ (default: ../locales next to this script)",
    )
    parser.add_argument(
        "--write-catalog",
        metavar="PATH",
        help="write key-catalog.json (all master keys + English values)",
    )
    parser.add_argument(
        "--write-status",
        metavar="PATH",
        help="write i18n-status.json (per-locale coverage report)",
    )
    parser.add_argument(
        "--status-full",
        action="store_true",
        help="include full missing/untranslated key lists in i18n-status.json",
    )
    parser.add_argument(
        "--min-translated-pct",
        type=float,
        default=None,
        metavar="PCT",
        help="fail if any non-default locale is below this %% key coverage in override JSON",
    )
    parser.add_argument(
        "--min-localized-pct",
        type=float,
        default=None,
        metavar="PCT",
        help="fail if merged values still match English below this %%",
    )
    parser.add_argument(
        "--require-complete",
        action="store_true",
        help="build gate: fail on unknown keys or empty required strings (not on ... or English copy)",
    )
    parser.add_argument(
        "--locale",
        action="append",
        metavar="ID",
        help="only report missing/untranslated keys for these locale ids",
    )
    parser.add_argument("--quiet", action="store_true", help="only print errors")
    args = parser.parse_args()

    script_dir = Path(__file__).resolve().parent
    locales_dir = Path(args.locales_dir) if args.locales_dir else script_dir.parent / "locales"
    manifest = LocaleManifest.load(locales_dir)
    issues, coverages = validate_locales(manifest)

    catalog_path = Path(args.write_catalog) if args.write_catalog else locales_dir / "key-catalog.json"
    write_key_catalog(manifest, catalog_path)

    status_path = Path(args.write_status) if args.write_status else locales_dir / "i18n-status.json"
    write_i18n_status(manifest, coverages, status_path, include_key_lists=args.status_full)

    filter_locales = set(args.locale or [])
    exit_code = 0

    blocking_kinds = {"unknown_key", "empty_value"}
    blocking_issues = [issue for issue in issues if issue.kind in blocking_kinds]
    if args.require_complete:
        if blocking_issues:
            exit_code = 1
        if not args.quiet and blocking_issues:
            print("Locale validation errors:", file=sys.stderr)
            for issue in blocking_issues:
                if filter_locales and issue.locale not in filter_locales:
                    continue
                print(f"  [{issue.kind}] {issue.locale}: {issue.message}", file=sys.stderr)
    elif issues:
        exit_code = 1
        if not args.quiet:
            print("Locale validation errors:", file=sys.stderr)
            for issue in issues:
                if filter_locales and issue.locale not in filter_locales:
                    continue
                print(f"  [{issue.kind}] {issue.locale}: {issue.message}", file=sys.stderr)

    if args.min_translated_pct is not None:
        for cov in coverages:
            if cov.locale == manifest.default_locale:
                continue
            if filter_locales and cov.locale not in filter_locales:
                continue
            if cov.translated_pct < args.min_translated_pct:
                exit_code = 1
                print(
                    f"  [coverage] {cov.locale}: {cov.translated_pct:.1f}% keys in locale files "
                    f"(minimum {args.min_translated_pct}%)",
                    file=sys.stderr,
                )

    if args.min_localized_pct is not None:
        for cov in coverages:
            if cov.locale == manifest.default_locale:
                continue
            if filter_locales and cov.locale not in filter_locales:
                continue
            localized_pct = (
                100.0 * cov.localized_count / cov.total_keys if cov.total_keys else 100.0
            )
            if localized_pct < args.min_localized_pct:
                exit_code = 1
                print(
                    f"  [localized] {cov.locale}: {localized_pct:.1f}% differ from English "
                    f"(minimum {args.min_localized_pct}%)",
                    file=sys.stderr,
                )

    if not args.quiet:
        print(f"Catalog: {catalog_path} ({manifest.merged_flat(manifest.default_locale).__len__()} keys)")
        print(f"Status:  {status_path}")
        print()
        print(f"{'locale':<6} {'keys':>11} {'localized':>11} {'missing files':>14}")
        for cov in coverages:
            if filter_locales and cov.locale not in filter_locales:
                continue
            missing_files = ",".join(cov.missing_module_files) if cov.missing_module_files else "-"
            loc_pct = 100.0 * cov.localized_count / cov.total_keys if cov.total_keys else 100.0
            print(
                f"{cov.locale:<6} "
                f"{cov.translated_count:>4}/{cov.total_keys} ({cov.translated_pct:5.1f}%) "
                f"{cov.localized_count:>4}/{cov.total_keys} ({loc_pct:5.1f}%) "
                f"{missing_files:>14}"
            )

        if filter_locales or args.locale:
            for cov in coverages:
                if filter_locales and cov.locale not in filter_locales:
                    continue
                if cov.locale == manifest.default_locale:
                    continue
                if cov.missing_override_keys:
                    print(f"\n{cov.locale}: missing override keys ({len(cov.missing_override_keys)}):")
                    for key in cov.missing_override_keys[:40]:
                        print(f"  - {key}")
                    if len(cov.missing_override_keys) > 40:
                        print(f"  ... and {len(cov.missing_override_keys) - 40} more")

    return exit_code


if __name__ == "__main__":
    raise SystemExit(main())
