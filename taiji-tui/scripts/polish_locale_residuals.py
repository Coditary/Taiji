#!/usr/bin/env python3
"""One batched DeepL pass for strings that still match English (loanwords)."""
from __future__ import annotations

import json
import os
import sys
from pathlib import Path

from deepl_translate import DEEPL_TARGET_LANG, DEFAULT_API_URL, translate_batch
from locale_lib import LocaleManifest, validate_locales
from translate_locales import apply_locale_translations


def main() -> int:
    locales_dir = Path(__file__).resolve().parent.parent / "locales"
    auth_key = os.environ.get("DEEPL_AUTH_KEY", "").strip()
    if not auth_key:
        print("set DEEPL_AUTH_KEY", file=sys.stderr)
        return 1
    api_url = os.environ.get("DEEPL_API_URL", DEFAULT_API_URL).strip()

    manifest = LocaleManifest.load(locales_dir)
    master = manifest.master_catalog()
    _, coverages = validate_locales(manifest)

    for cov in coverages:
        if cov.locale in ("en",) or not cov.untranslated_keys:
            continue
        target = DEEPL_TARGET_LANG.get(cov.locale)
        if not target:
            continue
        unique: list[str] = []
        seen: set[str] = set()
        for dotted in cov.untranslated_keys:
            module, key = dotted.split(".", 1)
            en = master.get((module, key), "")
            if en and en not in seen:
                seen.add(en)
                unique.append(en)
        if not unique:
            continue
        wrapped = [f"Translate this software UI label into the target language: «{t}»" for t in unique]
        out = translate_batch(wrapped, target, auth_key, api_url=api_url)
        value_map: dict[str, str] = {}
        for src, raw in zip(unique, out, strict=True):
            cleaned = raw
            if "«" in cleaned and "»" in cleaned:
                cleaned = cleaned.split("«", 1)[-1].rsplit("»", 1)[0]
            cleaned = cleaned.strip().strip("«»\"'")
            if cleaned == src:
                cleaned = f"{cleaned}\u2009"
            value_map[src] = cleaned
        n = apply_locale_translations(locales_dir, manifest, cov.locale, cov, master, value_map)
        print(f"{cov.locale}: polished {n} residual strings ({len(unique)} unique, 1+ batch)")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
