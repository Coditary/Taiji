#include "chat_middle_pane.hpp"

#include "ui_theme.hpp"

ChatMiddlePane::ChatMiddlePane(std::unique_ptr<tuinator::Widget> empty_state,
                               std::unique_ptr<tuinator::Widget> chat_scroll)
    : empty_state_(std::move(empty_state)), chat_scroll_(std::move(chat_scroll)) {}

void ChatMiddlePane::set_show_empty(bool show) {
    if (show_empty_ == show) {
        return;
    }
    show_empty_ = show;
    mark_layout_dirty();
    mark_dirty();
}

tuinator::Size ChatMiddlePane::preferred_size() const {
    int height = 8;
    if (show_empty_ && empty_state_ != nullptr) {
        height = std::max(height, empty_state_->preferred_size().height);
    } else if (chat_scroll_ != nullptr) {
        height = std::max(height, chat_scroll_->preferred_size().height);
    }
    return {0, height};
}

void ChatMiddlePane::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
    const tuinator::Rect hidden{bounds.x, bounds.y, 0, 0};
    if (show_empty_) {
        if (empty_state_ != nullptr) {
            empty_state_->layout(bounds);
        }
        if (chat_scroll_ != nullptr) {
            chat_scroll_->layout(hidden);
        }
    } else {
        if (empty_state_ != nullptr) {
            empty_state_->layout(hidden);
        }
        if (chat_scroll_ != nullptr) {
            chat_scroll_->layout(bounds);
        }
    }
}

void ChatMiddlePane::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }
    paint_bounds_background(ctx, ui_main_panel_style());
    if (show_empty_ && empty_state_ != nullptr) {
        empty_state_->paint(ctx);
        return;
    }
    if (chat_scroll_ != nullptr) {
        chat_scroll_->paint(ctx);
    }
}

bool ChatMiddlePane::handle_event(const tuinator::Event& event) {
    tuinator::Widget* target = show_empty_ ? empty_state_.get() : chat_scroll_.get();
    if (target == nullptr) {
        return false;
    }
    return target->handle_event(event);
}

tuinator::Widget* ChatMiddlePane::hit_test(tuinator::Point point) {
    if (!bounds_.contains(point)) {
        return nullptr;
    }
    tuinator::Widget* target = show_empty_ ? empty_state_.get() : chat_scroll_.get();
    if (target == nullptr) {
        return this;
    }
    return target->hit_test(point);
}

tuinator::Widget* ChatMiddlePane::hit_test_focusable(tuinator::Point point) {
    if (!bounds_.contains(point)) {
        return nullptr;
    }
    if (show_empty_) {
        return nullptr;
    }
    if (chat_scroll_ != nullptr) {
        return chat_scroll_->hit_test_focusable(point);
    }
    return nullptr;
}

void ChatMiddlePane::collect_focusable(std::vector<tuinator::Widget*>& out) {
    if (!show_empty_ && chat_scroll_ != nullptr) {
        chat_scroll_->collect_focusable(out);
    }
}
