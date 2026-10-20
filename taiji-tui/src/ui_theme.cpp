#include "ui_theme.hpp"

#include "ui_chat_column.hpp"

#include "ui_themes_bundled.hpp"

#include <tuinator/render/stylesheet.hpp>
#include <tuinator/render/scrollbar.hpp>

#include <algorithm>
#include <array>
#include <cctype>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <memory>
#include <stdexcept>
#include <string>
#include <string_view>
#include <unordered_map>
#include <vector>

#if defined(__linux__)
#include <unistd.h>
#endif

namespace {

enum class UiCornerStyle { Square, Rounded, Flat };

UiCornerStyle g_ui_corner_style = UiCornerStyle::Square;
UiChatRolePrefs g_ui_user_chat_prefs{
    .style = UiUserMessageStyle::Messenger,
    .align = UiMessageAlign::Auto,
    .max_width_pct = 80,
    .for_assistant = false,
};
UiChatRolePrefs g_ui_assistant_chat_prefs{
    .style = UiUserMessageStyle::Messenger,
    .align = UiMessageAlign::Auto,
    .max_width_pct = 80,
    .for_assistant = true,
};
bool g_ui_message_fill_background = true;

struct UiPalette {
    tuinator::Rgb main_bg{26, 27, 38};
    tuinator::Rgb sidebar_bg{22, 22, 30};
    tuinator::Rgb secondary_text{86, 95, 137};
    tuinator::Rgb section_heading{169, 177, 214};
    tuinator::Rgb session_active_bg{45, 36, 59};
    tuinator::Rgb session_active_bar{203, 166, 247};
    tuinator::Rgb badge_bg{41, 46, 66};
    tuinator::Rgb quick_action_pill_bg{36, 39, 58};
    tuinator::Rgb quick_action_code{137, 180, 250};
    tuinator::Rgb quick_action_test{148, 226, 213};
    tuinator::Rgb quick_action_git{250, 179, 135};
    tuinator::Rgb logo_shadow{69, 71, 90};
    tuinator::Rgb logo_top{180, 190, 254};
    tuinator::Rgb logo_mid{203, 166, 247};
    tuinator::Rgb logo_bottom{165, 140, 210};
    tuinator::Rgb status_ok{115, 218, 202};
    tuinator::Rgb status_warn{224, 175, 104};
    tuinator::Rgb status_idle{86, 95, 137};
    tuinator::Rgb git_branch_accent{125, 207, 255};
    tuinator::Rgb input_chrome_fill{42, 38, 58};
    tuinator::Rgb text{192, 195, 215};
    tuinator::Rgb accent{203, 166, 247};
    tuinator::Rgb overlay_bg{22, 22, 30};
    tuinator::Rgb list_active_bg{45, 36, 59};
    tuinator::Rgb list_active_fg{255, 255, 255};
    tuinator::Rgb border_chrome{130, 120, 170};
    tuinator::Rgb border_chrome_alt{168, 148, 215};
    tuinator::Rgb danger{240, 90, 90};
    tuinator::Rgb toggle_off{100, 98, 110};
    tuinator::Rgb caret_fg{28, 24, 40};
    tuinator::Rgb caret_bg{196, 178, 255};
    tuinator::Rgb emphasis{180, 200, 255};
    tuinator::Rgb user_bubble{203, 166, 247};
};

UiPalette g_palette;
tuinator::Theme g_app_theme = tuinator::dark_theme({});
const tuinator::Stylesheet* g_active_sheet = nullptr;

UiPalette& palette_mut() { return g_palette; }

const UiPalette& palette() { return g_palette; }

tuinator::Rgb token_rgb(const tuinator::Stylesheet& sheet, const char* name, tuinator::Rgb fallback) {
    const tuinator::Style resolved = sheet.resolve_theme_token(name, {});
    if (resolved.foreground_rgb.has_value()) {
        return *resolved.foreground_rgb;
    }
    if (resolved.background_rgb.has_value()) {
        return *resolved.background_rgb;
    }
    return fallback;
}

void refresh_palette(const tuinator::Stylesheet& sheet) {
    UiPalette& p = palette_mut();
    p.main_bg = token_rgb(sheet, "main-bg", p.main_bg);
    p.sidebar_bg = token_rgb(sheet, "sidebar-bg", p.sidebar_bg);
    p.secondary_text = token_rgb(sheet, "secondary-text", p.secondary_text);
    p.section_heading = token_rgb(sheet, "section-heading", p.section_heading);
    p.session_active_bg = token_rgb(sheet, "session-active-bg", p.session_active_bg);
    p.session_active_bar = token_rgb(sheet, "session-active-bar", p.session_active_bar);
    p.badge_bg = token_rgb(sheet, "badge-bg", p.badge_bg);
    p.quick_action_pill_bg = token_rgb(sheet, "quick-action-pill-bg", p.quick_action_pill_bg);
    p.quick_action_code = token_rgb(sheet, "quick-action-code", p.quick_action_code);
    p.quick_action_test = token_rgb(sheet, "quick-action-test", p.quick_action_test);
    p.quick_action_git = token_rgb(sheet, "quick-action-git", p.quick_action_git);
    p.logo_shadow = token_rgb(sheet, "logo-shadow", p.logo_shadow);
    p.logo_top = token_rgb(sheet, "logo-top", p.logo_top);
    p.logo_mid = token_rgb(sheet, "logo-mid", p.logo_mid);
    p.logo_bottom = token_rgb(sheet, "logo-bottom", p.logo_bottom);
    p.status_ok = token_rgb(sheet, "status-ok", p.status_ok);
    p.status_warn = token_rgb(sheet, "status-warn", p.status_warn);
    p.status_idle = token_rgb(sheet, "status-idle", p.status_idle);
    p.git_branch_accent = token_rgb(sheet, "git-branch-accent", p.git_branch_accent);
    p.input_chrome_fill = token_rgb(sheet, "input-chrome-fill", p.input_chrome_fill);
    p.text = token_rgb(sheet, "text", p.text);
    p.accent = token_rgb(sheet, "accent", p.accent);
    p.overlay_bg = token_rgb(sheet, "overlay-bg", p.sidebar_bg);
    p.list_active_bg = token_rgb(sheet, "list-active-bg", p.session_active_bg);
    p.list_active_fg = token_rgb(sheet, "list-active-fg", p.text);
    p.border_chrome =
        token_rgb(sheet, "border-chrome", token_rgb(sheet, "border", token_rgb(sheet, "section-heading", p.secondary_text)));
    p.border_chrome_alt = token_rgb(sheet, "border-chrome-alt", p.session_active_bar);
    p.danger = token_rgb(sheet, "danger", p.danger);
    p.toggle_off = token_rgb(sheet, "toggle-off", p.secondary_text);
    p.caret_fg = token_rgb(sheet, "caret-fg", p.main_bg);
    p.caret_bg = token_rgb(sheet, "caret-bg", p.session_active_bar);
    p.emphasis = token_rgb(sheet, "emphasis", p.quick_action_code);
    p.user_bubble = token_rgb(sheet, "user-bubble", p.accent);
}

tuinator::Style token_style(const tuinator::Stylesheet& sheet, const char* name, tuinator::Style fallback) {
    return sheet.resolve_theme_token(name, fallback);
}

tuinator::Theme theme_from_stylesheet(const tuinator::Stylesheet& sheet) {
    const UiPalette& p = palette();
    tuinator::Theme theme = tuinator::dark_theme(sheet.theme_options());

    theme.label = token_style(sheet, "label", theme.label);
    if (!theme.label.foreground_rgb.has_value()) {
        theme.label.foreground_rgb = p.text;
    }
    theme.label.background_rgb = p.overlay_bg;

    theme.muted = token_style(sheet, "muted", theme.muted);
    theme.muted.foreground_rgb = token_rgb(sheet, "muted", p.secondary_text);
    theme.muted.background_rgb = p.overlay_bg;

    theme.heading = token_style(sheet, "heading", theme.heading);
    if (!theme.heading.foreground_rgb.has_value()) {
        theme.heading.foreground_rgb = p.section_heading;
    }
    theme.heading.background_rgb = p.overlay_bg;

    theme.accent = token_style(sheet, "accent", theme.accent);
    if (!theme.accent.foreground_rgb.has_value()) {
        theme.accent.foreground_rgb = p.accent;
    }
    theme.accent.background_rgb = p.overlay_bg;

    theme.button = token_style(sheet, "button", theme.button);
    theme.border = token_style(sheet, "border", theme.border);
    theme.text_input = token_style(sheet, "input-fg", theme.text_input);
    if (!theme.text_input.foreground_rgb.has_value()) {
        theme.text_input.foreground_rgb = token_rgb(sheet, "input-fg", p.text);
    }
    theme.text_input.background_rgb = token_rgb(sheet, "input-bg", p.input_chrome_fill);
    theme.text_input_focused = theme.text_input;
    theme.text_input_focused.foreground_rgb =
        token_rgb(sheet, "input-focused-fg", token_rgb(sheet, "input-fg", p.text));
    theme.text_input_focused.background_rgb = theme.text_input.background_rgb;

    tuinator::Style focused_fg = token_style(sheet, "button-focused-fg", theme.button_focused);
    tuinator::Style focused_bg = token_style(sheet, "button-focused-bg", theme.button_focused);
    theme.button_focused = focused_fg;
    if (focused_bg.background_rgb.has_value()) {
        theme.button_focused.background_rgb = focused_bg.background_rgb;
    } else if (focused_bg.foreground_rgb.has_value()) {
        theme.button_focused.background_rgb = focused_bg.foreground_rgb;
    }

    return theme;
}

std::filesystem::path executable_directory() {
#if defined(__linux__)
    std::array<char, 4096> buffer{};
    const ssize_t length = readlink("/proc/self/exe", buffer.data(), buffer.size() - 1);
    if (length > 0) {
        buffer[static_cast<std::size_t>(length)] = '\0';
        return std::filesystem::path(buffer.data()).parent_path();
    }
#endif
    return std::filesystem::current_path();
}

bool is_valid_theme_id(const std::string& theme_id) {
    if (theme_id.empty()) {
        return false;
    }
    for (unsigned char ch : theme_id) {
        if (std::isalnum(ch) == 0 && ch != '_' && ch != '-') {
            return false;
        }
    }
    return true;
}

std::string normalize_theme_id(std::string theme_id) {
    while (!theme_id.empty() && std::isspace(static_cast<unsigned char>(theme_id.front()))) {
        theme_id.erase(theme_id.begin());
    }
    while (!theme_id.empty() && std::isspace(static_cast<unsigned char>(theme_id.back()))) {
        theme_id.pop_back();
    }
    if (!is_valid_theme_id(theme_id)) {
        return "dark";
    }
    return theme_id;
}

bool try_load_theme_file(tuinator::Application& app, const std::filesystem::path& path) {
    std::error_code ec;
    if (!std::filesystem::is_regular_file(path, ec)) {
        return false;
    }
    app.load_stylesheet(path);
    return true;
}

bool try_load_bundled_theme(tuinator::Application& app, const std::string& theme_id) {
    const char* text = ui_bundled_themes::theme_text_for_id(theme_id.c_str());
    if (text == nullptr) {
        return false;
    }
    app.set_stylesheet(tuinator::Stylesheet::load_from_string(text));
    return true;
}

std::string load_theme_stylesheet(tuinator::Application& app, const std::string& theme_id) {
    const std::string id = normalize_theme_id(theme_id);
    const std::string filename = id + ".theme";

    const std::filesystem::path user_dir = ui_user_themes_directory(false);
    if (!user_dir.empty()) {
        if (try_load_theme_file(app, user_dir / filename)) {
            return id;
        }
    }

#ifdef TAIJI_UI_THEMES_DIR
    if (try_load_theme_file(app, std::filesystem::path(TAIJI_UI_THEMES_DIR) / filename)) {
        return id;
    }
#endif

    const std::filesystem::path exe_dir = executable_directory();
    if (!exe_dir.empty()) {
        if (try_load_theme_file(app, exe_dir / "themes" / filename)) {
            return id;
        }
        if (try_load_theme_file(app, exe_dir / ".." / "share" / "taiji-tui" / "themes" / filename)) {
            return id;
        }
    }

    if (const char* home = std::getenv("HOME")) {
        if (try_load_theme_file(app, std::filesystem::path(home) / ".local" / "share" / "taiji-tui" / "themes" /
                                           filename)) {
            return id;
        }
    }

    if (try_load_theme_file(app, std::filesystem::current_path() / "themes" / filename)) {
        return id;
    }

    if (try_load_bundled_theme(app, id)) {
        return id;
    }

    if (id != "dark") {
        return load_theme_stylesheet(app, "dark");
    }

    if (try_load_bundled_theme(app, "dark")) {
        return "dark";
    }

    throw std::runtime_error("Taiji UI theme not found: " + id);
}

} // namespace

