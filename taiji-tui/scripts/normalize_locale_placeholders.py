#!/usr/bin/env python3
"""Replace locale override values that still copy English with LOCALE_PLACEHOLDER."""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from locale_lib import LOCALE_PLACEHOLDER, LocaleManifest, load_json, set_nested, validate_locales, write_key_catalog


def apply_placeholders(manifest: LocaleManifest, dry_run: bool) -> int:
    master = manifest.master_catalog()
    changed = 0
    pending: dict[tuple[str, str], dict] = {}

    for locale_id in manifest.locale_ids:
        if locale_id == manifest.default_locale:
            continue
        for (module, key), value in manifest.override_flat(locale_id).items():
            en_value = master.get((module, key), "")
            if not en_value.strip():
                continue
            if value != en_value or value == LOCALE_PLACEHOLDER:
                continue
            cache_key = (locale_id, module)
            if cache_key not in pending:
                pending[cache_key] = load_json(manifest.locales_dir / locale_id / f"{module}.json")
            set_nested(pending[cache_key], key, LOCALE_PLACEHOLDER)
            changed += 1

    if not dry_run:
        for (locale_id, module), data in pending.items():
            path = manifest.locales_dir / locale_id / f"{module}.json"
            path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    return changed


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("locales_dir", type=Path)
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()
    manifest = LocaleManifest.load(args.locales_dir)
    n = apply_placeholders(manifest, args.dry_run)
    if not args.dry_run and n:
        write_key_catalog(manifest, args.locales_dir / "key-catalog.json")
        _, _ = validate_locales(manifest)
    action = "would replace" if args.dry_run else "replaced"
    print(f"{action} {n} English-copy override values with {LOCALE_PLACEHOLDER!r}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
