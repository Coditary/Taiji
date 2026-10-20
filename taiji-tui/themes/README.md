# Taiji UI themes

Built-in themes ship inside `taiji-tui`:

`dark` (Tokyo Night), `high_contrast`, `gruvbox_dark`, `gruvbox_light`, `catppuccin_mocha`, `catppuccin_latte`, `nord`, `dracula`, `rose_pine`, `solarized_dark`, `everforest_dark`, `one_dark`

You can override them or add your own under:

```text
~/.config/taiji/themes/
```

(or `$XDG_CONFIG_HOME/taiji/themes/`)

## Custom theme

1. Run **`taiji theme init [name]`** to generate a commented starter file (or copy `dark.theme`).
2. Default output: `~/.config/taiji/themes/<name>.theme` (the **filename without `.theme`** is the theme id).
3. In Taiji: **Settings → Appearance → Color theme** — fuzzy picker with live preview while browsing; custom ids appear in the list.
4. Set `ui_theme: my-theme` in `.taiji/config.yaml` if you want it persisted.

Only use letters, digits, `_` and `-` in the filename.

### Optional `@theme` tokens

Taiji reads extra keys from `@theme` (with sensible fallbacks if omitted):

`overlay-bg`, `list-active-bg`, `list-active-fg`, `border-chrome`, `border-chrome-alt`, `danger`, `toggle-off`, `caret-fg`, `caret-bg`, `emphasis`, `input-focused-fg`, `user-bubble`

Built-in themes use defaults derived from `sidebar-bg`, `session-active-*`, `accent`, and `text`.