std::filesystem::path ui_user_themes_directory(bool ensure_exists) {
    std::filesystem::path base;
    if (const char* xdg = std::getenv("XDG_CONFIG_HOME")) {
        if (*xdg != '\0') {
            base = std::filesystem::path(xdg);
        }
    }
    if (base.empty()) {
        if (const char* home = std::getenv("HOME")) {
            if (*home != '\0') {
                base = std::filesystem::path(home) / ".config";
            }
        }
    }
    if (base.empty()) {
        return {};
    }
    const std::filesystem::path themes = base / "taiji" / "themes";
    if (ensure_exists) {
        std::error_code ec;
        std::filesystem::create_directories(themes, ec);
    }
    return themes;
}

bool ui_is_bundled_theme(const std::string& theme_id) {
    return ui_bundled_themes::theme_text_for_id(theme_id.c_str()) != nullptr;
}

std::string ui_theme_display_name(const std::string& theme_id) {
    static const std::unordered_map<std::string, std::string> kNames = {
        {"dark", "Tokyo Night"},
        {"high_contrast", "High contrast"},
        {"gruvbox_dark", "Gruvbox Dark"},
        {"gruvbox_light", "Gruvbox Light"},
        {"catppuccin_mocha", "Catppuccin Mocha"},
        {"catppuccin_latte", "Catppuccin Latte"},
        {"nord", "Nord"},
        {"dracula", "Dracula"},
        {"rose_pine", "Rosé Pine"},
        {"solarized_dark", "Solarized Dark"},
        {"everforest_dark", "Everforest Dark"},
        {"one_dark", "One Dark"},
    };
    const auto it = kNames.find(theme_id);
    if (it != kNames.end()) {
        return it->second;
    }
    return theme_id;
}

