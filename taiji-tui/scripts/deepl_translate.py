"""DeepL batch translation helpers (minimal requests, deduplicated strings)."""
from __future__ import annotations

import json
import re
import time
import urllib.error
import urllib.request
from typing import Iterable

PH_RE = re.compile(r"\{[^{}]+\}")

# Taiji locale id -> DeepL target_lang (free/pro API codes).
DEEPL_TARGET_LANG = {
    "de": "DE",
    "es": "ES",
    "fr": "FR",
    "it": "IT",
    "pt": "PT-PT",
    "nl": "NL",
    "pl": "PL",
    "ru": "RU",
    "ja": "JA",
    "zh": "ZH-HANS",
    "hi": "HI",
    "ar": "AR",
    "id": "ID",
    "vi": "VI",
    "tr": "TR",
}

DEFAULT_API_URL = "https://api-free.deepl.com/v2/translate"
MAX_TEXTS_PER_REQUEST = 50
MAX_CHARS_PER_REQUEST = 12_000


def protect_placeholders(text: str) -> tuple[str, dict[str, str]]:
    """Use opaque tokens DeepL usually leaves unchanged (no XML — avoids '&' in UI copy)."""
    tokens: dict[str, str] = {}

    def repl(match: re.Match[str]) -> str:
        token = f"__PH{len(tokens)}__"
        tokens[token] = match.group(0)
        return token

    return PH_RE.sub(repl, text), tokens


def restore_placeholders(text: str, tokens: dict[str, str]) -> str:
    for token, value in tokens.items():
        text = text.replace(token, value)
    return text


def _chunk_pairs(pairs: list[tuple[str, dict[str, str]]]) -> list[list[tuple[str, dict[str, str]]]]:
    batches: list[list[tuple[str, dict[str, str]]]] = []
    current: list[tuple[str, dict[str, str]]] = []
    current_chars = 0
    for text, tokens in pairs:
        size = len(text) + 2
        if current and (
            len(current) >= MAX_TEXTS_PER_REQUEST
            or current_chars + size > MAX_CHARS_PER_REQUEST
        ):
            batches.append(current)
            current = []
            current_chars = 0
        current.append((text, tokens))
        current_chars += size
    if current:
        batches.append(current)
    return batches


def translate_batch(
    texts: list[str],
    target_lang: str,
    auth_key: str,
    *,
    api_url: str = DEFAULT_API_URL,
    source_lang: str = "EN",
    pause_s: float = 0.2,
) -> list[str]:
    """Translate many strings in few HTTP calls. Order matches input."""
    if not texts:
        return []
    protected_pairs = [protect_placeholders(t) for t in texts]
    out: list[str] = []
    for batch in _chunk_pairs(protected_pairs):
        batch_texts = [text for text, _tokens in batch]
        payload = json.dumps(
            {
                "text": batch_texts,
                "source_lang": source_lang,
                "target_lang": target_lang,
                "preserve_formatting": True,
            }
        ).encode("utf-8")
        request = urllib.request.Request(
            api_url,
            data=payload,
            headers={
                "Authorization": f"DeepL-Auth-Key {auth_key}",
                "Content-Type": "application/json",
            },
            method="POST",
        )
        for attempt in range(4):
            try:
                with urllib.request.urlopen(request, timeout=120) as response:
                    body = json.loads(response.read().decode("utf-8"))
                break
            except urllib.error.HTTPError as exc:
                if exc.code == 429 and attempt < 3:
                    time.sleep(2.0 * (attempt + 1))
                    continue
                detail = exc.read().decode("utf-8", errors="replace")
                raise RuntimeError(f"DeepL HTTP {exc.code}: {detail}") from exc
        else:
            raise RuntimeError("DeepL request failed after retries")

        translations = body.get("translations") or []
        if len(translations) != len(batch_texts):
            raise RuntimeError(
                f"DeepL returned {len(translations)} texts for batch of {len(batch_texts)}"
            )
        for ( _text, tokens), item in zip(batch, translations, strict=True):
            raw = str(item.get("text", ""))
            out.append(restore_placeholders(raw, tokens))
        if pause_s > 0:
            time.sleep(pause_s)
    return out


def translate_unique(
    unique_texts: Iterable[str],
    target_lang: str,
    auth_key: str,
    cache: dict[str, str],
    cache_prefix: str,
    *,
    retry_prefix: str = "",
    **kwargs,
) -> dict[str, str]:
    """Translate deduplicated strings; fill cache."""
    texts = [t for t in unique_texts if t]
    pending = [t for t in texts if cache.get(f"{cache_prefix}|{t}") in (None, t)]

    if pending:
        api_input = [f"{retry_prefix}{t}" if retry_prefix else t for t in pending]
        translated_list = translate_batch(api_input, target_lang, auth_key, **kwargs)
        for src, dst in zip(pending, translated_list, strict=True):
            if retry_prefix and dst.startswith(retry_prefix):
                dst = dst[len(retry_prefix) :].lstrip()
            cache[f"{cache_prefix}|{src}"] = dst

    return {t: cache.get(f"{cache_prefix}|{t}", t) for t in texts}
