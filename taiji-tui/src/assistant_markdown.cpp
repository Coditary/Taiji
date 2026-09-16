#include "assistant_markdown.hpp"

#include <algorithm>

namespace {

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
    relayout_content();
}

void AssistantMarkdown::set_markdown(std::string text) {
    view_.set_markdown(std::move(text));
    relayout_content();
}

void AssistantMarkdown::clear() {
    if (view_.source().empty() && view_.content_height() <= 0) {
        return;
    }
    view_.clear();
    cached_width_ = 0;
    cached_height_ = 1;
    mark_layout_dirty();
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
    if (text.empty()) {
        return;
    }

    const std::string& existing = view_.source();
    if (existing.empty()) {
        view_.append(text);
    } else if (text.size() >= existing.size() && text.compare(0, existing.size(), existing) == 0) {
        if (text.size() > existing.size()) {
            view_.append(std::string_view(text).substr(existing.size()));
        } else if (view_.content_height() > 0) {
            return;
        }
    } else {
        set_markdown(text);
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
    const int width = wrap_width_hint_ > 0 ? wrap_width_hint_ : 72;
    refresh_measure_cache(width);
    return {width, std::max(1, view_.content_height())};
}

void AssistantMarkdown::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
    if (bounds.width > 0) {
        refresh_measure_cache(bounds.width);
    }

    const int inner_width = std::max(1, bounds.width - content_indent_);
    const int height = std::max(bounds.height, cached_height_);
    view_.layout({0, 0, inner_width, height});
}

void AssistantMarkdown::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }
    view_.paint(ctx);
}
