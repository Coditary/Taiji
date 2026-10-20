#pragma once

#include <tuinator/core/application.hpp>
#include <tuinator/render/border_style.hpp>
#include <tuinator/render/glyphs.hpp>
#include <tuinator/render/paint_context.hpp>
#include <tuinator/render/theme.hpp>
#include <tuinator/tuinator.hpp>
#include <tuinator/widgets/display/markdown_view.hpp>
#include <tuinator/widgets/widget.hpp>

#include <filesystem>
#include <string>
#include <vector>

/// `~/.config/taiji/themes` (or `$XDG_CONFIG_HOME/taiji/themes`).
std::filesystem::path ui_user_themes_directory(bool ensure_exists = false);

/// Built-in ids plus `*.theme` files in the user themes directory (sorted, unique).
std::vector<std::string> ui_list_theme_ids();

bool ui_is_bundled_theme(const std::string& theme_id);
std::string ui_theme_display_name(const std::string& theme_id);

/// Load stylesheet (user override → dev tree → embedded default), apply palette + Tuinator theme.
tuinator::Theme ui_apply_stylesheet(tuinator::Application& app, const std::string& theme_id);

/// Last theme produced by `ui_apply_stylesheet` (use in paint for overlays that must track live preview).
const tuinator::Theme& ui_app_theme();

inline constexpr const char kDefaultFooterShortcuts[] =
    "Tab: focus · Shift+Enter: newline · Enter: send · Esc: stop";

// Nerd Font glyphs (terminal must use a Nerd Font).
inline constexpr const char kNfModel[] = u8"󰚩";
inline constexpr const char kNfUser[] = u8"󰀄";
inline constexpr const char kNfMicrophone[] = u8"󰍬";
inline constexpr const char kNfSettings[] = u8"󰒓";
inline constexpr const char kNfGitBranch[] = u8"󰘬";
inline constexpr const char kNfFile[] = u8"󰈔";
inline constexpr const char kNfTest[] = u8"󰙨";
inline constexpr const char kNfGit[] = u8"󰊢";
inline constexpr const char kNfAdd[] = u8"󰐕";
inline constexpr const char kNfFolder[] = u8"󰉋";

// Semantic colors (from the active stylesheet @theme block).
tuinator::Rgb ui_main_bg();
tuinator::Rgb ui_sidebar_bg();
tuinator::Rgb ui_secondary_text();
tuinator::Rgb ui_section_heading();
tuinator::Rgb ui_session_active_bg();
tuinator::Rgb ui_session_active_bar();
tuinator::Rgb ui_badge_bg();
tuinator::Rgb ui_quick_action_pill_bg();
bool ui_message_fill_background();
void ui_apply_message_fill_background(bool enabled);
/** Panel fill for framed chat bubbles (pill color or main chat bg when fill is off). */
tuinator::Rgb ui_message_panel_bg();
tuinator::Rgb ui_quick_action_code();
tuinator::Rgb ui_quick_action_test();
tuinator::Rgb ui_quick_action_git();
tuinator::Rgb ui_logo_shadow();
tuinator::Rgb ui_logo_top();
tuinator::Rgb ui_logo_mid();
tuinator::Rgb ui_logo_bottom();
tuinator::Rgb ui_status_ok();
tuinator::Rgb ui_status_warn();
tuinator::Rgb ui_status_idle();
tuinator::Rgb ui_git_branch_accent();
tuinator::Rgb ui_input_chrome_fill();
tuinator::Rgb ui_text();
tuinator::Rgb ui_accent();
tuinator::Rgb ui_overlay_bg();
tuinator::Rgb ui_list_active_bg();
tuinator::Rgb ui_list_active_fg();
tuinator::Rgb ui_border_chrome();
tuinator::Rgb ui_border_chrome_alt();
tuinator::Rgb ui_danger();
tuinator::Rgb ui_toggle_off();
tuinator::Rgb ui_caret_fg();
tuinator::Rgb ui_caret_bg();
tuinator::Rgb ui_emphasis();
tuinator::Rgb ui_user_bubble();

bool ui_is_palette_surface(tuinator::Rgb bg);

