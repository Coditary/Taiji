# Taiji TUI locales

Interface strings for the settings UI and shared chrome live under `locales/` as JSON modules. At build time `scripts/embed_locales.py` merges each locale with the English master and emits `ui_locales_embed.cpp` (TUI) and `taiji-ai/internal/uilocale/catalog_gen.go` (embed / turn metrics / tool titles).

## Layout

- `manifest.json` — supported locale ids, human names, and module list (`common`, `settings`, `shell`, `tools`).
- `en/<module>.json` — **master catalog**; every key must exist here first.
- `<locale>/<module>.json` — overrides merged on top of `en` at build time.
- `key-catalog.json` — generated list of every master key (regenerated on build; safe to commit for review).
- `i18n-status.json` — generated per-locale coverage (override count, untranslated vs English).

## Modules

| Module     | Contents |
|-----------|----------|
| `common`  | App chrome, theme picker, `locale_names.*` for the language picker |
| `settings`| Categories, section headers, row `label` / `hint`, enum display text under `options.<group>.<value_id>` and `values.*` |
| `shell`   | Main chrome: sidebar, context panel, empty chat, footer status, session menu |
| `tools`   | Tool activity titles (`names.*`), activity scopes (`scopes.*`), turn-metrics suffixes (`metrics.*`) |

Missing `shell.json` / `tools.json` in a locale directory is allowed: those strings fall back to English via merge at embed time. Row **hints** are not inherited from `en` unless the locale file defines them (`sanitize_settings_locale` in `locale_lib.py`).

## Validation

Before embed, CMake runs:

```bash
python3 scripts/check_locales.py locales/
```

- **Fails the build** (`--require-complete`) only for `unknown_key` or **empty** required strings (`""`). Use `...` for “not translated yet” (does not fail the build; runtime falls back to English). Do not use `""` for pending work — that blocks the build and shows blank text in the TUI.
- Writes `key-catalog.json` and `i18n-status.json`.

Useful commands while translating:

```bash
# Summary table + status files
python3 scripts/check_locales.py locales/

# Keys still missing in the French override files
python3 scripts/check_locales.py locales/ --locale fr

# CI gate for “fully translated” locales (example: German)
python3 scripts/check_locales.py locales/ --min-translated-pct 85 --locale de
```

A key is **untranslated** when the merged value is `...` or still identical to English. Run `python3 scripts/normalize_locale_placeholders.py locales/` to turn English copies into `...`.

## Runtime API (`ui_locale.hpp`)

- `ui_set_locale(id)` — active locale (`ui_locale` in chat settings / `cfg.UI.Locale` in taiji-ai).
- `ui_tr("settings", "categories.model")` — dotted key path inside a module.
- `ui_settings_row_label` / `ui_settings_row_hint` — helpers for settings rows.

## Machine translation (DeepL, batched)

Use **one env var** for the API key (never commit it):

```bash
export DEEPL_AUTH_KEY='your-key:fx'   # free API keys end with :fx
# optional: export DEEPL_API_URL='https://api-free.deepl.com/v2/translate'

# Preview request count (deduped strings, ~50 texts per HTTP call)
python3 scripts/translate_locales.py locales/ --all --dry-run

# Translate everything that still matches English
python3 scripts/translate_locales.py locales/ --all --backend deepl

# Optional: loanwords still identical to English (Markdown, MCP, …)
python3 scripts/polish_locale_residuals.py
```

Strings are **deduplicated per locale** and sent in batches to save quota. Placeholders like `{count}` are wrapped for DeepL `tag_handling=xml`. Results are cached in `locales/.translate-cache.json`.

## Adding or editing translations

1. Add new strings to `en/<module>.json` first, then mirror in other locales.
2. Run `python3 scripts/check_locales.py locales/` and fix any `unknown_key` errors.
3. Rebuild `taiji-tui` (CMake regenerates the embed table).
4. Pick **Interface language** under Settings → Appearance to verify.

New locales: add an entry to `manifest.json` (include a `native` autonym), run `python3 scripts/sync_locale_autonyms.py locales/`, then create `<id>/settings.json` etc.

The settings language picker shows each locale’s **autonym** (`manifest.json` → `native`, stored in `locale_names.<id>`).
