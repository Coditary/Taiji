#include "assistant_markdown.hpp"

#include "chat_message_chrome.hpp"
#include "ui_anim.hpp"
#include "ui_chat_column.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

#include <algorithm>

namespace {

std::string trim_trailing_whitespace(std::string text) {
    while (!text.empty()) {
        const char ch = text.back();
        if (ch != '\n' && ch != '\r' && ch != ' ' && ch != '\t') {
            break;
        }
        text.pop_back();
    }
    return text;
}

int effective_wrap_width(int wrap_width_hint, int bounds_width) {
    if (wrap_width_hint > 0) {
        return wrap_width_hint;
    }
    if (bounds_width > 0) {
        return bounds_width;
    }
    return 72;
}

} // namespace

AssistantMarkdown::AssistantMarkdown(tuinator::MarkdownViewOptions options) : view_(std::move(options)) {
    view_.set_follow_tail(false);
}

void AssistantMarkdown::set_markdown_styles(tuinator::MarkdownStyleSet styles) {
    view_.set_styles(std::move(styles));
    mark_dirty();
}

void AssistantMarkdown::set_author_label(std::string label) {
    if (author_label_ == label) {
        return;
    }
    author_label_ = std::move(label);
    mark_dirty();
    mark_layout_dirty();
}

void AssistantMarkdown::invalidate_for_user_style_change() {
    cached_inner_width_ = 0;
    mark_dirty();
    mark_layout_dirty();
    relayout_content();
}

void AssistantMarkdown::append(std::string_view chunk) {
    if (chunk.empty()) {
        return;
    }
    view_.append(chunk);
    stream_glow_ms_ = 90;
    relayout_content();
}

void AssistantMarkdown::tick_stream_glow(int dt_ms) {
    if (stream_glow_ms_ <= 0) {
        return;
    }
    stream_glow_ms_ = std::max(0, stream_glow_ms_ - dt_ms);
    mark_dirty();
}

void AssistantMarkdown::set_markdown(std::string text) {
    view_.set_markdown(trim_trailing_whitespace(std::move(text)));
    relayout_content();
}

void AssistantMarkdown::clear() {
    if (view_.source().empty() && view_.content_height() <= 0) {
        return;
    }
    view_.clear();
    cached_inner_width_ = 0;
    cached_height_ = 0;
    mark_layout_dirty();
}

bool AssistantMarkdown::is_empty() const {
    return view_.source().empty();
}

void AssistantMarkdown::set_wrap_width(int width) {
    width = std::max(0, width);
    if (wrap_width_hint_ == width) {
        return;
    }
    wrap_width_hint_ = width;
    cached_inner_width_ = 0;
    mark_layout_dirty();
}

void AssistantMarkdown::sync_from(const std::string& text, bool clear_first) {
    if (clear_first) {
        clear();
    }
    std::string normalized = trim_trailing_whitespace(text);
    if (normalized.empty()) {
        return;
    }

    const std::string& existing = view_.source();
    if (existing.empty()) {
        view_.append(normalized);
    } else if (normalized.size() >= existing.size() &&
               normalized.compare(0, existing.size(), existing) == 0) {
        if (normalized.size() > existing.size()) {
            view_.append(std::string_view(normalized).substr(existing.size()));
        } else if (view_.content_height() > 0) {
            return;
        }
    } else {
        set_markdown(std::move(normalized));
        return;
    }

    relayout_content();
}

int AssistantMarkdown::stream_content_width() const {
    const int panel =
        effective_wrap_width(wrap_width_hint_, bounds_.width > 0 ? bounds_.width : chat_stream_row_preferred_width(0));
    const ChatStreamLayout column = chat_stream_layout_in_widget(panel);
    return std::max(1, column.content_width);
}

int AssistantMarkdown::markdown_inner_width() const {
    const UiChatRolePrefs prefs = ui_assistant_chat_prefs();
    const int stream_w = stream_content_width();
    const ChatFrameLayout frame = chat_frame_layout(0, stream_w, 8, stream_w, prefs);
    if (prefs.style == UiUserMessageStyle::Messenger) {
        const int pad_x = ui_message_fill_background() ? 0 : kBubblePadX;
        return std::max(8, frame.content_w - pad_x * 2);
    }
    return std::max(8, frame.body_w);
}

void AssistantMarkdown::relayout_content() {
    const int inner_width = markdown_inner_width();
    const int height = std::max(1, view_.content_height());

    cached_inner_width_ = 0;
    view_.layout({0, 0, inner_width, height});
    if (!view_.source().empty() && view_.content_height() <= 0) {
        view_.set_markdown(view_.source());
        view_.layout({0, 0, inner_width, std::max(1, view_.content_height())});
    }
    cached_height_ = std::max(1, view_.content_height());
    mark_layout_dirty();
}