std::vector<std::string> ui_list_theme_ids() {
    std::vector<std::string> ids;
    ids.reserve(ui_bundled_themes::kBundledThemeCount + 8);
    for (std::size_t i = 0; i < ui_bundled_themes::kBundledThemeCount; ++i) {
        ids.emplace_back(ui_bundled_themes::kBundledThemes[i].id);
    }

    const std::filesystem::path user_dir = ui_user_themes_directory(true);
    if (!user_dir.empty()) {
        std::error_code ec;
        for (const std::filesystem::directory_entry& entry :
             std::filesystem::directory_iterator(user_dir, ec)) {
            if (ec || !entry.is_regular_file()) {
                continue;
            }
            if (entry.path().extension() != ".theme") {
                continue;
            }
            const std::string id = entry.path().stem().string();
            if (!is_valid_theme_id(id)) {
                continue;
            }
            if (std::find(ids.begin(), ids.end(), id) == ids.end()) {
                ids.push_back(id);
            }
        }
    }

    if (ids.size() > ui_bundled_themes::kBundledThemeCount) {
        std::sort(ids.begin() + static_cast<std::ptrdiff_t>(ui_bundled_themes::kBundledThemeCount), ids.end());
    }
    return ids;
}

tuinator::Rgb ui_main_bg() { return palette().main_bg; }
tuinator::Rgb ui_sidebar_bg() { return palette().sidebar_bg; }
tuinator::Rgb ui_secondary_text() { return palette().secondary_text; }
tuinator::Rgb ui_section_heading() { return palette().section_heading; }
tuinator::Rgb ui_session_active_bg() { return palette().session_active_bg; }
tuinator::Rgb ui_session_active_bar() { return palette().session_active_bar; }
tuinator::Rgb ui_badge_bg() { return palette().badge_bg; }
tuinator::Rgb ui_quick_action_pill_bg() { return palette().quick_action_pill_bg; }

