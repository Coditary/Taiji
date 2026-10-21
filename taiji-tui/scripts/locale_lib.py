"""Shared locale catalog loading, merge, and validation for taiji-tui."""
from __future__ import annotations

import json
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any


def load_json(path: Path) -> dict[str, Any]:
    if not path.is_file():
        return {}
    with path.open(encoding="utf-8") as handle:
        data = json.load(handle)
    if not isinstance(data, dict):
        raise TypeError(f"{path}: expected JSON object at root")
    return data


def flatten(obj: object, prefix: str = "") -> list[tuple[str, str]]:
    pairs: list[tuple[str, str]] = []
    if isinstance(obj, dict):
        for key, value in obj.items():
            child = f"{prefix}.{key}" if prefix else str(key)
            pairs.extend(flatten(value, child))
    elif isinstance(obj, str):
        pairs.append((prefix, obj))
    else:
        raise TypeError(f"unsupported locale value at {prefix}: {type(obj)}")
    return pairs


def set_nested(root: dict[str, Any], dotted: str, value: str) -> None:
    parts = dotted.split(".")
    cur: dict[str, Any] = root
    for part in parts[:-1]:
        nxt = cur.get(part)
        if not isinstance(nxt, dict):
            nxt = {}
            cur[part] = nxt
        cur = nxt
    cur[parts[-1]] = value


def deep_merge(base: dict[str, Any], override: dict[str, Any]) -> dict[str, Any]:
    out = dict(base)
    for key, value in override.items():
        if isinstance(value, dict) and isinstance(out.get(key), dict):
            out[key] = deep_merge(out[key], value)
        else:
            out[key] = value
    return out


def sanitize_settings_locale(
    locale_id: str, default_locale: str, merged: dict[str, Any], override: dict[str, Any]
) -> dict[str, Any]:
    """Drop English-only keys that partial locale files did not override."""
    if locale_id == default_locale:
        return merged
    override = override or {}
    out = dict(merged)

    rows = out.get("rows")
    override_rows = override.get("rows") or {}
    if isinstance(rows, dict):
        cleaned_rows: dict[str, Any] = {}
        for row_key, row in rows.items():
            if not isinstance(row, dict):
                cleaned_rows[row_key] = row
                continue
            row = dict(row)
            override_row = override_rows.get(row_key)
            if not isinstance(override_row, dict) or "hint" not in override_row:
                row.pop("hint", None)
            cleaned_rows[row_key] = row
        out["rows"] = cleaned_rows

    return out


@dataclass
class LocaleManifest:
    locales_dir: Path
    default_locale: str
    modules: list[str]
    locale_ids: list[str]

    @classmethod
    def load(cls, locales_dir: Path) -> LocaleManifest:
        manifest = load_json(locales_dir / "manifest.json")
        default_locale = str(manifest.get("default_locale", "en"))
        modules = list(manifest.get("modules", ["common", "settings"]))
        locale_ids = [entry["id"] for entry in manifest.get("locales", []) if "id" in entry]
        return cls(locales_dir, default_locale, modules, locale_ids)

    def master_catalog(self) -> dict[tuple[str, str], str]:
        """All dotted keys from the default locale modules."""
        catalog: dict[tuple[str, str], str] = {}
        for module in self.modules:
            path = self.locales_dir / self.default_locale / f"{module}.json"
            for key, value in flatten(load_json(path)):
                catalog[(module, key)] = value
        return catalog

    def merged_module(self, locale_id: str, module: str) -> dict[str, Any]:
        base = load_json(self.locales_dir / self.default_locale / f"{module}.json")
        override: dict[str, Any] = {}
        if locale_id != self.default_locale:
            override = load_json(self.locales_dir / locale_id / f"{module}.json")
            base = deep_merge(base, override)
        if module == "settings":
            base = sanitize_settings_locale(locale_id, self.default_locale, base, override)
        return base

    def merged_flat(self, locale_id: str) -> dict[tuple[str, str], str]:
        out: dict[tuple[str, str], str] = {}
        for module in self.modules:
            for key, value in flatten(self.merged_module(locale_id, module)):
                out[(module, key)] = value
        return out

    def override_flat(self, locale_id: str) -> dict[tuple[str, str], str]:
        if locale_id == self.default_locale:
            return self.merged_flat(locale_id)
        out: dict[tuple[str, str], str] = {}
        for module in self.modules:
            path = self.locales_dir / locale_id / f"{module}.json"
            if not path.is_file():
                continue
            for key, value in flatten(load_json(path)):
                out[(module, key)] = value
        return out


@dataclass
class ValidationIssue:
    kind: str
    locale: str
    module: str
    key: str
    message: str


LOCALE_PLACEHOLDER = "..."


def is_blocking_empty_value(english_master: str, value: str) -> bool:
    """Build fails only when English expects text but the locale value is blank (not '...')."""
    if not english_master.strip():
        return False
    stripped = value.strip()
    if stripped == LOCALE_PLACEHOLDER:
        return False
    return stripped == ""


