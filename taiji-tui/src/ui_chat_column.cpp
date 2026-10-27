#include "ui_chat_column.hpp"

#include "ui_theme.hpp"

#include <algorithm>

namespace {

int g_chat_stream_max_cols = kChatStreamMaxCols;
int g_chat_stream_panel_width = 0;

} // namespace

void ui_set_chat_stream_max_cols(int max_cols) {
    g_chat_stream_max_cols = max_cols > 0 ? max_cols : kChatStreamMaxCols;
}

void ui_set_chat_stream_panel_width(int panel_width) {
    g_chat_stream_panel_width = std::max(0, panel_width);
}

int ui_chat_stream_panel_width() {
    return g_chat_stream_panel_width;
}

int chat_stream_content_width(int panel_width) {
    if (panel_width <= 0) {
        return kChatStreamMaxCols;
    }
    const int inner = std::max(16, panel_width - kChatStreamPadX * 2);
    return std::min(g_chat_stream_max_cols, inner);
}

ChatStreamLayout chat_stream_layout_for_panel(int panel_width) {
    return chat_stream_layout_for_panel(panel_width, g_chat_stream_max_cols);
}

ChatStreamLayout chat_stream_layout_for_panel(int panel_width, int max_content_cols) {
    if (g_chat_stream_panel_width > 0) {
        panel_width = g_chat_stream_panel_width;
    }
    ChatStreamLayout out;
    if (panel_width <= 0) {
        out.content_width = std::max(16, max_content_cols);
        return out;
    }
    const int usable = std::max(0, panel_width - kChatStreamPadX * 2);
    out.content_width = std::min(max_content_cols, usable);
    out.offset_x = kChatStreamPadX + std::max(0, (usable - out.content_width) / 2);
    return out;
}

ChatStreamLayout chat_stream_layout_in_widget(int widget_width) {
    const int panel_w = g_chat_stream_panel_width > 0 ? g_chat_stream_panel_width : widget_width;
    ChatStreamLayout column = chat_stream_layout_for_panel(panel_w);
    if (widget_width > 0 && panel_w > widget_width) {
        const int shift = (panel_w - widget_width) / 2;
        column.offset_x = std::max(0, column.offset_x - shift);
        column.content_width = std::min(column.content_width, widget_width - column.offset_x);
    }
    return column;
}

int chat_bubble_max_cols(int stream_width_cols, int max_width_pct) {
    if (max_width_pct <= 0) {
        max_width_pct = kChatBubbleMaxWidthPercent;
    }
    if (max_width_pct < 40) {
        max_width_pct = 40;
    }
    if (max_width_pct > 100) {
        max_width_pct = 100;
    }
    if (stream_width_cols <= 0) {
        return std::max(12, kChatStreamMaxCols * max_width_pct / 100);
    }
    return std::max(12, stream_width_cols * max_width_pct / 100);
}

int chat_stream_row_preferred_width(int widget_bounds_width) {
    if (g_chat_stream_panel_width > 0) {
        return g_chat_stream_panel_width;
    }
    if (widget_bounds_width > 0) {
        return widget_bounds_width;
    }
    return chat_stream_content_width(0);
}

CenteredChatColumn::CenteredChatColumn(std::unique_ptr<tuinator::Widget> content) : content_(std::move(content)) {
    if (content_ != nullptr) {
        attach_child_widget(content_.get());
        if (on_layout_) {
            content_->set_on_layout(on_layout_);
        }
        if (on_dirty_) {
            content_->set_on_dirty(on_dirty_);
        }
    }
}

void CenteredChatColumn::set_on_layout(std::function<void()> callback) {
    Widget::set_on_layout(std::move(callback));
    if (content_ != nullptr) {
        content_->set_on_layout(on_layout_);
    }
}

void CenteredChatColumn::set_on_dirty(std::function<void(tuinator::Rect)> callback) {
    Widget::set_on_dirty(std::move(callback));
    if (content_ != nullptr) {
        content_->set_on_dirty(on_dirty_);
    }
}

void CenteredChatColumn::set_max_content_width(int cols) {
    cols = std::max(16, cols);
    if (max_content_width_ == cols) {
        return;
    }
    max_content_width_ = cols;
    mark_layout_dirty();
}

tuinator::Size CenteredChatColumn::preferred_size() const {
    const int panel_w = last_panel_width_ > 0 ? last_panel_width_ : kChatStreamMaxCols;
    const int top_pad = apply_top_pad_ ? kChatStreamPadTop : 0;
    if (content_ == nullptr) {
        return {panel_w, 1 + top_pad};
    }
    const tuinator::Size child = content_->preferred_size();
    return {panel_w, child.height + top_pad};
}

void CenteredChatColumn::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
    last_panel_width_ = bounds_.width;
    if (content_ == nullptr || bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    const int top_pad = apply_top_pad_ ? kChatStreamPadTop : 0;
    const int avail_h = std::max(0, bounds_.height - top_pad);
    const tuinator::Size pref = content_->preferred_size();
    const int content_h = std::min(avail_h, std::max(1, pref.height));
    const int y = bounds_.y + top_pad + (avail_h - content_h);
    const ChatStreamLayout column = chat_stream_layout_in_widget(bounds_.width);
    content_->layout({bounds_.x + column.offset_x, y, column.content_width, content_h});
}

void CenteredChatColumn::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }
    ui_fill_widget_bounds(ctx, *this, ui_main_bg());
    if (content_ == nullptr) {
        return;
    }
    const ChatStreamLayout column = chat_stream_layout_in_widget(bounds_.width);
    const tuinator::Rect local{column.offset_x, 0, column.content_width, bounds_.height};
    ctx.with_clip(local, [&](tuinator::PaintContext& child_ctx) { content_->paint(child_ctx); });
}

bool CenteredChatColumn::handle_event(const tuinator::Event& event) {
    return content_ != nullptr && content_->handle_event(event);
}

tuinator::Widget* CenteredChatColumn::hit_test(tuinator::Point point) {
    if (content_ == nullptr) {
        return nullptr;
    }
    return content_->hit_test(point);
}

tuinator::Widget* CenteredChatColumn::hit_test_focusable(tuinator::Point point) {
    if (content_ == nullptr) {
        return nullptr;
    }
    return content_->hit_test_focusable(point);
}

void CenteredChatColumn::collect_focusable(std::vector<tuinator::Widget*>& out) {
    if (content_ != nullptr) {
        content_->collect_focusable(out);
    }
}

void CenteredChatColumn::for_each_descendant(const std::function<void(tuinator::Widget*)>& visitor) {
    if (content_ != nullptr) {
        visitor(content_.get());
        content_->for_each_descendant(visitor);
    }
    Widget::for_each_descendant(visitor);
}
