#include "ui_navigation.hpp"

#include "ui_locale.hpp"
#include "ui_theme.hpp"

#include <algorithm>
#include <cctype>

namespace {

bool g_vim_modal = false;
UiFocusPane g_focus_pane = UiFocusPane::Prompt;
UiInputMode g_input_mode = UiInputMode::Insert;

} // namespace

UiKeybindingStyle ui_parse_keybinding_style(const std::string& value) {
    std::string lower = value;
    for (char& ch : lower) {
        ch = static_cast<char>(std::tolower(static_cast<unsigned char>(ch)));
    }
    if (lower == "vim" || lower == "modal" || lower == "vim_modal") {
        return UiKeybindingStyle::VimModal;
    }
    return UiKeybindingStyle::Standard;
}

std::string ui_keybinding_style_id(UiKeybindingStyle style) {
    return style == UiKeybindingStyle::VimModal ? "vim" : "standard";
}

std::string ui_keybinding_style_label(UiKeybindingStyle style) {
    return style == UiKeybindingStyle::VimModal ? "Vim (modal)" : "Standard";
}

void ui_set_navigation_chrome(bool vim_modal, UiFocusPane pane, UiInputMode mode) {
    g_vim_modal = vim_modal;
    g_focus_pane = pane;
    g_input_mode = mode;
}

bool ui_vim_modal_chrome_enabled() { return g_vim_modal; }

UiFocusPane ui_focus_pane() { return g_focus_pane; }

UiInputMode ui_input_mode() { return g_input_mode; }

void ui_paint_vim_focus_accent(tuinator::PaintContext& ctx, UiFocusPane pane, int height) {
    if (!g_vim_modal || g_focus_pane != pane || height <= 0) {
        return;
    }
    tuinator::Rgb accent = ui_quick_action_git();
    if (pane == UiFocusPane::Prompt && g_input_mode == UiInputMode::Insert) {
        accent = ui_quick_action_code();
    }
    tuinator::Style bar;
    bar.background_rgb = accent;
    bar.foreground_rgb = accent;
    ctx.canvas.fill_rect({0, 0, 1, height}, ' ', bar);
}

bool ui_key_ctrl_char(const tuinator::KeyPress& key, char ch, bool allow_shift) {
    if (!key.ctrl || key.alt) {
        return false;
    }
    if (!allow_shift && key.shift) {
        return false;
    }
    const char target = static_cast<char>(std::tolower(static_cast<unsigned char>(ch)));
    if (key.character != '\0') {
        return static_cast<char>(std::tolower(static_cast<unsigned char>(key.character))) == target;
    }
    return false;
}

bool ui_key_plain_char(const tuinator::KeyPress& key, char ch) {
    if (key.ctrl || key.alt) {
        return false;
    }
    const char target = static_cast<char>(std::tolower(static_cast<unsigned char>(ch)));
    if (key.character != '\0') {
        return static_cast<char>(std::tolower(static_cast<unsigned char>(key.character))) == target;
    }
    return false;
}

std::string ui_footer_shortcuts_for_state(bool vim_modal, UiInputMode mode) {
    if (!vim_modal) {
        return ui_default_footer_shortcuts();
    }
    if (mode == UiInputMode::Normal) {
        return "i: insert · a: append · Ctrl+h/l/j/k: panes · Ctrl+T: themes · Ctrl+s: settings";
    }
    return "Esc: normal · Ctrl+h/l/j/k: panes · Shift+Enter: newline · Enter: send";
}
