"""Extract ui_tr / ui_shell references from taiji-tui C++ sources."""
from __future__ import annotations

import re
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class LocaleRef:
    module: str
    key: str
    file: str
    line: int

    @property
    def dotted(self) -> str:
        return f"{self.module}.{self.key}"


_RE_UI_TR_TWO = re.compile(
    r"""ui_tr\s*\(\s*"([^"]+)"\s*,\s*"([^"]+)"\s*\)"""
)
_RE_UI_TR_ONE = re.compile(
    r"""ui_tr\s*\(\s*"([^"]+)"\s*\)"""
)
_RE_UI_SHELL = re.compile(
    r"""ui_shell\s*\(\s*"([^"]+)"\s*\)"""
)
# English literals used when ui_shell/ui_tr returns empty (i18n smell).
_RE_EMPTY_FALLBACK = re.compile(
    r"""ui_(?:shell|tr)\([^)]+\)\.empty\(\)\s*\?\s*"([^"]{3,})"\s*:"""
)

_SKIP_SRC = {"ui_locales_embed.cpp"}


def scan_sources(src_dir: Path) -> tuple[list[LocaleRef], list[tuple[str, int, str]]]:
    refs: list[LocaleRef] = []
    fallbacks: list[tuple[str, int, str]] = []
    for path in sorted(src_dir.rglob("*.cpp")):
        if path.name in _SKIP_SRC:
            continue
        text = path.read_text(encoding="utf-8")
        rel = str(path.relative_to(src_dir.parent))
        for line_no, line in enumerate(text.splitlines(), start=1):
            for module, key in _RE_UI_TR_TWO.findall(line):
                refs.append(LocaleRef(module, key, rel, line_no))
            for key in _RE_UI_TR_ONE.findall(line):
                refs.append(LocaleRef("common", key, rel, line_no))
            for key in _RE_UI_SHELL.findall(line):
                refs.append(LocaleRef("shell", key, rel, line_no))
            for literal in _RE_EMPTY_FALLBACK.findall(line):
                fallbacks.append((rel, line_no, literal))
    return refs, fallbacks


def dedupe_refs(refs: list[LocaleRef]) -> list[LocaleRef]:
    seen: set[tuple[str, str]] = set()
    out: list[LocaleRef] = []
    for ref in refs:
        token = (ref.module, ref.key)
        if token in seen:
            continue
        seen.add(token)
        out.append(ref)
    return sorted(out, key=lambda r: (r.module, r.key))
