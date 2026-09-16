#include "chat_message.hpp"

#include <tuinator/render/text.hpp>

#include <algorithm>
#include <sstream>

namespace {

constexpr int kUserBarCols = 1;
constexpr int kUserTextIndent = 2;

std::vector<std::string> wrap_paragraph(const std::string& paragraph, int max_cols) {
    if (max_cols <= 0 || paragraph.empty()) {
        return paragraph.empty() ? std::vector<std::string>{} : std::vector<std::string>{paragraph};
    }

    std::vector<std::string> lines;
    std::istringstream words(paragraph);
    std::string word;
    std::string current;

    while (words >> word) {
        const std::string candidate = current.empty() ? word : current + " " + word;
        if (tuinator::text_display_width(candidate) <= max_cols) {
            current = candidate;
            continue;
        }

        if (!current.empty()) {
            lines.push_back(current);
            current = word;
            continue;
        }

        std::size_t offset = 0;
        while (offset < word.size()) {
            const std::size_t bytes =
                tuinator::text_byte_length_for_width(std::string_view(word).substr(offset), max_cols);
            if (bytes == 0) {
                break;
            }
            lines.push_back(word.substr(offset, bytes));
            offset += bytes;
        }
        current.clear();
    }

    if (!current.empty()) {
        lines.push_back(current);
    }
    return lines;
}

std::vector<std::string> wrap_text(const std::string& text, int max_cols) {
    std::vector<std::string> lines;
    std::size_t start = 0;
    while (start <= text.size()) {
        const std::size_t end = text.find('\n', start);
        const std::string paragraph =
            end == std::string::npos ? text.substr(start) : text.substr(start, end - start);
        const auto wrapped = wrap_paragraph(paragraph, max_cols);
        lines.insert(lines.end(), wrapped.begin(), wrapped.end());
        if (end == std::string::npos) {
            break;
        }
        start = end + 1;
    }
    if (lines.empty() && !text.empty()) {
        lines.push_back(text);
    }
    return lines;
}

void paint_plain_line(tuinator::PaintContext& ctx, int x, int y, int max_columns, const std::string& line,
                      const tuinator::Style& style) {
    if (max_columns <= 0) {
        return;
    }
    const std::size_t bytes = tuinator::text_byte_length_for_width(line, max_columns);
    if (bytes == 0) {
        return;
    }
    ctx.canvas.draw_text({x, y}, line.substr(0, bytes), style);
}

} // namespace

ChatMessage::ChatMessage(std::string text, tuinator::Style style, Align align, Variant variant)
    : text_(std::move(text)), style_(style), align_(align), variant_(variant) {}

void ChatMessage::set_text(std::string text) {
    if (text_ == text) {
        return;
    }
    text_ = std::move(text);
    mark_dirty();
    mark_layout_dirty();
}

void ChatMessage::set_style(tuinator::Style style) {
    style_ = style;
    mark_dirty();
}

void ChatMessage::set_variant(Variant variant) {
    if (variant_ == variant) {
        return;
    }
    variant_ = variant;
    mark_dirty();
    mark_layout_dirty();
}

void ChatMessage::set_wrap_width(int width) {
    width = std::max(0, width);
    if (wrap_width_hint_ == width) {
        return;
    }
    wrap_width_hint_ = width;
    mark_layout_dirty();
}

int ChatMessage::wrap_cols() const {
    if (bounds_.width > 0) {
        return std::max(8, bounds_.width);
    }
    if (wrap_width_hint_ > 0) {
        return std::max(8, wrap_width_hint_);
    }
    return 72;
}

int ChatMessage::content_wrap_cols() const {
    return std::max(8, wrap_cols() - kUserTextIndent);
}

std::vector<std::string> ChatMessage::content_lines() const {
    return wrap_text(text_, content_wrap_cols());
}

std::vector<std::string> ChatMessage::wrapped_lines() const {
    return wrap_text(text_, wrap_cols());
}

tuinator::Size ChatMessage::preferred_size() const {
    if (text_.empty()) {
        return {0, 0};
    }

    if (variant_ == Variant::UserPrompt) {
        const auto lines = content_lines();
        const int height = 1 + std::max(1, static_cast<int>(lines.size()));
        return {wrap_cols(), height};
    }

    const int cols = wrap_cols();
    const auto lines = wrap_text(text_, cols);
    int max_width = 0;
    for (const auto& line : lines) {
        max_width = std::max(max_width, tuinator::text_display_width(line));
    }
    return {max_width, std::max(1, static_cast<int>(lines.size()))};
}

void ChatMessage::layout(tuinator::Rect bounds) {
    if (bounds_.width != bounds.width) {
        bounds_ = bounds;
        mark_layout_dirty();
        return;
    }
    bounds_ = bounds;
}

void ChatMessage::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0 || text_.empty()) {
        return;
    }

    if (variant_ == Variant::UserPrompt) {
        paint_bounds_background(ctx, style_);

        tuinator::Style bar_style = style_;
        bar_style.bold = true;

        tuinator::Style label_style = style_;
        label_style.bold = true;

        tuinator::Style body_style = style_;
        body_style.bold = false;

        const auto lines = content_lines();
        int y = 0;

        if (y < bounds_.height) {
            ctx.canvas.draw_text({0, y}, "│", bar_style);
            paint_plain_line(ctx, kUserTextIndent, y, bounds_.width - kUserTextIndent, "Du", label_style);
            ++y;
        }

        for (const auto& line : lines) {
            if (y >= bounds_.height) {
                break;
            }
            ctx.canvas.draw_text({0, y}, "│", bar_style);
            paint_plain_line(ctx, kUserTextIndent, y, bounds_.width - kUserTextIndent, line, body_style);
            ++y;
        }
        return;
    }

    const tuinator::Style base_style = ctx.styles().text(*this, style_);
    int y = 0;

    const auto lines = wrapped_lines();
    for (const auto& line : lines) {
        if (y >= bounds_.height) {
            break;
        }
        const int line_width = tuinator::text_display_width(line);
        const int x = align_ == Align::Left ? 0 : std::max(0, bounds_.width - line_width);
        const int max_columns = std::max(0, bounds_.width - x);
        paint_plain_line(ctx, x, y, max_columns, line, base_style);
        ++y;
    }
}
