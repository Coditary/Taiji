#include "turn_metrics_note.hpp"

#include "ui_chat_column.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

#include <algorithm>
#include <sstream>

namespace {

constexpr int kActionGapCols = 2;
constexpr int kVersionGapCols = 3;
constexpr int kNavGapCols = 1;
// Font Awesome PUA: redo (U+F01E), copy (U+F0C5), code-branch (U+F126)
constexpr char kRedoGlyph[] = "\xEF\x80\x9E";
constexpr char kCopyGlyph[] = "\xEF\x83\x85";
constexpr char kBranchGlyph[] = "\xEF\x84\xA6";
constexpr char kPrevGlyph[] = "<";
constexpr char kNextGlyph[] = ">";

int glyph_width(const char* glyph) {
    return tuinator::text_display_width(glyph);
}

std::vector<std::string> split_lines(const std::string& text) {
    std::vector<std::string> lines;
    std::string current;
    for (char ch : text) {
        if (ch == '\n') {
            lines.push_back(current);
            current.clear();
        } else {
            current.push_back(ch);
        }
    }
    if (!current.empty() || text.empty()) {
        lines.push_back(current);
    }
    return lines;
}

} // namespace

TurnMetricsNote::TurnMetricsNote(tuinator::Theme theme, std::function<void(Action)> on_action)
    : theme_(theme), style_(theme.muted), on_action_(std::move(on_action)) {}

void TurnMetricsNote::set_content(std::string metrics_text, bool actions_visible, std::size_t version_index,
                                  std::size_t version_count) {
    const bool changed = metrics_text_ != metrics_text || actions_enabled_ != actions_visible ||
                         version_index_ != version_index || version_count_ != version_count;
    metrics_text_ = std::move(metrics_text);
    actions_enabled_ = actions_visible;
    version_index_ = version_index;
    version_count_ = version_count;
    if (changed) {
        mark_layout_dirty();
    }
    mark_dirty();
}

