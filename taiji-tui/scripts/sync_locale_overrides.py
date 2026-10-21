#!/usr/bin/env python3
"""Copy missing master keys from English into each locale override file."""
from __future__ import annotations

import argparse
import json
from pathlib import Path

from locale_lib import LOCALE_PLACEHOLDER, LocaleManifest, load_json, set_nested, validate_locales


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("locales_dir", nargs="?", default=None)
    parser.add_argument("--locale", action="append", metavar="ID")
    args = parser.parse_args()

    script_dir = Path(__file__).resolve().parent
    locales_dir = Path(args.locales_dir) if args.locales_dir else script_dir.parent / "locales"
    manifest = LocaleManifest.load(locales_dir)
    master = manifest.master_catalog()
    _, coverages = validate_locales(manifest)

    targets = args.locale or [c.locale for c in coverages if c.locale != manifest.default_locale]
    for locale_id in targets:
        cov = next(c for c in coverages if c.locale == locale_id)
        if not cov.missing_override_keys:
            continue
        by_module: dict[str, dict] = {}
        for module in manifest.modules:
            path = locales_dir / locale_id / f"{module}.json"
            by_module[module] = load_json(path) if path.is_file() else {}

        for dotted in cov.missing_override_keys:
            module, key = dotted.split(".", 1)
            en_value = master[(module, key)]
            placeholder = en_value if not en_value.strip() else LOCALE_PLACEHOLDER
            set_nested(by_module[module], key, placeholder)

        for module, data in by_module.items():
            out = locales_dir / locale_id / f"{module}.json"
            out.parent.mkdir(parents=True, exist_ok=True)
            out.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
        print(f"{locale_id}: added {len(cov.missing_override_keys)} keys")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