tuinator::Style ui_accent_style();
tuinator::Style ui_accent_on_main();
tuinator::Style ui_accent_on_sidebar();
void ui_paint_vertical_divider(tuinator::PaintContext& ctx, tuinator::Rect local_divider, tuinator::Rgb background);

tuinator::Style ui_overlay_panel_style();
tuinator::Style ui_list_selection_style(bool selected);

struct GitWorkspaceStatus {
    bool valid = false;
    std::string branch;
    int uncommitted = 0;
};

bool ui_corners_rounded();
bool ui_corners_flat();
bool ui_corners_framed();
tuinator::BorderStyle ui_frame_border_style();
tuinator::BorderGlyphs ui_frame_border_glyphs();
void ui_apply_corners_pref(const std::string& pref);

enum class UiMessageAlign { Auto, Left, Right, Center };

enum class UiUserMessageStyle {
    Block,
    Plain,
    Messenger,
    Header,
    Terminal,
    Log,
    Card,
    Section,
    Cornered,
};

struct UiChatRolePrefs {
    UiUserMessageStyle style = UiUserMessageStyle::Messenger;
    UiMessageAlign align = UiMessageAlign::Auto;
    int max_width_pct = 80;
    bool for_assistant = false;
};

UiUserMessageStyle ui_user_message_style();
void ui_apply_user_message_style_pref(const std::string& pref);

UiChatRolePrefs ui_user_chat_prefs();
UiChatRolePrefs ui_assistant_chat_prefs();
void ui_apply_chat_message_prefs(const std::string& user_style, const std::string& user_align, int user_max_width_pct,
                                 const std::string& assistant_style, const std::string& assistant_align,
                                 int assistant_max_width_pct);
UiMessageAlign ui_parse_message_align(const std::string& raw);
UiMessageAlign ui_chat_resolved_align(const UiChatRolePrefs& prefs);
int ui_chat_frame_max_cols(int stream_width_cols, const UiChatRolePrefs& prefs);
int ui_chat_frame_origin(int box_x, int box_w, int content_w, const UiChatRolePrefs& prefs);
UiUserMessageStyle ui_parse_message_style(const std::string& pref);
struct UiUserBubbleInsets {
    int pad_x = 1;
    int body_pad_x = 2;
    int bottom_inset = 1;
};

UiUserBubbleInsets ui_user_bubble_insets();
struct UiUserBubblePanelOptions {
    bool flush_top = false;
};

void ui_paint_user_bubble_panel(tuinator::PaintContext& ctx, tuinator::Rect rect, tuinator::Rgb fill,
                               UiUserBubblePanelOptions options = {});
void ui_paint_chat_bubble_panel(tuinator::PaintContext& ctx, tuinator::Rect rect, tuinator::Rgb fill,
                                tuinator::Rgb border);

tuinator::Style ui_thinking_style();
tuinator::Style ui_composer_input_style();
tuinator::MarkdownStyleSet ui_chat_markdown_styles(const tuinator::Theme& theme);

tuinator::Style ui_sidebar_panel_style();
tuinator::Style ui_main_panel_style();
void ui_fill_widget_bounds(tuinator::PaintContext& ctx, const tuinator::Widget& widget, tuinator::Rgb background);
void ui_fill_widget_bounds(tuinator::PaintContext& ctx, const tuinator::Widget& widget, const tuinator::Style& panel);
tuinator::Style ui_secondary_style(const tuinator::Theme& theme);

tuinator::Style ui_resolve_panel_text(tuinator::Style style, tuinator::Rgb panel_bg);
tuinator::Style ui_resolve_sidebar_text(const tuinator::PaintContext& ctx, const tuinator::Widget& widget,
                                        tuinator::Style style);
tuinator::Style ui_resolve_main_text(tuinator::Style style);
tuinator::Style ui_resolve_footer_text(tuinator::Style style);

tuinator::ScrollbarOptions ui_chat_scrollbar_options();
void ui_paint_vertical_scrollbar_track_column(tuinator::PaintContext& ctx, int x, int y, int height);

GitWorkspaceStatus read_git_workspace_status(const std::string& repo_path);
std::string shell_quote_single(const std::string& text);
std::string run_command_capture(const std::string& command);