bool ui_message_fill_background() { return g_ui_message_fill_background; }

void ui_apply_message_fill_background(bool enabled) { g_ui_message_fill_background = enabled; }

tuinator::Rgb ui_message_panel_bg() {
    return g_ui_message_fill_background ? ui_quick_action_pill_bg() : ui_main_bg();
}
tuinator::Rgb ui_quick_action_code() { return palette().quick_action_code; }
tuinator::Rgb ui_quick_action_test() { return palette().quick_action_test; }
tuinator::Rgb ui_quick_action_git() { return palette().quick_action_git; }
tuinator::Rgb ui_logo_shadow() { return palette().logo_shadow; }
tuinator::Rgb ui_logo_top() { return palette().logo_top; }
tuinator::Rgb ui_logo_mid() { return palette().logo_mid; }
tuinator::Rgb ui_logo_bottom() { return palette().logo_bottom; }
tuinator::Rgb ui_status_ok() { return palette().status_ok; }
tuinator::Rgb ui_status_warn() { return palette().status_warn; }
tuinator::Rgb ui_status_idle() { return palette().status_idle; }
tuinator::Rgb ui_git_branch_accent() { return palette().git_branch_accent; }
tuinator::Rgb ui_input_chrome_fill() { return palette().input_chrome_fill; }
tuinator::Rgb ui_text() { return palette().text; }
tuinator::Rgb ui_accent() { return palette().accent; }
tuinator::Rgb ui_overlay_bg() { return palette().overlay_bg; }
tuinator::Rgb ui_list_active_bg() { return palette().list_active_bg; }
tuinator::Rgb ui_list_active_fg() { return palette().list_active_fg; }
tuinator::Rgb ui_border_chrome() { return palette().border_chrome; }
tuinator::Rgb ui_border_chrome_alt() { return palette().border_chrome_alt; }
tuinator::Rgb ui_danger() { return palette().danger; }
tuinator::Rgb ui_toggle_off() { return palette().toggle_off; }
tuinator::Rgb ui_caret_fg() { return palette().caret_fg; }
tuinator::Rgb ui_caret_bg() { return palette().caret_bg; }
tuinator::Rgb ui_emphasis() { return palette().emphasis; }
tuinator::Rgb ui_user_bubble() { return palette().user_bubble; }

tuinator::Style ui_accent_style() {
    tuinator::Style style;
    style.foreground_rgb = ui_accent();
    return style;
}

tuinator::Style ui_accent_on_main() { return ui_resolve_panel_text(ui_accent_style(), ui_main_bg()); }

tuinator::Style ui_accent_on_sidebar() { return ui_resolve_panel_text(ui_accent_style(), ui_sidebar_bg()); }

