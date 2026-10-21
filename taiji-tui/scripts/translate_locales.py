#!/usr/bin/env python3
"""Translate locale override strings that still match the English master."""
from __future__ import annotations

import argparse
import json
import os
import sys
import time
from pathlib import Path

from deepl_translate import DEEPL_TARGET_LANG, DEFAULT_API_URL, translate_unique
from locale_lib import LOCALE_PLACEHOLDER, LocaleManifest, load_json, set_nested, validate_locales

PH_RE = __import__("re").compile(r"\{[^{}]+\}")

GOOGLE_TARGETS = {"zh": "zh-CN"}

MYMEMORY_TARGETS = {
    "de": "de-DE",
    "es": "es-ES",
    "fr": "fr-FR",
    "it": "it-IT",
    "pt": "pt-PT",
    "nl": "nl-NL",
    "pl": "pl-PL",
    "ru": "ru-RU",
    "ja": "ja-JP",
    "zh": "zh-CN",
    "hi": "hi-IN",
    "ar": "ar-SA",
    "id": "id-ID",
    "vi": "vi-VN",
    "tr": "tr-TR",
}


def protect_placeholders(text: str) -> tuple[str, dict[str, str]]:
    tokens: dict[str, str] = {}

    def repl(match) -> str:
        token = f"⟦{len(tokens)}⟧"
        tokens[token] = match.group(0)
        return token

    return PH_RE.sub(repl, text), tokens


def restore_placeholders(text: str, tokens: dict[str, str]) -> str:
    for token, value in tokens.items():
        text = text.replace(token, value)
    return text


def make_google_translator(target: str):
    from deep_translator import GoogleTranslator, MyMemoryTranslator

    google_target = GOOGLE_TARGETS.get(target, target)
    memory_target = MYMEMORY_TARGETS.get(target, google_target)
    google = GoogleTranslator(source="en", target=google_target)
    memory = MyMemoryTranslator(source="en-US", target=memory_target)

    def translate(protected: str) -> str:
        for attempt, backend in enumerate((google, memory)):
            try:
                return backend.translate(protected)
            except Exception:
                if attempt == 0:
                    time.sleep(1.5)
                    continue
                raise
        return protected

    return translate


def translate_text_google(text: str, target: str, translate_fn) -> str:
    if not text.strip():
        return text
    protected, tokens = protect_placeholders(text)
    out = translate_fn(protected)
    return restore_placeholders(out, tokens)


