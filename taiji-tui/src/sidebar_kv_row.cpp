#include "sidebar_kv_row.hpp"

#include "ui_layout.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

SidebarKvRow::SidebarKvRow(tuinator::Theme theme, std::string title)
    : theme_(std::move(theme)), title_(std::move(title)) {}

void SidebarKvRow::set_title(std::string title) {
    if (title_ == title) {
        return;
    }
    title_ = std::move(title);
    mark_dirty();
    mark_layout_dirty();
}

void SidebarKvRow::set_empty_value(std::string text) {
    if (empty_value_ == text) {
        return;
    }
    empty_value_ = std::move(text);
    mark_dirty();
    mark_layout_dirty();
}

void SidebarKvRow::set_detail(std::string detail) {
    if (detail_ == detail) {
        return;
    }
    detail_ = std::move(detail);
    mark_dirty();
    mark_layout_dirty();
}

void SidebarKvRow::set_collapse_when_empty(bool collapse) {
    if (collapse_when_empty_ == collapse) {
        return;
    }
    collapse_when_empty_ = collapse;
    mark_dirty();
    mark_layout_dirty();
}

bool SidebarKvRow::is_collapsed() const {
    return collapse_when_empty_ && detail_.empty();
}

tuinator::Size SidebarKvRow::preferred_size() const {
    if (is_collapsed()) {
        return {12, 1};
    }
    int detail_lines = 0;
    if (detail_.empty()) {
        detail_lines = 1;
    } else {
        detail_lines = 1;
        for (char ch : detail_) {
            if (ch == '\n') {
                ++detail_lines;
            }
        }
    }
    return {12, 1 + detail_lines};
}

void SidebarKvRow::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void SidebarKvRow::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    paint_bounds_background(ctx, ui_sidebar_panel_style());

    const int pad = kUiPadX;
    tuinator::Style title_style = theme_.heading;
    title_style.bold = true;
    title_style = ui_resolve_sidebar_text(ctx, *this, title_style);
    tuinator::Style muted = ui_resolve_sidebar_text(ctx, *this, ui_secondary_style(theme_));
    tuinator::Style label = ui_resolve_sidebar_text(ctx, *this, theme_.label);

    if (is_collapsed()) {
        ctx.canvas.draw_text({pad, 0}, title_, title_style);
        const std::string value = empty_value_.empty() ? "—" : empty_value_;
        tuinator::Style dot_style = muted;
        const bool inactive = value == "none" || value == "inactive" || value == "—";
        dot_style.foreground_rgb = inactive ? kUiStatusIdle : kUiStatusOk;
        dot_style = ui_resolve_sidebar_text(ctx, *this, dot_style);
        const std::string dot = inactive ? "○" : "●";
        const int dot_w = tuinator::text_display_width(dot);
        const int value_w = dot_w + 1 + tuinator::text_display_width(value);
        const int x = std::max(pad, bounds_.width - pad - value_w);
        ctx.canvas.draw_text({x, 0}, dot, dot_style);
        ctx.canvas.draw_text({x + dot_w + 1, 0}, value, muted);
        return;
    }

    ctx.canvas.draw_text({pad, 0}, title_, title_style);
    int y = 1;
    std::size_t start = 0;
    while (start <= detail_.size() && y < bounds_.height) {
        std::size_t end = detail_.find('\n', start);
        if (end == std::string::npos) {
            end = detail_.size();
        }
        const std::string line = detail_.substr(start, end - start);
        const int max_cols = std::max(0, bounds_.width - pad * 2);
        const std::size_t bytes = tuinator::text_byte_length_for_width(line, max_cols);
        if (bytes > 0) {
            ctx.canvas.draw_text({pad, y}, line.substr(0, bytes), label);
        }
        if (end >= detail_.size()) {
            break;
        }
        ++y;
        start = end + 1;
    }
}
