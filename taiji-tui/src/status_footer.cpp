#include "status_footer.hpp"

#include <tuinator/render/text.hpp>

StatusFooter::StatusFooter(tuinator::Style path_style, tuinator::Style status_style)
    : path_style_(path_style), status_style_(status_style) {}

void StatusFooter::set_path(std::string text) {
    if (path_ == text) {
        return;
    }
    path_ = std::move(text);
    mark_dirty();
}

void StatusFooter::set_status(std::string text) {
    if (status_ == text) {
        return;
    }
    status_ = std::move(text);
    mark_dirty();
}

tuinator::Size StatusFooter::preferred_size() const {
    return {1, 1};
}

void StatusFooter::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void StatusFooter::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    paint_bounds_background(ctx, path_style_);

    const tuinator::Style path = ctx.styles().text(*this, path_style_);
    const tuinator::Style status = ctx.styles().text(*this, status_style_);

    if (!path_.empty()) {
        const int max_width = std::max(0, bounds_.width);
        const std::size_t bytes = tuinator::text_byte_length_for_width(path_, max_width);
        if (bytes > 0) {
            ctx.canvas.draw_text({0, 0}, path_.substr(0, bytes), path);
        }
    }

    if (!status_.empty()) {
        const int status_width = tuinator::text_display_width(status_);
        const int x = std::max(0, bounds_.width - status_width);
        const std::size_t bytes = tuinator::text_byte_length_for_width(status_, bounds_.width);
        if (bytes > 0) {
            ctx.canvas.draw_text({x, 0}, status_.substr(0, bytes), status);
        }
    }
}
