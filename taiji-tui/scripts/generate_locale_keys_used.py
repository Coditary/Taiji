#!/usr/bin/env python3
"""Generate C++ header listing every ui_tr/ui_shell key referenced in taiji-tui/src."""
from __future__ import annotations

import argparse
from pathlib import Path

from locale_usage import dedupe_refs, scan_sources


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("src_dir", type=Path)
    parser.add_argument("output_header", type=Path)
    args = parser.parse_args()

    refs = dedupe_refs(scan_sources(args.src_dir)[0])
    lines = [
        "#pragma once",
        "",
        "#include <cstddef>",
        "",
        "struct TaijiLocaleKeyRef {",
        "    const char* module;",
        "    const char* key;",
        "};",
        "",
        f"inline constexpr std::size_t kTaijiLocaleKeyUsedCount = {len(refs)};",
        "inline constexpr TaijiLocaleKeyRef kTaijiLocaleKeysUsed[] = {",
    ]
    for ref in refs:
        lines.append(f'    {{"{ref.module}", "{ref.key}"}},')
    lines.append("};")
    lines.append("")

    args.output_header.parent.mkdir(parents=True, exist_ok=True)
    args.output_header.write_text("\n".join(lines) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
