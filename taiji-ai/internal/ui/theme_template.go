package ui

import "fmt"

// ThemeTemplate returns a commented Tuinator stylesheet starter for Taiji TUI.
// name is the theme id (filename without .theme); used only in the header comment.
func ThemeTemplate(name string) string {
	if name == "" {
		name = "custom"
	}
	return fmt.Sprintf(`# Taiji TUI color theme: %s
#
# Save as: ~/.config/taiji/themes/%s.theme
# Theme id (Settings → Appearance → Color theme): %s
#
# Syntax: Tuinator stylesheet (.theme) — CSS-like rules + an @theme { } token block.
# Lines starting with # are comments. Use $token inside rules to reference @theme colors.
#
# Built-in themes (embedded in taiji-tui): dark, high_contrast, gruvbox_*, catppuccin_*, nord, dracula, rose_pine, solarized_dark, everforest_dark, one_dark
# Your file overrides a built-in when the id matches (e.g. dark.theme).
#
# After editing: restart taiji-tui or pick the theme in Settings, then save (Ctrl+S).
# Or set in .taiji/config.yaml:  ui: { theme: %s }
#
# ── @theme block ─────────────────────────────────────────────────────────────
# Global options and named colors. Taiji reads these tokens for chat chrome,
# sidebars, activity log, composer, scrollbars, and Tuinator widget defaults.
#
# glyph-set: unicode | ascii | unicode_rounded | auto
# border-style: light | heavy | rounded | double | ascii  (panel/frame drawing)
#
# Semantic surface colors (used by custom Taiji paint code):
#   main-bg              chat / main column background
#   sidebar-bg           session list, context panel, footer strip
#   secondary-text       de-emphasized labels, hints
#   section-heading      sidebar section titles, activity headers
#   session-active-bg    highlighted session row background
#   session-active-bar   accent bar on active session
#   badge-bg             pills, inline code background
#   quick-action-pill-bg empty-state shortcut chips
#   quick-action-code      icon tint for “code” shortcuts
#   quick-action-test      icon tint for “test” shortcuts
#   quick-action-git       icon tint for “git” shortcuts
#   logo-shadow / logo-top / logo-mid / logo-bottom   empty-state logo gradient
#   status-ok / status-warn / status-idle             MCP dots, git hints
#   git-branch-accent    branch name on empty state
#   input-chrome-fill    composer frame, mic button, palette chrome
#
# Tuinator theme slots (Label, Button, TextInput, …):
#   text, heading, muted, accent, label, button, border
#   input-fg, input-bg, thinking-fg (agent “thinking” line)
#   button-focused-fg, button-focused-bg
#   user-bubble            (reserved; user bubble accent often follows accent)
#   scrollbar-track, scrollbar-thumb-fg, scrollbar-thumb-bg
#
# Color values: #rrggbb hex, named colors (white, cyan, …), modifiers bold dim reverse,
# or references like $accent. Example: color: #89b4fa bold;

@theme {
  glyph-set: unicode;
  # border-style: rounded;

  # --- surfaces ---
  main-bg: #1a1b26;
  sidebar-bg: #16161e;
  secondary-text: #565f89;
  section-heading: #a9b1d6;
  session-active-bg: #2d243b;
  session-active-bar: #cba6f7;
  badge-bg: #292e42;
  quick-action-pill-bg: #24273a;
  quick-action-code: #89b4fa;
  quick-action-test: #94e2d5;
  quick-action-git: #fab387;
  logo-shadow: #45475a;
  logo-top: #b4befe;
  logo-mid: #cba6f7;
  logo-bottom: #a58cd2;
  status-ok: #73daca;
  status-warn: #e0af68;
  status-idle: #565f89;
  git-branch-accent: #7dcfff;
  input-chrome-fill: #2a263a;

  # --- Tuinator text widgets ---
  text: #c0c3d7;
  heading: #a9b1d6;
  muted: #565f89;
  accent: #bb9af7;
  label: #c0c3d7;
  button: white;
  button-focused-fg: black bold;
  button-focused-bg: cyan;
  border: dim cyan;
  input-fg: #e1dcf0;
  input-bg: #2a263a;
  thinking-fg: #ff8c40 bold;
  user-bubble: #bb9af7;
  scrollbar-track: #3a344a;
  scrollbar-thumb-fg: #bca8f5;
  scrollbar-thumb-bg: #4e4468;
}

# ── Widget rules ─────────────────────────────────────────────────────────────
# Selectors:
#   *                 all widgets
#   Label             widget type (Tuinator class name)
#   #my-id            widget id (set_widget_id in C++)
#   .sidebar          widget class (add_widget_class)
#   Label:focused     pseudo-state (:focused :checked :hover …)
#   .sidebar Label   descendant (Label inside .sidebar)
#
# Common properties:
#   color / foreground          text color
#   background / bg             fill behind text
#   bold / dim / reverse        true | false
#   border-color                Panel borders
#   title-color                 Panel title
#   divider-color               SplitPane divider
#   focused-color / focused-background
#   selected-color              ListView, Table, Tabs
#   gap / padding               VBox, HBox layout (integer cells)
#   first-size / min-pane-size  SplitPane
#   outer-border                SplitPane true | false
#   border-top/right/bottom/left  Panel edges true | false
#   min-width, placeholder      TextInput
#   glyph-set / border-style    per-widget override (rare)

* {
  color: $text;
}

Label {
  color: $label;
}

TextInput {
  color: $input-fg;
  background: $input-bg;
}

TextInput:focused {
  color: white bold;
}

Button:focused {
  color: $button-focused-fg;
  background: $button-focused-bg;
}

Panel {
  border-color: $border;
  # title-color: bold $accent;
}

SplitPane {
  divider-color: dim white;
  # first-size: 28;
  # min-pane-size: 6;
}

# Example: style a widget class you add in code:
# .my-panel {
#   border-color: $accent;
#   padding: 1;
# }
`, name, name, name, name)
}
