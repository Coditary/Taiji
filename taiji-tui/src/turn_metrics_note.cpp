#include "turn_metrics_note.hpp"

#include <tuinator/render/text.hpp>

#include <algorithm>
#include <sstream>

namespace {

constexpr int kBarIndent = 0;
constexpr int kMetricsIndent = 2;
constexpr int kActionGapCols = 2;
constexpr int kVersionGapCols = 3;
constexpr int kNavGapCols = 1;
constexpr char kBarGlyph[] = "│";
// Font Awesome PUA: redo (U+F01E), copy (U+F0C5)
constexpr char kRedoGlyph[] = "\xEF\x80\x9E";
constexpr char kCopyGlyph[] = "\xEF\x83\x85";
constexpr char kPrevGlyph[] = "<";
constexpr char kNextGlyph[] = ">";

int glyph_width(const char* glyph) {
    return tuinator::text_display_width(glyph);
}

} // namespace

TurnMetricsNote::TurnMetricsNote(tuinator::Theme theme, std::function<void(Action)> on_action)
    : theme_(theme), style_(theme.muted), on_action_(std::move(on_action)) {}

void TurnMetricsNote::set_content(std::string metrics_text, bool actions_visible, std::size_t version_index,
                                  std::size_t version_count) {
    const bool changed = metrics_text_ != metrics_text || visible_ != actions_visible ||
                         version_index_ != version_index || version_count_ != version_count;
    metrics_text_ = std::move(metrics_text);
    visible_ = actions_visible;
    version_index_ = version_index;
    version_count_ = version_count;
    if (changed) {
        mark_layout_dirty();
    }
    mark_dirty();
}

void TurnMetricsNote::clear() {
    if (!visible_ && metrics_text_.empty()) {
        return;
    }
    metrics_text_.clear();
    visible_ = false;
    version_index_ = 0;
    version_count_ = 0;
    mark_dirty();
    mark_layout_dirty();
}

void TurnMetricsNote::set_wrap_width(int width) {
    width = std::max(0, width);
    if (wrap_width_hint_ == width) {
        return;
    }
    wrap_width_hint_ = width;
    mark_layout_dirty();
    mark_dirty();
}

void TurnMetricsNote::set_right_inset(int columns) {
    columns = std::max(0, columns);
    if (right_inset_ == columns) {
        return;
    }
    right_inset_ = columns;
    mark_layout_dirty();
    mark_dirty();
}

std::string TurnMetricsNote::version_label() const {
    if (version_count_ <= 1) {
        return {};
    }
    std::ostringstream out;
    out << (version_index_ + 1) << "/" << version_count_;
    return out.str();
}

int TurnMetricsNote::actions_cluster_width() const {
    int width = 0;
    if (visible_) {
        width += glyph_width(kRedoGlyph) + kActionGapCols + glyph_width(kCopyGlyph);
    }
    if (version_count_ > 1) {
        if (width > 0) {
            width += kVersionGapCols;
        }
        const std::string label = version_label();
        width += glyph_width(kPrevGlyph) + kNavGapCols + tuinator::text_display_width(label) + kNavGapCols +
                 glyph_width(kNextGlyph);
    }
    return width;
}

int TurnMetricsNote::actions_cluster_start_x() const {
    const int cluster_width = actions_cluster_width();
    if (cluster_width <= 0) {
        return bounds_.width;
    }
    return std::max(0, bounds_.width - right_inset_ - cluster_width);
}

int TurnMetricsNote::redo_x() const {
    return actions_cluster_start_x();
}

int TurnMetricsNote::copy_x() const {
    if (!visible_) {
        return actions_cluster_start_x();
    }
    return redo_x() + glyph_width(kRedoGlyph) + kActionGapCols;
}

int TurnMetricsNote::next_x() const {
    return std::max(0, bounds_.width - right_inset_ - glyph_width(kNextGlyph));
}