void ui_paint_vertical_divider(tuinator::PaintContext& ctx, tuinator::Rect local_divider, tuinator::Rgb background) {
    if (local_divider.width <= 0 || local_divider.height <= 0) {
        return;
    }
    tuinator::Style fill;
    fill.background_rgb = background;
    ctx.canvas.fill_rect(local_divider, ' ', fill);
    tuinator::Style line = fill;
    line.foreground_rgb = ui_border_chrome();
    const int x = local_divider.x + local_divider.width / 2;
    for (int y = 0; y < local_divider.height; ++y) {
        ctx.canvas.draw_char({x, local_divider.y + y}, '|', line);
    }
}

bool ui_is_palette_surface(tuinator::Rgb bg) {
    const UiPalette& p = palette();
    const auto same = [&](tuinator::Rgb c) { return bg.r == c.r && bg.g == c.g && bg.b == c.b; };
    return same(p.main_bg) || same(p.sidebar_bg) || same(p.overlay_bg) || same(p.badge_bg) ||
           same(p.list_active_bg) || same(p.session_active_bg) || same(p.quick_action_pill_bg) ||
           same(p.input_chrome_fill);
}

const tuinator::Theme& ui_app_theme() { return g_app_theme; }

tuinator::Style ui_overlay_panel_style() {
    tuinator::Style style;
    style.background_rgb = ui_overlay_bg();
    style.foreground_rgb = ui_text();
    return style;
}

tuinator::Style ui_list_selection_style(bool selected) {
    tuinator::Style style;
    style.background_rgb = selected ? ui_list_active_bg() : ui_overlay_bg();
    style.foreground_rgb = selected ? ui_list_active_fg() : ui_text();
    return style;
}

tuinator::Theme ui_apply_stylesheet(tuinator::Application& app, const std::string& theme_id) {
    (void)load_theme_stylesheet(app, theme_id);
    const tuinator::Stylesheet* sheet = app.stylesheet();
    if (sheet == nullptr) {
        throw std::runtime_error("Stylesheet missing after load for theme: " + theme_id);
    }
    g_active_sheet = sheet;
    refresh_palette(*sheet);
    tuinator::Theme theme = theme_from_stylesheet(*sheet);
    g_app_theme = theme;
    app.set_theme(theme);
    return theme;
}

bool ui_corners_rounded() { return g_ui_corner_style == UiCornerStyle::Rounded; }

bool ui_corners_flat() { return g_ui_corner_style == UiCornerStyle::Flat; }

bool ui_corners_framed() { return g_ui_corner_style != UiCornerStyle::Flat; }

tuinator::BorderStyle ui_frame_border_style() {
    return ui_corners_rounded() ? tuinator::BorderStyle::Rounded : tuinator::BorderStyle::Heavy;
}

tuinator::BorderGlyphs ui_frame_border_glyphs() {
    return tuinator::border_glyphs_for(ui_frame_border_style());
}

UiUserMessageStyle ui_user_message_style() { return g_ui_user_chat_prefs.style; }

UiChatRolePrefs ui_user_chat_prefs() { return g_ui_user_chat_prefs; }

UiChatRolePrefs ui_assistant_chat_prefs() { return g_ui_assistant_chat_prefs; }

UiMessageAlign ui_parse_message_align(const std::string& raw) {
    std::string lower;
    lower.reserve(raw.size());
    for (char ch : raw) {
        lower.push_back(static_cast<char>(std::tolower(static_cast<unsigned char>(ch))));
    }
    if (lower == "left" || lower == "l") {
        return UiMessageAlign::Left;
    }
    if (lower == "right" || lower == "r") {
        return UiMessageAlign::Right;
    }
    if (lower == "center" || lower == "centre" || lower == "middle" || lower == "c") {
        return UiMessageAlign::Center;
    }
    return UiMessageAlign::Auto;
}

UiUserMessageStyle ui_parse_message_style(const std::string& pref) {
    std::string lower;
    lower.reserve(pref.size());
    for (char ch : pref) {
        lower.push_back(static_cast<char>(std::tolower(static_cast<unsigned char>(ch))));
    }
    if (lower == "plain" || lower == "inline" || lower == "text") {
        return UiUserMessageStyle::Plain;
    }
    if (lower == "messenger" || lower == "bubble" || lower == "chat") {
        return UiUserMessageStyle::Messenger;
    }
    if (lower == "header" || lower == "header_strip" || lower == "title") {
        return UiUserMessageStyle::Header;
    }
    if (lower == "terminal" || lower == "prompt" || lower == "cli") {
        return UiUserMessageStyle::Terminal;
    }
    if (lower == "log") {
        return UiUserMessageStyle::Log;
    }
    if (lower == "card" || lower == "pill" || lower == "chip") {
        return UiUserMessageStyle::Card;
    }
    if (lower == "section" || lower == "divider" || lower == "rule") {
        return UiUserMessageStyle::Section;
    }
    if (lower == "cornered" || lower == "corners" || lower == "corner_box") {
        return UiUserMessageStyle::Cornered;
    }
    if (lower == "block" || lower == "framed" || lower == "card_block") {
        return UiUserMessageStyle::Block;
    }
    return UiUserMessageStyle::Messenger;
}

