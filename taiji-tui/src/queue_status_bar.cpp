#include "queue_status_bar.hpp"

#include "ui_chat_column.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

#include <sstream>

namespace {

constexpr int kGapCols = 1;
constexpr char kQueuePrefix[] = "» ";
constexpr char kPromoteGlyph[] = "↑";
constexpr char kDeleteGlyph[] = "🗑";

std::string truncate_to_width(const std::string& text, int max_cols) {
    if (max_cols <= 0 || text.empty()) {
        return {};
    }
    if (tuinator::text_display_width(text) <= max_cols) {
        return text;
    }
    if (max_cols <= 1) {
        return "…";
    }

    const std::size_t bytes = tuinator::text_byte_length_for_width(text, max_cols - 1);
    if (bytes == 0) {
        return "…";
    }
    return text.substr(0, bytes) + "…";
}

} // namespace

QueueStatusBar::QueueStatusBar(tuinator::Theme theme, std::function<void()> on_promote,
                               std::function<void()> on_delete)
    : theme_(theme), on_promote_(std::move(on_promote)), on_delete_(std::move(on_delete)) {}

void QueueStatusBar::set_preview(std::string text, std::size_t extra_waiting) {
    preview_ = std::move(text);
    extra_waiting_ = extra_waiting;
    const bool next_visible = !preview_.empty();
    if (visible_ != next_visible) {
        visible_ = next_visible;
        mark_layout_dirty();
    }
    mark_dirty();
}

void QueueStatusBar::set_wrap_width(int width) {
    width = std::max(0, width);
    if (wrap_width_hint_ == width) {
        return;
    }
    wrap_width_hint_ = width;
    mark_layout_dirty();
    mark_dirty();
}

int QueueStatusBar::action_area_width() const {
    return tuinator::text_display_width(kPromoteGlyph) + kGapCols + tuinator::text_display_width(kDeleteGlyph);
}

int QueueStatusBar::delete_x() const {
    return std::max(0, bounds_.width - tuinator::text_display_width(kDeleteGlyph));
}

int QueueStatusBar::promote_x() const {
    return std::max(0, delete_x() - kGapCols - tuinator::text_display_width(kPromoteGlyph));
}

std::string QueueStatusBar::display_text() const {
    if (!visible_ || preview_.empty()) {
        return {};
    }

    const int width = bounds_.width > 0 ? bounds_.width : wrap_width_hint_;
    const int prefix_cols = tuinator::text_display_width(kQueuePrefix);
    const int text_cols = std::max(8, width - action_area_width() - prefix_cols);

    std::ostringstream out;
    out << kQueuePrefix << truncate_to_width(preview_, text_cols);
    if (extra_waiting_ > 0) {
        out << " (+" << extra_waiting_ << ")";
    }
    return out.str();
}

tuinator::Size QueueStatusBar::preferred_size() const {
    if (!visible_ || preview_.empty()) {
        return {0, 0};
    }
    const int width = wrap_width_hint_ > 0 ? wrap_width_hint_ : bounds_.width;
    return {std::max(16, width), 1};
}

void QueueStatusBar::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void QueueStatusBar::paint(tuinator::PaintContext& ctx) const {
    if (!visible_ || bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    paint_bounds_background(ctx, ui_main_panel_style());

    const ChatStreamLayout column = chat_stream_layout_in_widget(bounds_.width);
    const int x0 = column.offset_x;
    const int inner_w = column.content_width;

    const std::string text = display_text();
    if (!text.empty()) {
        tuinator::Style line = ui_resolve_main_text(theme_.accent);
        ctx.canvas.draw_text({x0, 0}, text, line);
    }

    tuinator::Style promote_style = ui_resolve_main_text(theme_.accent);
    promote_style.bold = true;
    const int delete_col = x0 + std::max(0, inner_w - tuinator::text_display_width(kDeleteGlyph));
    const int promote_col =
        delete_col - kGapCols - tuinator::text_display_width(kPromoteGlyph);
    ctx.canvas.draw_text({promote_col, 0}, kPromoteGlyph, promote_style);

    tuinator::Style delete_style = ui_resolve_main_text(theme_.danger);
    delete_style.bold = true;
    ctx.canvas.draw_text({delete_col, 0}, kDeleteGlyph, delete_style);
}

bool QueueStatusBar::handle_event(const tuinator::Event& event) {
    if (!visible_) {
        return false;
    }

    const auto* mouse = std::get_if<tuinator::MouseEvent>(&event);
    if (mouse == nullptr || mouse->action != tuinator::MouseAction::Press) {
        return false;
    }

    const tuinator::Point local{mouse->position.x - bounds_.x, mouse->position.y - bounds_.y};
    if (local.y != 0 || local.x < 0 || local.x >= bounds_.width) {
        return false;
    }

    if (local.x >= delete_x()) {
        if (on_delete_) {
            on_delete_();
        }
        return true;
    }

    if (local.x >= promote_x()) {
        if (on_promote_) {
            on_promote_();
        }
        return true;
    }

    return false;
}
