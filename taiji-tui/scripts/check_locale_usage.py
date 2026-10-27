#!/usr/bin/env python3
"""Ensure every ui_tr/ui_shell key in taiji-tui/src exists in the English locale master."""
from __future__ import annotations

import argparse
import sys
from pathlib import Path

from locale_lib import LocaleManifest, flatten, load_json
from locale_usage import dedupe_refs, scan_sources


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "locales_dir",
        nargs="?",
        default=None,
        help="path to locales/ (default: ../locales next to this script)",
    )
    parser.add_argument(
        "--src",
        default=None,
        help="taiji-tui/src directory (default: ../src)",
    )
    parser.add_argument(
        "--warn-fallbacks",
        action="store_true",
        help="print English literals used as ui_shell/ui_tr empty() fallbacks",
    )
    parser.add_argument("--quiet", action="store_true")
    args = parser.parse_args()

    script_dir = Path(__file__).resolve().parent
    locales_dir = Path(args.locales_dir) if args.locales_dir else script_dir.parent / "locales"
    src_dir = Path(args.src) if args.src else script_dir.parent / "src"

    manifest = LocaleManifest.load(locales_dir)
    master = manifest.master_catalog()

    refs, fallbacks = scan_sources(src_dir)
    unique_refs = dedupe_refs(refs)

    missing: list[str] = []
    for ref in unique_refs:
        if (ref.module, ref.key) not in master:
            missing.append(f"{ref.dotted} (first: {ref.file}:{ref.line})")

    exit_code = 0
    if missing:
        exit_code = 1
        if not args.quiet:
            print(f"Missing {len(missing)} locale key(s) in {manifest.default_locale} master:", file=sys.stderr)
            for item in missing:
                print(f"  - {item}", file=sys.stderr)

    if args.warn_fallbacks and fallbacks and not args.quiet:
        print(f"\nHardcoded empty() fallbacks ({len(fallbacks)} occurrence(s)):", file=sys.stderr)
        for file, line, literal in fallbacks[:80]:
            print(f"  - {file}:{line}: \"{literal}\"", file=sys.stderr)
        if len(fallbacks) > 80:
            print(f"  ... and {len(fallbacks) - 80} more", file=sys.stderr)

    if exit_code == 0 and not args.quiet:
        print(f"OK: {len(unique_refs)} ui_tr/ui_shell keys referenced in src match the master catalog.")
    return exit_code


if __name__ == "__main__":
    raise SystemExit(main())
