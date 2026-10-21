#!/usr/bin/env python3
"""Set locale_names.<id> to each language's autonym (manifest native) for the settings picker."""
from __future__ import annotations

import json
import sys
from pathlib import Path

from locale_lib import LocaleManifest, load_json, set_nested


def main() -> int:
    locales_dir = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(__file__).resolve().parent.parent / "locales"
    manifest = LocaleManifest.load(locales_dir)
    manifest_data = json.loads((locales_dir / "manifest.json").read_text(encoding="utf-8"))
    autonyms: dict[str, str] = {}
    for entry in manifest_data.get("locales", []):
        lid = entry.get("id")
        native = entry.get("native")
        if isinstance(lid, str) and isinstance(native, str) and lid and native.strip():
            autonyms[lid] = native.strip()

    en_path = locales_dir / "en" / "common.json"
    en_data = load_json(en_path)
    for lid, name in autonyms.items():
        set_nested(en_data, f"locale_names.{lid}", name)
    en_path.write_text(json.dumps(en_data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    for locale_id in manifest.locale_ids:
        path = locales_dir / locale_id / "common.json"
        if not path.is_file():
            continue
        data = load_json(path)
        self_name = autonyms.get(locale_id)
        if self_name:
            set_nested(data, f"locale_names.{locale_id}", self_name)
        path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    print(f"synced autonyms for {len(autonyms)} locales")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