int TurnMetricsNote::version_label_x() const {
    const std::string label = version_label();
    return std::max(0, next_x() - kNavGapCols - tuinator::text_display_width(label));
}

int TurnMetricsNote::prev_x() const {
    return std::max(0, version_label_x() - kNavGapCols - glyph_width(kPrevGlyph));
}

int TurnMetricsNote::metrics_x() const {
    return kBarIndent + glyph_width(kBarGlyph) + 1;
}

int TurnMetricsNote::metrics_max_width() const {
    const int cluster_start = actions_cluster_width() > 0 ? actions_cluster_start_x() : bounds_.width - right_inset_;
    return std::max(0, cluster_start - kActionGapCols - metrics_x());
}

tuinator::Size TurnMetricsNote::preferred_size() const {
    if (!visible_ && metrics_text_.empty()) {
        return {0, 0};
    }
    const int width = bounds_.width > 0 ? bounds_.width : (wrap_width_hint_ > 0 ? wrap_width_hint_ : 48);
    return {std::max(16, width), 1};
}

void TurnMetricsNote::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void TurnMetricsNote::paint(tuinator::PaintContext& ctx) const {
    if ((!visible_ && metrics_text_.empty()) || bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    paint_bounds_background(ctx, style_);

    tuinator::Style bar_style = style_;
    bar_style.bold = true;
    bar_style.foreground_rgb = tuinator::rgb(120, 110, 150);
    ctx.canvas.draw_text({kBarIndent, 0}, kBarGlyph, bar_style);

    if (!metrics_text_.empty()) {
        tuinator::Style body_style = style_;
        body_style.bold = false;
        body_style.foreground_rgb = tuinator::rgb(170, 165, 185);

        const int max_columns = metrics_max_width();
        const std::size_t bytes = tuinator::text_byte_length_for_width(metrics_text_, max_columns);
        if (bytes > 0) {
            ctx.canvas.draw_text({metrics_x(), 0}, metrics_text_.substr(0, bytes), body_style);
        }
    }

    if (visible_) {
        tuinator::Style action_style = theme_.accent;
        action_style.bold = true;
        ctx.canvas.draw_text({redo_x(), 0}, kRedoGlyph, action_style);
        ctx.canvas.draw_text({copy_x(), 0}, kCopyGlyph, action_style);
    }

    if (version_count_ <= 1) {
        return;
    }

    tuinator::Style nav_style = theme_.accent;
    nav_style.bold = true;

    tuinator::Style disabled_style = theme_.muted;
    disabled_style.bold = false;

    const bool has_prev = version_index_ > 0;
    const bool has_next = version_index_ + 1 < version_count_;

    ctx.canvas.draw_text({prev_x(), 0}, kPrevGlyph, has_prev ? nav_style : disabled_style);

    const std::string label = version_label();
    if (!label.empty()) {
        tuinator::Style label_style = theme_.muted;
        ctx.canvas.draw_text({version_label_x(), 0}, label, label_style);
    }

    ctx.canvas.draw_text({next_x(), 0}, kNextGlyph, has_next ? nav_style : disabled_style);
}

bool TurnMetricsNote::handle_event(const tuinator::Event& event) {
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

    if (local.x >= copy_x() && local.x < copy_x() + glyph_width(kCopyGlyph)) {
        if (on_action_) {
            on_action_(Action::Copy);
        }
        return true;
    }

    if (local.x >= redo_x() && local.x < redo_x() + glyph_width(kRedoGlyph)) {
        if (on_action_) {
            on_action_(Action::Redo);
        }
        return true;
    }

    if (version_count_ > 1) {
        if (local.x >= next_x() && version_index_ + 1 < version_count_) {
            if (on_action_) {
                on_action_(Action::NextVersion);
            }
            return true;
        }

        if (local.x >= prev_x() && local.x < prev_x() + glyph_width(kPrevGlyph) && version_index_ > 0) {
            if (on_action_) {
                on_action_(Action::PrevVersion);
            }
            return true;
        }
    }

    return false;
}