def apply_locale_translations(
    locales_dir: Path,
    manifest: LocaleManifest,
    locale_id: str,
    cov,
    master: dict,
    value_map: dict[str, str],
) -> int:
    overrides = manifest.override_flat(locale_id)
    by_module: dict[str, dict] = {}
    for module in manifest.modules:
        path = locales_dir / locale_id / f"{module}.json"
        by_module[module] = load_json(path) if path.is_file() else {}

    changed = 0
    for dotted in cov.untranslated_keys:
        module, key = dotted.split(".", 1)
        en_value = master.get((module, key), "")
        if not en_value:
            continue
        current = overrides.get((module, key), en_value)
        if current not in (en_value, LOCALE_PLACEHOLDER):
            continue
        translated = value_map.get(en_value)
        if not translated or translated == en_value:
            translated = LOCALE_PLACEHOLDER
        set_nested(by_module[module], key, translated)
        changed += 1

    for module, data in by_module.items():
        out = locales_dir / locale_id / f"{module}.json"
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    return changed


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("locales_dir", nargs="?", default=None)
    parser.add_argument("--locale", action="append", metavar="ID")
    parser.add_argument("--all", action="store_true", help="translate every non-English locale")
    parser.add_argument(
        "--backend",
        choices=("deepl", "google"),
        default="deepl",
        help="translation provider (default: deepl)",
    )
    parser.add_argument("--sleep", type=float, default=0.4, help="google: seconds between API calls")
    parser.add_argument("--cache", metavar="PATH", help="JSON cache file for translations")
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="only print how many unique strings / API batches would be used",
    )
    args = parser.parse_args()

    script_dir = Path(__file__).resolve().parent
    locales_dir = Path(args.locales_dir) if args.locales_dir else script_dir.parent / "locales"
    cache_path = Path(args.cache) if args.cache else locales_dir / ".translate-cache.json"
    cache: dict[str, str] = {}
    if cache_path.is_file():
        cache = json.loads(cache_path.read_text(encoding="utf-8"))

    manifest = LocaleManifest.load(locales_dir)
    master = manifest.master_catalog()
    _, coverages = validate_locales(manifest)

    targets = args.locale
    if args.all:
        targets = [c.locale for c in coverages if c.locale != manifest.default_locale]
    if not targets:
        print("pass --locale <id> or --all", file=sys.stderr)
        return 1

    auth_key = os.environ.get("DEEPL_AUTH_KEY", "").strip()
    api_url = os.environ.get("DEEPL_API_URL", DEFAULT_API_URL).strip()
    if args.backend == "deepl" and not args.dry_run and not auth_key:
        print("set DEEPL_AUTH_KEY (never commit it to git)", file=sys.stderr)
        return 1

    total_requests_estimate = 0
    for locale_id in targets:
        if locale_id == manifest.default_locale:
            continue
        cov = next(c for c in coverages if c.locale == locale_id)
        if not cov.untranslated_keys:
            print(f"{locale_id}: already localized", flush=True)
            continue

        unique_en: list[str] = []
        seen: set[str] = set()
        for dotted in cov.untranslated_keys:
            module, key = dotted.split(".", 1)
            en_value = master.get((module, key), "")
            if not en_value or en_value in seen:
                continue
            seen.add(en_value)
            unique_en.append(en_value)

        pending: list[str] = []
        for text in unique_en:
            cache_key = f"{locale_id}|{text}"
            cached = cache.get(cache_key)
            if cached is not None and cached != text:
                continue
            if cached == text:
                del cache[cache_key]
            pending.append(text)
        from deepl_translate import _chunk_pairs

        batches = (
            len(_chunk_pairs([(t, {}) for t in pending])) if pending else 0
        )
        total_requests_estimate += batches
        print(
            f"{locale_id}: {len(cov.untranslated_keys)} keys, "
            f"{len(unique_en)} unique strings, "
            f"{len(pending)} to fetch ({batches} DeepL request(s))",
            flush=True,
        )
        if args.dry_run:
            continue

        if args.backend == "deepl":
            target_lang = DEEPL_TARGET_LANG.get(locale_id)
            if not target_lang:
                print(f"unsupported DeepL target for locale {locale_id}", file=sys.stderr)
                return 1
            retry_prefix = ""
            if pending and all(cache.get(f"{locale_id}|{t}") == t for t in pending):
                retry_prefix = "User interface label: "
            value_map = translate_unique(
                unique_en,
                target_lang,
                auth_key,
                cache,
                locale_id,
                api_url=api_url,
                retry_prefix=retry_prefix,
            )
            cache_path.write_text(json.dumps(cache, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
        else:
            try:
                from deep_translator import GoogleTranslator  # noqa: F401
            except ImportError:
                print("install deep-translator: pip install deep-translator", file=sys.stderr)
                return 1
            translate_fn = make_google_translator(locale_id)
            value_map = {}
            for en_value in unique_en:
                cache_key = f"{locale_id}|{en_value}"
                if cache_key in cache:
                    value_map[en_value] = cache[cache_key]
                    continue
                translated = translate_text_google(en_value, locale_id, translate_fn)
                cache[cache_key] = translated
                value_map[en_value] = translated
                cache_path.write_text(json.dumps(cache, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
                time.sleep(args.sleep)

        changed = apply_locale_translations(
            locales_dir, manifest, locale_id, cov, master, value_map
        )
        print(f"{locale_id}: wrote {changed} strings", flush=True)
        _, coverages = validate_locales(manifest)

    if args.dry_run:
        print(f"estimated DeepL requests (all listed locales): {total_requests_estimate}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