int ui_clamp_message_width_pct(int pct) {
    if (pct <= 0) {
        return 80;
    }
    if (pct < 40) {
        return 40;
    }
    if (pct > 100) {
        return 100;
    }
    return pct;
}

UiMessageAlign ui_chat_resolved_align(const UiChatRolePrefs& prefs) {
    if (prefs.align != UiMessageAlign::Auto) {
        return prefs.align;
    }
    switch (prefs.style) {
        case UiUserMessageStyle::Section:
            return UiMessageAlign::Center;
        case UiUserMessageStyle::Messenger:
        case UiUserMessageStyle::Card:
        case UiUserMessageStyle::Cornered:
            return prefs.for_assistant ? UiMessageAlign::Left : UiMessageAlign::Right;
        default:
            return UiMessageAlign::Left;
    }
}

int ui_chat_frame_max_cols(int stream_width_cols, const UiChatRolePrefs& prefs) {
    return chat_bubble_max_cols(stream_width_cols, prefs.max_width_pct);
}

int ui_chat_frame_origin(int box_x, int box_w, int content_w, const UiChatRolePrefs& prefs) {
    if (content_w <= 0 || box_w <= 0) {
        return box_x;
    }
    content_w = std::min(content_w, box_w);
    switch (ui_chat_resolved_align(prefs)) {
        case UiMessageAlign::Right:
            return box_x + std::max(0, box_w - content_w);
        case UiMessageAlign::Center:
            return box_x + std::max(0, (box_w - content_w) / 2);
        case UiMessageAlign::Left:
        case UiMessageAlign::Auto:
        default:
            return box_x;
    }
}

void ui_apply_user_message_style_pref(const std::string& pref) {
    g_ui_user_chat_prefs.style = ui_parse_message_style(pref);
}

void ui_apply_chat_message_prefs(const std::string& user_style, const std::string& user_align, int user_max_width_pct,
                                 const std::string& assistant_style, const std::string& assistant_align,
                                 int assistant_max_width_pct) {
    g_ui_user_chat_prefs.style = ui_parse_message_style(user_style);
    g_ui_user_chat_prefs.align = ui_parse_message_align(user_align);
    g_ui_user_chat_prefs.max_width_pct = ui_clamp_message_width_pct(user_max_width_pct);
    g_ui_user_chat_prefs.for_assistant = false;

    g_ui_assistant_chat_prefs.style = ui_parse_message_style(assistant_style.empty() ? user_style : assistant_style);
    g_ui_assistant_chat_prefs.align = ui_parse_message_align(assistant_align);
    g_ui_assistant_chat_prefs.max_width_pct = ui_clamp_message_width_pct(assistant_max_width_pct);
    g_ui_assistant_chat_prefs.for_assistant = true;
}

void ui_apply_corners_pref(const std::string& pref) {
    std::string normalized;
    normalized.reserve(pref.size());
    for (unsigned char ch : pref) {
        normalized.push_back(static_cast<char>(std::tolower(ch)));
    }
    if (normalized == "rounded") {
        g_ui_corner_style = UiCornerStyle::Rounded;
    } else if (normalized == "flat" || normalized == "filled" || normalized == "minimal") {
        g_ui_corner_style = UiCornerStyle::Flat;
    } else {
        g_ui_corner_style = UiCornerStyle::Square;
    }
}

UiUserBubbleInsets ui_user_bubble_insets() {
    if (ui_corners_flat()) {
        return {1, 2, 1};
    }
    if (ui_corners_rounded()) {
        return {2, 3, 1};
    }
    return {2, 2, 1};
}

void ui_paint_user_bubble_panel(tuinator::PaintContext& ctx, tuinator::Rect rect, tuinator::Rgb fill,
                              UiUserBubblePanelOptions /*options*/) {
    if (rect.width <= 0 || rect.height <= 0) {
        return;
    }
    const bool solid = ui_message_fill_background();
    const tuinator::Rgb panel_bg = solid ? fill : ui_main_bg();
    tuinator::Style fill_style;
    fill_style.background_rgb = panel_bg;
    if (ui_corners_flat() || rect.width < 3 || rect.height < 3) {
        if (solid) {
            ctx.canvas.fill_rect(rect, ' ', fill_style);
        }
        return;
    }

    const tuinator::BorderGlyphs glyphs = ui_frame_border_glyphs();

    tuinator::Style border_style;
    border_style.background_rgb = panel_bg;
    border_style.foreground_rgb = ui_user_bubble();

    if (solid) {
        ctx.canvas.fill_rect(rect, ' ', fill_style);
    }
    ctx.canvas.draw_box(rect, border_style, glyphs);
}