void AssistantMarkdown::refresh_measure_cache(int inner_width) const {
    inner_width = std::max(8, inner_width);
    const bool needs_rewrap = !view_.source().empty() && view_.content_height() <= 0;
    if (inner_width == cached_inner_width_ && !needs_rewrap) {
        return;
    }

    auto& self = const_cast<AssistantMarkdown&>(*this);
    int height = std::max(1, self.view_.content_height());
    self.view_.layout({0, 0, inner_width, height});
    if (!self.view_.source().empty() && self.view_.content_height() <= 0) {
        self.view_.set_markdown(self.view_.source());
        height = std::max(1, self.view_.content_height());
        self.view_.layout({0, 0, inner_width, height});
    }
    self.cached_height_ = std::max(1, self.view_.content_height());
    self.cached_inner_width_ = inner_width;
}

tuinator::Size AssistantMarkdown::preferred_size() const {
    if (view_.source().empty()) {
        return {0, 0};
    }
    const UiChatRolePrefs prefs = ui_assistant_chat_prefs();
    const int width =
        chat_stream_row_preferred_width(bounds_.width > 0 ? bounds_.width : (wrap_width_hint_ > 0 ? wrap_width_hint_ : 72));
    refresh_measure_cache(markdown_inner_width());
    int height = cached_height_;
    if (prefs.style == UiUserMessageStyle::Messenger && !ui_message_fill_background()) {
        height += kBubblePadY * 2;
    }
    if (prefs.style == UiUserMessageStyle::Messenger && !author_label_.empty()) {
        height += 1;
    }
    if (prefs.style == UiUserMessageStyle::Cornered) {
        height += 2;
    } else if (prefs.style == UiUserMessageStyle::Header) {
        height += 1;
    } else if (prefs.style == UiUserMessageStyle::Block) {
        const UiUserBubbleInsets chrome = ui_user_bubble_insets();
        height += chrome.bottom_inset + (author_label_.empty() ? 0 : 1);
        if (!ui_corners_flat()) {
            height += 1;
        }
    } else if (!author_label_.empty()) {
        height += 1;
    }
    return {width, std::max(1, height)};
}

void AssistantMarkdown::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
    const int inner_width = markdown_inner_width();
    refresh_measure_cache(inner_width);

    const int height =
        cached_height_ > 0 ? cached_height_ : (bounds.height > 0 ? bounds.height : view_.content_height());
    view_.layout({0, 0, inner_width, std::max(1, height)});
}

void AssistantMarkdown::paint_stream_glow(tuinator::PaintContext& ctx, const tuinator::Rect& inner,
                                          tuinator::Rgb bg) const {
    if (stream_glow_ms_ <= 0 || view_.source().empty()) {
        return;
    }
    const int last_row = std::min(inner.height - 1, view_.content_height() - 1);
    if (last_row < 0) {
        return;
    }
    const std::string& src = view_.source();
    const std::size_t end = src.find_last_not_of(" \n\r\t");
    if (end == std::string::npos) {
        return;
    }
    std::size_t start = src.find_last_of(" \n\r\t", end);
    if (start == std::string::npos) {
        start = 0;
    } else {
        ++start;
    }
    const std::string tail = src.substr(start, end - start + 1);
    if (tail.empty()) {
        return;
    }
    tuinator::Style glow;
    glow.background_rgb = bg;
    glow.foreground_rgb =
        ui_fade_toward(ui_text(), ui_secondary_text(), 1.0 - static_cast<double>(stream_glow_ms_) / 90.0);
    const int tail_w = tuinator::text_display_width(tail);
    const int x = std::max(0, inner.width - tail_w);
    ctx.canvas.draw_text({x, last_row}, tail, glow);
}

