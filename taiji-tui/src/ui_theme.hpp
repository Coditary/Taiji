#pragma once

#include <tuinator/render/border_style.hpp>
#include <tuinator/render/glyphs.hpp>
#include <tuinator/render/paint_context.hpp>
#include <tuinator/render/theme.hpp>
#include <tuinator/tuinator.hpp>
#include <tuinator/widgets/display/markdown_view.hpp>
#include <tuinator/widgets/widget.hpp>

#include <string>

// Tokyo Night / Catppuccin-inspired surfaces (TrueColor).
constexpr tuinator::Rgb kUiMainBackground{26, 27, 38};       // #1a1b26 main stage
constexpr tuinator::Rgb kUiSidebarBackground{22, 22, 30};    // #16161e sidebars + footer
constexpr tuinator::Rgb kUiSecondaryText{86, 95, 137};
constexpr tuinator::Rgb kUiSectionHeading{169, 177, 214};    // #a9b1d6
constexpr tuinator::Rgb kUiSessionActiveBackground{45, 36, 59}; // #2d243b
constexpr tuinator::Rgb kUiSessionActiveBar{203, 166, 247};   // #cba6f7
constexpr tuinator::Rgb kUiBadgeBackground{41, 46, 66};       // #292e42
constexpr tuinator::Rgb kUiQuickActionPillBg{36, 39, 58};     // #24273a
constexpr tuinator::Rgb kUiQuickActionCode{137, 180, 250};   // #89b4fa
constexpr tuinator::Rgb kUiQuickActionTest{148, 226, 213};   // #94e2d5
constexpr tuinator::Rgb kUiQuickActionGit{250, 179, 135};    // #fab387
constexpr tuinator::Rgb kUiLogoShadow{69, 71, 90};            // #45475a
constexpr tuinator::Rgb kUiLogoTop{180, 190, 254};           // #b4befe
constexpr tuinator::Rgb kUiLogoMid{203, 166, 247};           // #cba6f7
constexpr tuinator::Rgb kUiLogoBottom{165, 140, 210};
constexpr tuinator::Rgb kUiStatusOk{115, 218, 202};          // #73daca
constexpr tuinator::Rgb kUiStatusWarn{224, 175, 104};        // #e0af68
constexpr tuinator::Rgb kUiStatusIdle{86, 95, 137};
constexpr tuinator::Rgb kUiGitBranchAccent{125, 207, 255};
constexpr tuinator::Rgb kUiInputChromeFill{42, 38, 58};

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
struct UiUserBubbleInsets {
    int pad_x = 1;
    int body_pad_x = 2;
    int bottom_inset = 1;
};

UiUserBubbleInsets ui_user_bubble_insets();
/// User message bubble; square uses heavy frame (┏), rounded uses ╭╮ against chat background.
void ui_paint_user_bubble_panel(tuinator::PaintContext& ctx, tuinator::Rect rect, tuinator::Rgb fill);

void tune_chat_theme(tuinator::Theme& theme);
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

GitWorkspaceStatus read_git_workspace_status(const std::string& repo_path);
std::string shell_quote_single(const std::string& text);
std::string run_command_capture(const std::string& command);
