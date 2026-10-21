#!/usr/bin/env python3
"""Ensure locale_names use TUI-safe exonyms (no CJK/Cyrillic/autonym scripts in the picker)."""
from __future__ import annotations

import json
import sys
from pathlib import Path

from locale_lib import LocaleManifest, load_json, set_nested

# Latin-only labels for languages whose autonyms break the TUI font.
SCRIPT_EXONYMS: dict[str, dict[str, str]] = {
    "en": {"ja": "Japanese", "zh": "Chinese", "ru": "Russian"},
    "de": {"ja": "Japanisch", "zh": "Chinesisch", "ru": "Russisch"},
    "es": {"ja": "Japonés", "zh": "Chino", "ru": "Ruso"},
    "fr": {"ja": "japonais", "zh": "chinois", "ru": "russe"},
    "it": {"ja": "giapponese", "zh": "cinese", "ru": "russo"},
    "pt": {"ja": "japonês", "zh": "chinês", "ru": "russo"},
    "nl": {"ja": "Japans", "zh": "Chinees", "ru": "Russisch"},
    "pl": {"ja": "japoński", "zh": "chiński", "ru": "rosyjski"},
    "ru": {"ja": "Japanese", "zh": "Chinese", "ru": "Russian"},
    "ja": {"ja": "Japanese", "zh": "Chinese", "ru": "Russian"},
    "zh": {"ja": "Japanese", "zh": "Chinese", "ru": "Russian"},
    "tr": {"ja": "Japonca", "zh": "Çince", "ru": "Rusça"},
}

WORLD_EXONYMS: dict[str, dict[str, str]] = {
    "en": {
        "hi": "Hindi",
        "ar": "Arabic",
        "id": "Indonesian",
        "vi": "Vietnamese",
    },
    "de": {
        "hi": "Hindi",
        "ar": "Arabisch",
        "id": "Indonesisch",
        "vi": "Vietnamesisch",
    },
    "es": {
        "hi": "hindi",
        "ar": "árabe",
        "id": "indonesio",
        "vi": "vietnamita",
    },
    "fr": {
        "hi": "hindi",
        "ar": "arabe",
        "id": "indonésien",
        "vi": "vietnamien",
    },
    "it": {
        "hi": "hindi",
        "ar": "arabo",
        "id": "indonesiano",
        "vi": "vietnamita",
    },
    "pt": {
        "hi": "hindi",
        "ar": "árabe",
        "id": "indonésio",
        "vi": "vietnamita",
    },
    "nl": {
        "hi": "Hindi",
        "ar": "Arabisch",
        "id": "Indonesisch",
        "vi": "Vietnamees",
    },
    "pl": {
        "hi": "hindi",
        "ar": "arabski",
        "id": "indonezyjski",
        "vi": "wietnamski",
    },
    "tr": {
        "hi": "Hintçe",
        "ar": "Arapça",
        "id": "Endonezce",
        "vi": "Vietnamca",
    },
}


def picker_exonyms_for(locale_id: str) -> dict[str, str]:
    base = dict(WORLD_EXONYMS.get(locale_id, WORLD_EXONYMS["en"]))
    base.update(SCRIPT_EXONYMS.get(locale_id, SCRIPT_EXONYMS.get("en", {})))
    return base


def main() -> int:
    locales_dir = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(__file__).resolve().parent.parent / "locales"
    manifest = LocaleManifest.load(locales_dir)
    changed = 0
    for locale_id in manifest.locale_ids:
        exonyms = picker_exonyms_for(locale_id)
        path = locales_dir / locale_id / "common.json"
        if not path.is_file():
            continue
        data = load_json(path)
        names = data.get("locale_names")
        if not isinstance(names, dict):
            continue
        for key, value in exonyms.items():
            if names.get(key) != value:
                set_nested(data, f"locale_names.{key}", value)
                changed += 1
        path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"updated {changed} locale_names entries")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
