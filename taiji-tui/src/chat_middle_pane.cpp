#include "chat_middle_pane.hpp"

#include "chat_message.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/scrollbar/core.hpp>
#include <tuinator/widgets/containers/scroll_view.hpp>

namespace {

constexpr int kStickyUserBodyLines = 3;

int chat_scroll_viewport_width(const tuinator::ScrollView& scroll) {
    const int w = scroll.bounds().width;
    const int h = scroll.bounds().height;
    if (w <= 0 || h <= 0) {
        return w;
    }
    if (scroll.max_scroll_y() <= 0) {
        return w;
    }
    const int content_height = scroll.max_scroll_y() + h;
    const tuinator::ScrollbarMetrics metrics = tuinator::compute_scrollbar_metrics(
        w, h, w, content_height, scroll.options().scrollbars.config);
    return metrics.viewport_width > 0 ? metrics.viewport_width : w;
}

} // namespace

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

void ChatMiddlePane::set_sticky_user_message_enabled(bool enabled) {
    if (sticky_user_message_enabled_ == enabled) {
        return;
    }
    sticky_user_message_enabled_ = enabled;
    mark_layout_dirty();
    mark_dirty();
}

void ChatMiddlePane::set_sticky_user_resolver(std::function<const ChatMessage*()> resolver) {
    sticky_user_resolver_ = std::move(resolver);
    mark_dirty();
}

int ChatMiddlePane::sticky_overlay_height() const {
    if (show_empty_ || !sticky_user_message_enabled_ || sticky_user_resolver_ == nullptr) {
        return 0;
    }
    const ChatMessage* sticky = sticky_user_resolver_();
    if (sticky == nullptr) {
        return 0;
    }
    return sticky->sticky_preview_height(kStickyUserBodyLines);
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
            // Keep the scroll viewport at full height so the scrollbar thumb size stays stable
            // while sticky is painted as an overlay (not by shrinking this widget).
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
        const tuinator::Rect local{
            empty_state_->bounds().x - bounds_.x,
            empty_state_->bounds().y - bounds_.y,
            empty_state_->bounds().width,
            empty_state_->bounds().height,
        };
        ctx.with_clip(local, [&](tuinator::PaintContext& child_ctx) { empty_state_->paint(child_ctx); });
        return;
    }

    const int sticky_h = sticky_overlay_height();
    auto* scroll_view = dynamic_cast<tuinator::ScrollView*>(chat_scroll_.get());

    if (chat_scroll_ != nullptr) {
        const tuinator::Rect local{
            chat_scroll_->bounds().x - bounds_.x,
            chat_scroll_->bounds().y - bounds_.y,
            chat_scroll_->bounds().width,
            chat_scroll_->bounds().height,
        };
        ctx.with_clip(local, [&](tuinator::PaintContext& child_ctx) { chat_scroll_->paint(child_ctx); });
    }

    if (sticky_h <= 0 || !sticky_user_message_enabled_ || sticky_user_resolver_ == nullptr) {
        return;
    }
    const ChatMessage* sticky = sticky_user_resolver_();
    if (sticky == nullptr) {
        return;
    }

    const int content_w =
        scroll_view != nullptr ? chat_scroll_viewport_width(*scroll_view) : bounds_.width;
    const int sticky_w = std::max(0, std::min(content_w, bounds_.width));
    ctx.with_clip({0, 0, sticky_w, sticky_h}, [&](tuinator::PaintContext& sticky_ctx) {
        sticky->paint_sticky_preview(sticky_ctx, sticky_w, kStickyUserBodyLines);
    });
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
