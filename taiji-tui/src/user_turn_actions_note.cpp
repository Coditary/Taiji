#include "user_turn_actions_note.hpp"

#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

#include <algorithm>

namespace {

constexpr int kActionGapCols = 2;
constexpr const char* kEditGlyph = kNfPencil;
// Font Awesome PUA: copy (U+F0C5) — matches assistant turn actions
constexpr char kCopyGlyph[] = "\xEF\x83\x85";

int glyph_width(const char* glyph) {
    return tuinator::text_display_width(glyph);
}

} // namespace

UserTurnActionsNote::UserTurnActionsNote(tuinator::Theme theme, std::function<void(Action)> on_action)
    : theme_(theme), on_action_(std::move(on_action)) {}

void UserTurnActionsNote::set_actions_visible(bool visible) {
    if (actions_enabled_ == visible) {
        return;
    }
    actions_enabled_ = visible;
    if (!visible) {
        actions_hover_ = false;
    }
    mark_layout_dirty();
    mark_dirty();
}

void UserTurnActionsNote::clear() {
    if (!actions_enabled_) {
        return;
    }
    actions_enabled_ = false;
    actions_hover_ = false;
    mark_dirty();
    mark_layout_dirty();
}

void UserTurnActionsNote::set_wrap_width(int width) {
    width = std::max(0, width);
    if (wrap_width_hint_ == width) {
        return;
    }
    wrap_width_hint_ = width;
    mark_layout_dirty();
    mark_dirty();
}

bool UserTurnActionsNote::set_actions_hover(bool hovered) {
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

int UserTurnActionsNote::reserved_right_width(bool actions_visible) {
    if (!actions_visible) {
        return 0;
    }
    return glyph_width(kEditGlyph) + kActionGapCols + glyph_width(kCopyGlyph) + kActionGapCols;
}

bool UserTurnActionsNote::show_primary_actions() const {
    return actions_enabled_ && actions_hover_;
}

int UserTurnActionsNote::primary_actions_width() const {
    if (!actions_enabled_) {
        return 0;
    }
    return glyph_width(kEditGlyph) + kActionGapCols + glyph_width(kCopyGlyph);
}

ChatStreamLayout UserTurnActionsNote::chat_stream_layout() const {
    const int width = bounds_.width > 0 ? bounds_.width : (wrap_width_hint_ > 0 ? wrap_width_hint_ : 72);
    return chat_stream_layout_in_widget(width);
}

int UserTurnActionsNote::content_right_x() const {
    const ChatStreamLayout column = chat_stream_layout();
    if (bounds_.width > 0) {
        return std::min(bounds_.width, column.offset_x + column.content_width);
    }
    return column.offset_x + column.content_width;
}

int UserTurnActionsNote::actions_cluster_width() const {
    return show_primary_actions() ? primary_actions_width() : 0;
}

int UserTurnActionsNote::actions_cluster_start_x() const {
    const int cluster_width = actions_cluster_width();
    if (cluster_width <= 0) {
        return content_right_x();
    }
    return std::max(0, content_right_x() - right_inset_ - cluster_width);
}

int UserTurnActionsNote::edit_x() const {
    return actions_cluster_start_x();
}

int UserTurnActionsNote::copy_x() const {
    if (!show_primary_actions()) {
        return actions_cluster_start_x();
    }
    return edit_x() + glyph_width(kEditGlyph) + kActionGapCols;
}

tuinator::Size UserTurnActionsNote::preferred_size() const {
    if (!actions_enabled_) {
        return {0, 0};
    }
    const int width = chat_stream_row_preferred_width(bounds_.width > 0 ? bounds_.width : wrap_width_hint_);
    return {std::max(16, width), 1};
}

void UserTurnActionsNote::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void UserTurnActionsNote::paint(tuinator::PaintContext& ctx) const {
    if (!actions_enabled_ || bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    ui_fill_widget_bounds(ctx, *this, ui_main_bg());

    if (show_primary_actions()) {
        tuinator::Style action_style = ui_resolve_main_text(ui_thinking_style());
        ctx.canvas.draw_text({edit_x(), 0}, kEditGlyph, action_style);
        ctx.canvas.draw_text({copy_x(), 0}, kCopyGlyph, action_style);
    }
}

bool UserTurnActionsNote::handle_event(const tuinator::Event& event) {
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
    const int edit = std::max(0, content_right_x() - right_inset_ - primary_w);
    const int copy = edit + glyph_width(kEditGlyph) + kActionGapCols;

    if (local.x >= copy && local.x < copy + glyph_width(kCopyGlyph)) {
        if (on_action_) {
            on_action_(Action::Copy);
        }
        return true;
    }

    if (local.x >= edit && local.x < edit + glyph_width(kEditGlyph)) {
        if (on_action_) {
            on_action_(Action::Edit);
        }
        return true;
    }

    return false;
}