void TurnMetricsNote::clear() {
    if (!actions_enabled_ && metrics_text_.empty()) {
        return;
    }
    metrics_text_.clear();
    actions_enabled_ = false;
    actions_hover_ = false;
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

bool TurnMetricsNote::set_actions_hover(bool hovered) {
    if (!actions_enabled_) {
        hovered = false;
    }
    if (actions_hover_ == hovered) {
        return false;
    }
    actions_hover_ = hovered;
    mark_dirty();
    return true;
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

int TurnMetricsNote::reserved_right_width(bool actions_visible, std::size_t version_count) {
    int width = 0;
    if (actions_visible) {
        width += glyph_width(kRedoGlyph) + kActionGapCols + glyph_width(kCopyGlyph) + kActionGapCols +
                 glyph_width(kBranchGlyph);
    }
    if (version_count > 1) {
        if (width > 0) {
            width += kVersionGapCols;
        }
        width += glyph_width(kPrevGlyph) + kNavGapCols + 5 + kNavGapCols + glyph_width(kNextGlyph);
    }
    if (width > 0) {
        width += kActionGapCols;
    }
    return width;
}

std::vector<std::string> TurnMetricsNote::metric_lines() const {
    return split_lines(metrics_text_);
}

std::string TurnMetricsNote::version_label() const {
    if (version_count_ <= 1) {
        return {};
    }
    std::ostringstream out;
    out << (version_index_ + 1) << "/" << version_count_;
    return out.str();
}

bool TurnMetricsNote::show_primary_actions() const {
    return actions_enabled_ && actions_hover_;
}

int TurnMetricsNote::primary_actions_width() const {
    if (!actions_enabled_) {
        return 0;
    }
    return glyph_width(kRedoGlyph) + kActionGapCols + glyph_width(kCopyGlyph) + kActionGapCols +
           glyph_width(kBranchGlyph);
}

int TurnMetricsNote::actions_cluster_width() const {
    int width = 0;
    if (show_primary_actions()) {
        width += primary_actions_width();
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

ChatStreamLayout TurnMetricsNote::chat_stream_layout() const {
    const int width = bounds_.width > 0 ? bounds_.width : (wrap_width_hint_ > 0 ? wrap_width_hint_ : 72);
    return chat_stream_layout_in_widget(width);
}

int TurnMetricsNote::content_right_x() const {
    const ChatStreamLayout column = chat_stream_layout();
    if (bounds_.width > 0) {
        return std::min(bounds_.width, column.offset_x + column.content_width);
    }
    return column.offset_x + column.content_width;
}

int TurnMetricsNote::actions_cluster_start_x() const {
    const int cluster_width = actions_cluster_width();
    if (cluster_width <= 0) {
        return content_right_x();
    }
    return std::max(0, content_right_x() - right_inset_ - cluster_width);
}

int TurnMetricsNote::redo_x() const {
    return actions_cluster_start_x();
}

int TurnMetricsNote::copy_x() const {
    if (!show_primary_actions()) {
        return actions_cluster_start_x();
    }
    return redo_x() + glyph_width(kRedoGlyph) + kActionGapCols;
}

int TurnMetricsNote::branch_x() const {
    if (!show_primary_actions()) {
        return actions_cluster_start_x();
    }
    return copy_x() + glyph_width(kCopyGlyph) + kActionGapCols;
}

int TurnMetricsNote::next_x() const {
    return std::max(0, content_right_x() - right_inset_ - glyph_width(kNextGlyph));
}

int TurnMetricsNote::version_label_x() const {
    const std::string label = version_label();
    return std::max(0, next_x() - kNavGapCols - tuinator::text_display_width(label));
}

int TurnMetricsNote::prev_x() const {
    return std::max(0, version_label_x() - kNavGapCols - glyph_width(kPrevGlyph));
}

int TurnMetricsNote::metrics_x() const {
    return chat_stream_layout().offset_x;
}

int TurnMetricsNote::metrics_max_width() const {
    const int cluster_start =
        actions_cluster_width() > 0 ? actions_cluster_start_x() : content_right_x() - right_inset_;
    return std::max(0, cluster_start - kActionGapCols - metrics_x());
}

tuinator::Size TurnMetricsNote::preferred_size() const {
    const auto lines = metric_lines();
    if (!actions_enabled_ && lines.empty()) {
        return {0, 0};
    }
    const int line_count = std::max(1, static_cast<int>(lines.empty() ? 1 : lines.size()));
    const int width =
        chat_stream_row_preferred_width(bounds_.width > 0 ? bounds_.width : wrap_width_hint_);
    return {std::max(16, width), line_count};
}

void TurnMetricsNote::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void TurnMetricsNote::paint(tuinator::PaintContext& ctx) const {
    const auto lines = metric_lines();
    if ((!actions_enabled_ && lines.empty()) || bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    ui_fill_widget_bounds(ctx, *this, ui_main_bg());

    tuinator::Style body_style = ui_resolve_main_text(ui_secondary_style(theme_));

    const int max_columns = metrics_max_width();
    for (std::size_t i = 0; i < lines.size() && static_cast<int>(i) < bounds_.height; ++i) {
        if (lines[i].empty()) {
            continue;
        }
        const std::size_t bytes = tuinator::text_byte_length_for_width(lines[i], max_columns);
        if (bytes > 0) {
            ctx.canvas.draw_text({metrics_x(), static_cast<int>(i)}, lines[i].substr(0, bytes), body_style);
        }
    }

    if (show_primary_actions()) {
        tuinator::Style action_style = ui_resolve_main_text(ui_thinking_style());
        ctx.canvas.draw_text({redo_x(), 0}, kRedoGlyph, action_style);
        ctx.canvas.draw_text({copy_x(), 0}, kCopyGlyph, action_style);
        ctx.canvas.draw_text({branch_x(), 0}, kBranchGlyph, action_style);
    }

    if (version_count_ <= 1) {
        return;
    }

    tuinator::Style nav_style = ui_resolve_main_text(ui_thinking_style());

    tuinator::Style disabled_style = ui_resolve_main_text(ui_secondary_style(theme_));

    const bool has_prev = version_index_ > 0;
    const bool has_next = version_index_ + 1 < version_count_;

    ctx.canvas.draw_text({prev_x(), 0}, kPrevGlyph, has_prev ? nav_style : disabled_style);

    const std::string label = version_label();
    if (!label.empty()) {
        tuinator::Style label_style = ui_resolve_main_text(ui_secondary_style(theme_));
        ctx.canvas.draw_text({version_label_x(), 0}, label, label_style);
    }

    ctx.canvas.draw_text({next_x(), 0}, kNextGlyph, has_next ? nav_style : disabled_style);
}

bool TurnMetricsNote::handle_event(const tuinator::Event& event) {
    if (!actions_enabled_) {
        return false;
    }

    const auto* mouse = std::get_if<tuinator::MouseEvent>(&event);
    if (mouse == nullptr) {
        return false;
    }

    if (mouse->action == tuinator::MouseAction::Move) {
        const bool inside = bounds_.contains(mouse->position);
        if (inside != actions_hover_) {
            set_actions_hover(inside);
        }
        return inside;
    }

    if (mouse->action != tuinator::MouseAction::Press) {
        return false;
    }

    if (!bounds_.contains(mouse->position)) {
        return false;
    }

    set_actions_hover(true);

    const tuinator::Point local{mouse->position.x - bounds_.x, mouse->position.y - bounds_.y};
    if (local.y != 0 || local.x < 0 || local.x >= bounds_.width) {
        return false;
    }

    const int primary_w = primary_actions_width();
    const int redo = std::max(0, content_right_x() - right_inset_ - primary_w);
    const int copy = redo + glyph_width(kRedoGlyph) + kActionGapCols;
    const int branch = copy + glyph_width(kCopyGlyph) + kActionGapCols;

    if (local.x >= branch && local.x < branch + glyph_width(kBranchGlyph)) {
        if (on_action_) {
            on_action_(Action::Branch);
        }
        return true;
    }

    if (local.x >= copy && local.x < copy + glyph_width(kCopyGlyph)) {
        if (on_action_) {
            on_action_(Action::Copy);
        }
        return true;
    }

    if (local.x >= redo && local.x < redo + glyph_width(kRedoGlyph)) {
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