void ui_paint_chat_bubble_panel(tuinator::PaintContext& ctx, tuinator::Rect rect, tuinator::Rgb fill,
                                tuinator::Rgb border) {
    if (rect.width <= 0 || rect.height <= 0) {
        return;
    }
    const bool solid = ui_message_fill_background();
    const tuinator::Rgb panel_bg = solid ? fill : ui_main_bg();
    tuinator::Style fill_style;
    fill_style.background_rgb = panel_bg;
    if (ui_corners_flat() || rect.width < 3 || rect.height < 3) {
        if (solid) {
            ctx.canvas.fill_rect(rect, ' ', fill_style);
        }
        return;
    }

    const tuinator::BorderGlyphs glyphs = ui_frame_border_glyphs();

    tuinator::Style border_style;
    border_style.background_rgb = panel_bg;
    border_style.foreground_rgb = border;

    if (solid) {
        ctx.canvas.fill_rect(rect, ' ', fill_style);
    }
    ctx.canvas.draw_box(rect, border_style, glyphs);
}

tuinator::Style ui_thinking_style() {
    tuinator::Style fallback;
    fallback.foreground_rgb = tuinator::rgb(255, 140, 64);
    fallback.bold = true;
    if (g_active_sheet != nullptr) {
        return g_active_sheet->resolve_theme_token("thinking-fg", fallback);
    }
    return fallback;
}

tuinator::Style ui_composer_input_style() {
    tuinator::Style style;
    style.foreground_rgb = ui_text();
    style.background_rgb = ui_input_chrome_fill();
    if (g_active_sheet != nullptr) {
        style = g_active_sheet->resolve_theme_token("input-fg", style);
        style.background_rgb = token_rgb(*g_active_sheet, "input-bg", ui_input_chrome_fill());
    }
    return style;
}

tuinator::ScrollbarOptions ui_chat_scrollbar_options() {
    const tuinator::Rgb track_default{58, 52, 74};
    const tuinator::Rgb thumb_fg_default{188, 168, 245};
    const tuinator::Rgb thumb_bg_default{78, 68, 104};
    const tuinator::Rgb panel_bg = ui_main_bg();

    tuinator::Style track_style;
    tuinator::Style thumb_style;
    if (g_active_sheet != nullptr) {
        track_style.foreground_rgb = token_rgb(*g_active_sheet, "scrollbar-track", track_default);
        thumb_style.foreground_rgb = token_rgb(*g_active_sheet, "scrollbar-thumb-fg", thumb_fg_default);
        thumb_style.background_rgb = token_rgb(*g_active_sheet, "scrollbar-thumb-bg", thumb_bg_default);
    } else {
        track_style.foreground_rgb = track_default;
        thumb_style.foreground_rgb = thumb_fg_default;
        thumb_style.background_rgb = thumb_bg_default;
    }
    track_style.background_rgb = panel_bg;
    track_style.dim = true;
    if (!thumb_style.background_rgb.has_value()) {
        thumb_style.background_rgb = panel_bg;
    }

    tuinator::ScrollbarGlyphPatch glyphs;
    glyphs.vertical_track = "│";
    glyphs.thumb = "┃";

    const auto style = tuinator::ScrollbarStyle::thin()
                           .patch_glyphs(glyphs)
                           .with_track_style(track_style)
                           .with_thumb_style(thumb_style);

    return tuinator::ScrollbarOptions::classic()
        .with_horizontal(false)
        .with_behavior(tuinator::ScrollbarBehavior::minimal())
        .with_style(style);
}

void ui_paint_vertical_scrollbar_track_column(tuinator::PaintContext& ctx, int x, int y, int height) {
    if (height <= 0) {
        return;
    }
    const tuinator::ScrollbarOptions options = ui_chat_scrollbar_options();
    const tuinator::ScrollbarGlyphs glyphs = options.style.glyphs();
    tuinator::Style track = options.style.palette().track;
    if (!track.background_rgb.has_value()) {
        track.background_rgb = ui_main_bg();
    }
    for (int row = 0; row < height; ++row) {
        const std::string& glyph = glyphs.vertical_track;
        if (glyph.size() == 1) {
            ctx.canvas.draw_char({x, y + row}, glyph[0], track);
        } else if (!glyph.empty()) {
            ctx.canvas.draw_text({x, y + row}, glyph, track);
        }
    }
}

tuinator::Style ui_resolve_chat_message_text(tuinator::Style style) {
    return ui_resolve_panel_text(style, ui_message_panel_bg());
}