void AssistantMarkdown::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0 || view_.source().empty()) {
        return;
    }
    ui_fill_widget_bounds(ctx, *this, ui_main_bg());
    const ChatStreamLayout column = chat_stream_layout_in_widget(bounds_.width);
    const int box_x = column.offset_x;
    const int box_w = column.content_width;
    const UiChatRolePrefs prefs = ui_assistant_chat_prefs();
    const std::string& title = author_label_;
    const int header_min = chat_header_strip_min_width(title, "");
    const int body_text_w = std::max(markdown_inner_width(), header_min);
    const ChatFrameLayout frame = chat_frame_layout(box_x, box_w, header_min, body_text_w, prefs);

    tuinator::Style accent = ui_thinking_style();
    tuinator::Style body = ui_resolve_main_text(accent);
    body.bold = false;
    body.foreground_rgb = ui_text();

    int y = 0;
    const tuinator::Rgb panel_bg = ui_message_panel_bg();

    switch (prefs.style) {
        case UiUserMessageStyle::Messenger: {
            if (!title.empty() && y < bounds_.height) {
                ctx.canvas.draw_text({frame.frame_x, y}, title, accent);
                ++y;
            }
            const int bubble_h = std::max(1, bounds_.height - y);
            const tuinator::Rect bubble{frame.frame_x, y, frame.content_w, bubble_h};
            ui_paint_chat_bubble_panel(ctx, bubble, ui_quick_action_pill_bg(), ui_secondary_text());
            const int pad_x = ui_message_fill_background() ? 0 : kBubblePadX;
            const int pad_y = ui_message_fill_background() ? 0 : kBubblePadY;
            const tuinator::Rect inner{bubble.x + pad_x, bubble.y + pad_y,
                                       std::max(1, bubble.width - pad_x * 2),
                                       std::max(1, bubble.height - pad_y * 2)};
            if (ui_message_fill_background()) {
                tuinator::Style inner_fill;
                inner_fill.background_rgb = ui_message_panel_bg();
                ctx.canvas.fill_rect(inner, ' ', inner_fill);
            }
            ctx.with_clip(inner, [&](tuinator::PaintContext& child_ctx) {
                view_.paint(child_ctx);
                paint_stream_glow(child_ctx, inner, ui_message_panel_bg());
            });
            return;
        }
        case UiUserMessageStyle::Cornered: {
            if (y < bounds_.height) {
                paint_chat_header_strip(ctx, y, frame, title, "", accent, accent);
                ++y;
            }
            const int body_top = y;
            const int body_rows = std::max(1, view_.content_height());
            paint_chat_framed_body_fill(ctx, frame, body_top, body_rows);
            ctx.with_clip({frame.body_x, body_top, frame.body_w, body_rows}, [&](tuinator::PaintContext& child_ctx) {
                view_.paint(child_ctx);
                paint_stream_glow(child_ctx, {0, 0, frame.body_w, body_rows}, panel_bg);
            });
            y = body_top + body_rows;
            if (y < bounds_.height) {
                paint_chat_cornered_bottom(ctx, y, frame, accent);
            }
            return;
        }
        case UiUserMessageStyle::Header: {
            if (y < bounds_.height) {
                paint_chat_header_strip(ctx, y, frame, title, "", accent, accent);
                ++y;
            }
            const int body_rows = std::max(1, view_.content_height());
            paint_chat_framed_body_fill(ctx, frame, y, body_rows);
            ctx.with_clip({frame.body_x, y, frame.body_w, body_rows}, [&](tuinator::PaintContext& child_ctx) {
                view_.paint(child_ctx);
                paint_stream_glow(child_ctx, {0, 0, frame.body_w, body_rows}, panel_bg);
            });
            return;
        }
        case UiUserMessageStyle::Block: {
            const UiUserBubbleInsets chrome = ui_user_bubble_insets();
            const int body_rows = std::max(1, view_.content_height());
            const int panel_h = std::max(3, body_rows + chrome.bottom_inset + (title.empty() ? 0 : 1));
            const tuinator::Rect panel{frame.frame_x, y, frame.content_w, std::min(panel_h, bounds_.height - y)};
            if (panel.height <= 0) {
                return;
            }
            ui_paint_chat_bubble_panel(ctx, panel, ui_quick_action_pill_bg(), ui_secondary_text());
            int ty = panel.y + (ui_corners_flat() ? 0 : 1);
            if (!title.empty() && ty < panel.y + panel.height) {
                ctx.canvas.draw_text({panel.x + chrome.pad_x, ty}, title, accent);
                ++ty;
            }
            const int body_top = ty;
            if (ui_message_fill_background()) {
                tuinator::Style fill;
                fill.background_rgb = panel_bg;
                const int fill_h = std::max(0, panel.y + panel.height - chrome.bottom_inset - body_top);
                if (fill_h > 0) {
                    ctx.canvas.fill_rect({panel.x + 1, body_top, std::max(1, panel.width - 2), fill_h}, ' ', fill);
                }
            }
            const tuinator::Rect inner{panel.x + chrome.body_pad_x, body_top,
                                       std::max(1, panel.width - chrome.body_pad_x * 2),
                                       std::max(1, panel.y + panel.height - chrome.bottom_inset - body_top)};
            ctx.with_clip(inner, [&](tuinator::PaintContext& child_ctx) {
                view_.paint(child_ctx);
                paint_stream_glow(child_ctx, inner, panel_bg);
            });
            return;
        }
        default: {
            if (!title.empty() && y < bounds_.height) {
                ctx.canvas.draw_text({frame.frame_x, y}, title, accent);
                ++y;
            }
            const int body_rows = std::max(1, view_.content_height());
            ctx.with_clip({frame.body_x, y, frame.body_w, body_rows}, [&](tuinator::PaintContext& child_ctx) {
                view_.paint(child_ctx);
                paint_stream_glow(child_ctx, {0, 0, frame.body_w, body_rows}, panel_bg);
            });
            return;
        }
    }
}
