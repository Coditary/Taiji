#include "assistant_markdown.hpp"

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
    cached_width_ = 0;
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
    cached_width_ = 0;
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

void AssistantMarkdown::relayout_content() {
    const int width = effective_wrap_width(wrap_width_hint_, bounds_.width);
    const int inner_width = std::max(1, width - content_indent_);
    const int height = std::max(1, view_.content_height());

    cached_width_ = 0;
    view_.layout({0, 0, inner_width, height});
    if (!view_.source().empty() && view_.content_height() <= 0) {
        view_.set_markdown(view_.source());
        view_.layout({0, 0, inner_width, std::max(1, view_.content_height())});
    }
    cached_height_ = std::max(1, view_.content_height());
    mark_layout_dirty();
}

void AssistantMarkdown::refresh_measure_cache(int width) const {
    width = effective_wrap_width(width, bounds_.width);
    if (width <= 0) {
        return;
    }
    const bool needs_rewrap = !view_.source().empty() && view_.content_height() <= 0;
    if (width == cached_width_ && !needs_rewrap) {
        return;
    }

    auto& self = const_cast<AssistantMarkdown&>(*this);
    const int inner_width = std::max(1, width - content_indent_);
    int height = std::max(1, self.view_.content_height());
    self.view_.layout({0, 0, inner_width, height});
    if (!self.view_.source().empty() && self.view_.content_height() <= 0) {
        self.view_.set_markdown(self.view_.source());
        height = std::max(1, self.view_.content_height());
        self.view_.layout({0, 0, inner_width, height});
    }
    self.cached_height_ = std::max(1, self.view_.content_height());
    self.cached_width_ = width;
}

tuinator::Size AssistantMarkdown::preferred_size() const {
    if (view_.source().empty()) {
        return {0, 0};
    }
    const int width =
        chat_stream_row_preferred_width(bounds_.width > 0 ? bounds_.width : (wrap_width_hint_ > 0 ? wrap_width_hint_ : 72));
    refresh_measure_cache(width);
    return {width, std::max(1, view_.content_height())};
}

void AssistantMarkdown::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
    const ChatStreamLayout column = chat_stream_layout_in_widget(bounds.width);
    const int inner_width = std::max(1, column.content_width - content_indent_);
    if (inner_width > 0) {
        refresh_measure_cache(inner_width);
    }

    const int height =
        cached_height_ > 0 ? cached_height_ : (bounds.height > 0 ? bounds.height : view_.content_height());
    view_.layout({0, 0, inner_width, std::max(1, height)});
}

void AssistantMarkdown::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }
    ui_fill_widget_bounds(ctx, *this, kUiMainBackground);
    const ChatStreamLayout column = chat_stream_layout_in_widget(bounds_.width);
    const tuinator::Rect local{column.offset_x, 0, column.content_width, bounds_.height};
    const int inner_width = column.content_width;
    ctx.with_clip(local, [&](tuinator::PaintContext& child_ctx) {
        view_.paint(child_ctx);
        if (stream_glow_ms_ <= 0 || view_.source().empty()) {
            return;
        }
        const int last_row = std::min(bounds_.height - 1, view_.content_height() - 1);
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
        glow.background_rgb = kUiMainBackground;
        glow.foreground_rgb = ui_fade_toward(tuinator::rgb(255, 255, 255), tuinator::rgb(220, 225, 240),
                                             1.0 - static_cast<double>(stream_glow_ms_) / 90.0);
        const int tail_w = tuinator::text_display_width(tail);
        const int x = std::max(content_indent_, inner_width - tail_w);
        child_ctx.canvas.draw_text({x, last_row}, tail, glow);
    });
}