tuinator::MarkdownStyleSet ui_chat_markdown_styles(const tuinator::Theme& theme) {
    tuinator::MarkdownStyleSet styles = tuinator::markdown_styles_from_theme(theme);
    styles.text = ui_resolve_chat_message_text(styles.text);
    styles.heading = ui_resolve_chat_message_text(styles.heading);
    styles.heading_sub = ui_resolve_chat_message_text(styles.heading_sub);
    styles.quote_text = ui_resolve_chat_message_text(styles.quote_text);
    styles.link = ui_resolve_chat_message_text(styles.link);
    styles.list_marker = ui_resolve_chat_message_text(styles.list_marker);
    styles.thematic_break = ui_resolve_chat_message_text(styles.thematic_break);
    styles.quote_bar = ui_resolve_chat_message_text(styles.quote_bar);

    styles.code = ui_resolve_chat_message_text(styles.code);
    styles.code.background_rgb = ui_badge_bg();
    styles.code.foreground_rgb = ui_quick_action_code();

    const tuinator::Rgb block_bg = ui_message_panel_bg();
    styles.code_block = ui_resolve_chat_message_text(styles.code_block);
    styles.code_block.background_rgb = block_bg;
    styles.code_block.foreground_rgb = ui_text();
    styles.code_block_fill.background_rgb = block_bg;
    return styles;
}

tuinator::Style ui_sidebar_panel_style() {
    tuinator::Style style;
    style.background_rgb = ui_sidebar_bg();
    return style;
}

tuinator::Style ui_main_panel_style() {
    tuinator::Style style;
    style.background_rgb = ui_main_bg();
    return style;
}

void ui_fill_widget_bounds(tuinator::PaintContext& ctx, const tuinator::Widget& widget, tuinator::Rgb background) {
    const tuinator::Rect& bounds = widget.bounds();
    if (bounds.width <= 0 || bounds.height <= 0) {
        return;
    }
    tuinator::Style fill;
    fill.background_rgb = background;
    ctx.canvas.fill_rect({0, 0, bounds.width, bounds.height}, ' ', fill);
}

void ui_fill_widget_bounds(tuinator::PaintContext& ctx, const tuinator::Widget& widget, const tuinator::Style& panel) {
    if (panel.background_rgb.has_value()) {
        ui_fill_widget_bounds(ctx, widget, *panel.background_rgb);
    }
}

tuinator::Style ui_secondary_style(const tuinator::Theme& /*theme*/) {
    tuinator::Style style;
    style.foreground_rgb = ui_secondary_text();
    style.dim = false;
    return style;
}

tuinator::Style ui_resolve_panel_text(tuinator::Style style, tuinator::Rgb panel_bg) {
    tuinator::Style out = style;
    out.background_rgb = panel_bg;
    out.background = tuinator::Color::Default;
    return out;
}

tuinator::Style ui_resolve_sidebar_text(const tuinator::PaintContext& /*ctx*/, const tuinator::Widget& /*widget*/,
                                        tuinator::Style style) {
    tuinator::Rgb panel = ui_sidebar_bg();
    if (style.background_rgb.has_value() && ui_is_palette_surface(*style.background_rgb)) {
        panel = *style.background_rgb;
    }
    return ui_resolve_panel_text(style, panel);
}

tuinator::Style ui_resolve_main_text(tuinator::Style style) {
    return ui_resolve_panel_text(style, ui_main_bg());
}

tuinator::Style ui_resolve_footer_text(tuinator::Style style) {
    return ui_resolve_panel_text(style, ui_sidebar_bg());
}

std::string shell_quote_single(const std::string& text) {
    std::string out = "'";
    for (char ch : text) {
        if (ch == '\'') {
            out += "'\\''";
        } else {
            out.push_back(ch);
        }
    }
    out.push_back('\'');
    return out;
}

std::string run_command_capture(const std::string& command) {
    std::array<char, 256> buffer{};
    std::string output;
    FILE* raw_pipe = popen(command.c_str(), "r");
    if (raw_pipe == nullptr) {
        return {};
    }
    std::unique_ptr<FILE, int (*)(FILE*)> pipe(raw_pipe, pclose);
    if (!pipe) {
        return {};
    }
    while (fgets(buffer.data(), static_cast<int>(buffer.size()), pipe.get()) != nullptr) {
        output += buffer.data();
    }
    while (!output.empty() && (output.back() == '\n' || output.back() == '\r')) {
        output.pop_back();
    }
    return output;
}

GitWorkspaceStatus read_git_workspace_status(const std::string& repo_path) {
    GitWorkspaceStatus out;
    if (repo_path.empty()) {
        return out;
    }
    const std::string quoted = shell_quote_single(repo_path);
    out.branch = run_command_capture("git -C " + quoted + " rev-parse --abbrev-ref HEAD 2>/dev/null");
    if (out.branch.empty() || out.branch == "HEAD") {
        out.branch.clear();
        return out;
    }

    const std::string status = run_command_capture("git -C " + quoted + " status --porcelain 2>/dev/null");
    if (!status.empty()) {
        for (char ch : status) {
            if (ch == '\n') {
                ++out.uncommitted;
            }
        }
        if (status.back() != '\n') {
            ++out.uncommitted;
        }
    }
    out.valid = true;
    return out;
}
