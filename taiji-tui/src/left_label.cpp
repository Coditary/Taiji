#include "left_label.hpp"

#include "ui_layout.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

LeftLabel::LeftLabel(std::string text, tuinator::Style style) : text_(std::move(text)), style_(style) {}

void LeftLabel::set_text(std::string text) {
    if (text_ == text) {
        return;
    }
    text_ = std::move(text);
    mark_dirty();
    mark_layout_dirty();
}

void LeftLabel::set_style(tuinator::Style style) {
    style_ = style;
    mark_dirty();
}

tuinator::Size LeftLabel::preferred_size() const {
    if (text_.empty()) {
        return {0, 0};
    }

    int max_width = 0;
    int lines = 0;
    std::size_t start = 0;
    while (start <= text_.size()) {
        std::size_t end = text_.find('\n', start);
        if (end == std::string::npos) {
            end = text_.size();
        }
        max_width =
            std::max(max_width, tuinator::text_display_width(std::string_view(text_).substr(start, end - start)));
        ++lines;
        if (end >= text_.size()) {
            break;
        }
        start = end + 1;
    }

    return {max_width, std::max(1, lines)};
}

void LeftLabel::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void LeftLabel::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0 || text_.empty()) {
        return;
    }

    if (style_.background_rgb.has_value() && style_.background_rgb.value() == kUiSidebarBackground) {
        paint_bounds_background(ctx, ui_sidebar_panel_style());
    } else {
        paint_bounds_background(ctx, style_);
    }

    tuinator::Style style = style_;
    if (style_.background_rgb.has_value() && style_.background_rgb.value() == kUiSidebarBackground) {
        style = ui_resolve_sidebar_text(ctx, *this, style_);
    } else {
        style = ui_resolve_main_text(style_);
    }
    int y = 0;
    std::size_t start = 0;
    while (start <= text_.size() && y < bounds_.height) {
        std::size_t end = text_.find('\n', start);
        if (end == std::string::npos) {
            end = text_.size();
        }

        const std::string line = text_.substr(start, end - start);
        const int max_columns = std::max(0, bounds_.width - kUiPadX * 2);
        const std::size_t bytes = tuinator::text_byte_length_for_width(line, max_columns);
        if (bytes > 0) {
            ctx.canvas.draw_text({kUiPadX, y}, line.substr(0, bytes), style);
        }

        if (end >= text_.size()) {
            break;
        }
        ++y;
        start = end + 1;
    }
}
