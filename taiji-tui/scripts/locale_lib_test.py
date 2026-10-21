#!/usr/bin/env python3
from __future__ import annotations

import json
import tempfile
import unittest
from pathlib import Path

from locale_lib import (
    LOCALE_PLACEHOLDER,
    LocaleManifest,
    is_blocking_empty_value,
    validate_locales,
    write_key_catalog,
)


class LocaleLibTest(unittest.TestCase):
    def test_empty_value_blocking(self) -> None:
        self.assertFalse(is_blocking_empty_value("", ""))
        self.assertFalse(is_blocking_empty_value("Save", LOCALE_PLACEHOLDER))
        self.assertFalse(is_blocking_empty_value("Save", "Save"))
        self.assertTrue(is_blocking_empty_value("Save", ""))
        self.assertTrue(is_blocking_empty_value("Save", "   "))

    def test_unknown_key_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "manifest.json").write_text(
                json.dumps(
                    {
                        "default_locale": "en",
                        "modules": ["common"],
                        "locales": [{"id": "en"}, {"id": "de"}],
                    }
                ),
                encoding="utf-8",
            )
            (root / "en").mkdir()
            (root / "de").mkdir()
            (root / "en" / "common.json").write_text('{"app": {"title": "Hi"}}', encoding="utf-8")
            (root / "de" / "common.json").write_text('{"app": {"typo": "Hallo"}}', encoding="utf-8")

            manifest = LocaleManifest.load(root)
            issues, _ = validate_locales(manifest)
            self.assertTrue(any(issue.kind == "unknown_key" for issue in issues))

    def test_key_catalog_lists_master_keys(self) -> None:
        locales_dir = Path(__file__).resolve().parent.parent / "locales"
        manifest = LocaleManifest.load(locales_dir)
        with tempfile.TemporaryDirectory() as tmp:
            out = Path(tmp) / "key-catalog.json"
            write_key_catalog(manifest, out)
            data = json.loads(out.read_text(encoding="utf-8"))
            self.assertEqual(data["default_locale"], "en")
            self.assertGreater(data["key_count"], 300)
            self.assertIn("settings", data["keys_by_module"])


if __name__ == "__main__":
    unittest.main()
