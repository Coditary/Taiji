#include "ui_theme.hpp"

#include <algorithm>
#include <array>
#include <cctype>
#include <cstdio>
#include <memory>
#include <string>

namespace {

enum class UiCornerStyle { Square, Rounded, Flat };

UiCornerStyle g_ui_corner_style = UiCornerStyle::Square;

} // namespace

bool ui_corners_rounded() { return g_ui_corner_style == UiCornerStyle::Rounded; }

bool ui_corners_flat() { return g_ui_corner_style == UiCornerStyle::Flat; }

bool ui_corners_framed() { return g_ui_corner_style != UiCornerStyle::Flat; }

tuinator::BorderStyle ui_frame_border_style() {
    return ui_corners_rounded() ? tuinator::BorderStyle::Rounded : tuinator::BorderStyle::Heavy;
}

tuinator::BorderGlyphs ui_frame_border_glyphs() {
    return tuinator::border_glyphs_for(ui_frame_border_style());
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

void ui_paint_user_bubble_panel(tuinator::PaintContext& ctx, tuinator::Rect rect, tuinator::Rgb fill) {
    if (rect.width <= 0 || rect.height <= 0) {
        return;
    }
    tuinator::Style fill_style;
    fill_style.background_rgb = fill;
    if (ui_corners_flat() || rect.width < 3 || rect.height < 3) {
        ctx.canvas.fill_rect(rect, ' ', fill_style);
        return;
    }

    tuinator::Style border_style = fill_style;
    border_style.foreground_rgb = tuinator::rgb(130, 120, 170);

    ctx.canvas.fill_rect({rect.x + 1, rect.y + 1, rect.width - 2, rect.height - 2}, ' ', fill_style);
    ctx.canvas.draw_box(rect, border_style, ui_frame_border_glyphs());
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

void tune_chat_theme(tuinator::Theme& theme) {
    theme.muted.foreground_rgb = kUiSecondaryText;
    theme.label.foreground_rgb = tuinator::rgb(192, 195, 215);
    theme.heading.foreground_rgb = kUiSectionHeading;
    theme.accent.foreground_rgb = tuinator::rgb(187, 154, 247);
    theme.label.background_rgb = kUiMainBackground;
    theme.heading.background_rgb = kUiMainBackground;
    theme.muted.background_rgb = kUiMainBackground;
    theme.accent.background_rgb = kUiMainBackground;
}

tuinator::MarkdownStyleSet ui_chat_markdown_styles(const tuinator::Theme& theme) {
    tuinator::MarkdownStyleSet styles = tuinator::markdown_styles_from_theme(theme);
    styles.text = ui_resolve_main_text(styles.text);
    styles.heading = ui_resolve_main_text(styles.heading);
    styles.heading_sub = ui_resolve_main_text(styles.heading_sub);
    styles.quote_text = ui_resolve_main_text(styles.quote_text);
    styles.link = ui_resolve_main_text(styles.link);
    styles.list_marker = ui_resolve_main_text(styles.list_marker);
    styles.thematic_break = ui_resolve_main_text(styles.thematic_break);
    styles.quote_bar = ui_resolve_main_text(styles.quote_bar);

    styles.code = ui_resolve_main_text(styles.code);
    styles.code.background_rgb = kUiBadgeBackground;
    styles.code.foreground_rgb = kUiQuickActionCode;

    styles.code_block = ui_resolve_main_text(styles.code_block);
    styles.code_block.background_rgb = kUiQuickActionPillBg;
    styles.code_block.foreground_rgb = tuinator::rgb(220, 225, 240);
    styles.code_block_fill.background_rgb = kUiQuickActionPillBg;
    return styles;
}

tuinator::Style ui_sidebar_panel_style() {
    tuinator::Style style;
    style.background_rgb = kUiSidebarBackground;
    return style;
}

tuinator::Style ui_main_panel_style() {
    tuinator::Style style;
    style.background_rgb = kUiMainBackground;
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

tuinator::Style ui_secondary_style(const tuinator::Theme& theme) {
    tuinator::Style style = theme.muted;
    style.foreground_rgb = kUiSecondaryText;
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
    tuinator::Rgb panel = kUiSidebarBackground;
    if (style.background_rgb.has_value()) {
        panel = *style.background_rgb;
    }
    return ui_resolve_panel_text(style, panel);
}

tuinator::Style ui_resolve_main_text(tuinator::Style style) {
    return ui_resolve_panel_text(style, kUiMainBackground);
}

tuinator::Style ui_resolve_footer_text(tuinator::Style style) {
    return ui_resolve_panel_text(style, kUiSidebarBackground);
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