@dataclass
class LocaleCoverage:
    locale: str
    total_keys: int
    override_keys: int
    missing_override_keys: list[str] = field(default_factory=list)
    unknown_override_keys: list[str] = field(default_factory=list)
    untranslated_keys: list[str] = field(default_factory=list)
    empty_value_keys: list[str] = field(default_factory=list)
    missing_module_files: list[str] = field(default_factory=list)

    @property
    def coverage_count(self) -> int:
        """Keys present in this locale's override JSON (explicit catalog entry)."""
        return self.total_keys - len(self.missing_override_keys)

    @property
    def translated_count(self) -> int:
        return self.coverage_count

    @property
    def translated_pct(self) -> float:
        if self.total_keys == 0:
            return 100.0
        return 100.0 * self.coverage_count / self.total_keys

    @property
    def localized_count(self) -> int:
        """Keys whose merged value differs from English (QA hint, not used for pass/fail)."""
        return self.total_keys - len(self.untranslated_keys)


def validate_locales(manifest: LocaleManifest) -> tuple[list[ValidationIssue], list[LocaleCoverage]]:
    master = manifest.master_catalog()
    master_key_set = set(master.keys())
    issues: list[ValidationIssue] = []
    coverages: list[LocaleCoverage] = []

    for locale_id in manifest.locale_ids:
        merged = manifest.merged_flat(locale_id)
        overrides = manifest.override_flat(locale_id)
        missing_modules = [
            module
            for module in manifest.modules
            if locale_id != manifest.default_locale
            and not (manifest.locales_dir / locale_id / f"{module}.json").is_file()
        ]

        unknown = sorted(
            f"{module}.{key}"
            for (module, key) in overrides
            if (module, key) not in master_key_set
        )
        for item in unknown:
            module, key = item.split(".", 1)
            issues.append(
                ValidationIssue(
                    kind="unknown_key",
                    locale=locale_id,
                    module=module,
                    key=key,
                    message=f"key not in {manifest.default_locale} master: {item}",
                )
            )

        missing_override: list[str] = []
        untranslated: list[str] = []
        empty_values: list[str] = []
        if locale_id != manifest.default_locale:
            for (module, key), en_value in master.items():
                dotted = f"{module}.{key}"
                if (module, key) not in overrides:
                    missing_override.append(dotted)
                merged_value = merged.get((module, key), en_value)
                if is_blocking_empty_value(en_value, merged_value):
                    empty_values.append(dotted)
                    issues.append(
                        ValidationIssue(
                            kind="empty_value",
                            locale=locale_id,
                            module=module,
                            key=key,
                            message=f"required string is empty (use {LOCALE_PLACEHOLDER!r} as placeholder): {dotted}",
                        )
                    )
                if not en_value:
                    continue
                if merged_value == en_value or merged_value == LOCALE_PLACEHOLDER:
                    untranslated.append(dotted)

        coverages.append(
            LocaleCoverage(
                locale=locale_id,
                total_keys=len(master),
                override_keys=len(overrides),
                missing_override_keys=missing_override,
                unknown_override_keys=unknown,
                untranslated_keys=untranslated,
                empty_value_keys=empty_values,
                missing_module_files=missing_modules,
            )
        )

    return issues, coverages


def write_key_catalog(manifest: LocaleManifest, output: Path) -> None:
    master = manifest.master_catalog()
    by_module: dict[str, list[str]] = {module: [] for module in manifest.modules}
    for module, key in sorted(master.keys()):
        by_module[module].append(key)

    payload = {
        "default_locale": manifest.default_locale,
        "modules": manifest.modules,
        "locale_ids": manifest.locale_ids,
        "key_count": len(master),
        "keys_by_module": by_module,
        "keys": [
            {"module": module, "key": key, "en": master[(module, key)]}
            for module, key in sorted(master.keys())
        ],
    }
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def write_i18n_status(
    manifest: LocaleManifest,
    coverages: list[LocaleCoverage],
    output: Path,
    *,
    include_key_lists: bool = False,
) -> None:
    locale_rows: list[dict[str, object]] = []
    for cov in coverages:
        row: dict[str, object] = {
            "id": cov.locale,
            "total_keys": cov.total_keys,
            "override_keys": cov.override_keys,
            "translated_keys": cov.translated_count,
            "translated_pct": round(cov.translated_pct, 1),
            "localized_keys": cov.localized_count,
            "localized_pct": round(
                100.0 * cov.localized_count / cov.total_keys if cov.total_keys else 100.0, 1
            ),
            "missing_module_files": cov.missing_module_files,
            "unknown_override_keys": cov.unknown_override_keys,
            "missing_override_key_count": len(cov.missing_override_keys),
            "untranslated_key_count": len(cov.untranslated_keys),
            "empty_value_key_count": len(cov.empty_value_keys),
        }
        if include_key_lists:
            row["missing_override_keys"] = cov.missing_override_keys
            row["untranslated_keys"] = cov.untranslated_keys
            row["empty_value_keys"] = cov.empty_value_keys
        locale_rows.append(row)
    payload = {
        "default_locale": manifest.default_locale,
        "locales": locale_rows,
    }
    output.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
