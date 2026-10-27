#include "chat_input_bar.hpp"

#include "ui_test_ids.hpp"
#include "speech_mic_button.hpp"
#include "ui_theme.hpp"

#include <algorithm>

namespace {

tuinator::Rect mic_local_bounds(const tuinator::Rect& inner_bounds, const tuinator::Size& mic_size) {
    const int local_x = std::max(1, inner_bounds.width - mic_size.width - 3);
    const int local_y = std::max(1, inner_bounds.height - 2);
    return {inner_bounds.x + local_x, inner_bounds.y + local_y, mic_size.width, 1};
}

tuinator::Rect chat_stream_inner_abs(const tuinator::Rect& host_bounds) {
    const ChatStreamLayout column = chat_stream_layout_in_widget(host_bounds.width);
    return {host_bounds.x + column.offset_x, host_bounds.y, column.content_width, host_bounds.height};
}

} // namespace

void ChatInputBar::set_composer(std::unique_ptr<tuinator::Widget> composer) {
    set_test_id(taiji::ui::test_id::kChatInputBar);
    auto column = std::make_unique<CenteredChatColumn>(std::move(composer));
    column->set_apply_top_pad(false);
    column_ = std::move(column);
    if (on_layout_ && column_ != nullptr) {
        column_->set_on_layout(on_layout_);
    }
}

void ChatInputBar::set_on_layout(std::function<void()> callback) {
    Widget::set_on_layout(std::move(callback));
    if (column_ != nullptr) {
        column_->set_on_layout(on_layout_);
    }
}

void ChatInputBar::set_mic(std::unique_ptr<SpeechMicButton> mic) {
    mic_ = std::move(mic);
    if (mic_ != nullptr) {
        mic_->set_test_id(taiji::ui::test_id::kSpeechMic);
    }
}

tuinator::Size ChatInputBar::preferred_size() const {
    const int width = bounds_.width > 0 ? bounds_.width : kChatStreamMaxCols;
    int height = 3;
    if (column_ != nullptr) {
        height = std::max(height, column_->preferred_size().height);
    }
    return {width, height};
}

void ChatInputBar::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
    if (column_ != nullptr) {
        column_->layout(bounds_);
    }
    if (mic_ != nullptr && bounds_.width > 0 && bounds_.height > 0) {
        const tuinator::Rect inner = chat_stream_inner_abs(bounds_);
        if (inner.width > 2 && inner.height > 2) {
            const tuinator::Size mic_size = mic_->preferred_size();
            mic_->layout(mic_local_bounds(inner, mic_size));
        }
    }
}

void ChatInputBar::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width > 0 && bounds_.height > 0) {
        paint_bounds_background(ctx, ui_main_panel_style());
    }
    if (column_ != nullptr) {
        column_->paint(ctx);
    }
    if (mic_ != nullptr && mic_->bounds().width > 0 && mic_->bounds().height > 0) {
        const tuinator::Rect local{
            mic_->bounds().x - bounds_.x,
            mic_->bounds().y - bounds_.y,
            mic_->bounds().width,
            mic_->bounds().height,
        };
        ctx.with_clip(local, [&](tuinator::PaintContext& mic_ctx) { mic_->paint(mic_ctx); });
    }
}

bool ChatInputBar::handle_event(const tuinator::Event& event) {
    if (mic_ != nullptr && mic_->handle_event(event)) {
        return true;
    }
    if (column_ != nullptr) {
        return column_->handle_event(event);
    }
    return false;
}

tuinator::Widget* ChatInputBar::hit_test(tuinator::Point point) {
    if (mic_ != nullptr) {
        if (tuinator::Widget* hit = mic_->hit_test(point)) {
            return hit;
        }
    }
    if (column_ != nullptr) {
        return column_->hit_test(point);
    }
    return nullptr;
}

tuinator::Widget* ChatInputBar::hit_test_focusable(tuinator::Point point) {
    if (column_ != nullptr) {
        return column_->hit_test_focusable(point);
    }
    return nullptr;
}

void ChatInputBar::for_each_descendant(const std::function<void(tuinator::Widget*)>& visitor) {
    if (column_ != nullptr) {
        visitor(column_.get());
        column_->for_each_descendant(visitor);
    }
    if (mic_ != nullptr) {
        visitor(mic_.get());
        mic_->for_each_descendant(visitor);
    }
    Widget::for_each_descendant(visitor);
}
