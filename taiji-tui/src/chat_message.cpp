#include "chat_message.hpp"

#include "ui_chat_column.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

#include <algorithm>
#include <sstream>

namespace {

constexpr int kUserHeaderPadX = 1;
constexpr int kUserBodyPadX = 2;

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

void ChatMessage::set_time_label(std::string time_label) {
    if (time_label_ == time_label) {
        return;
    }
    time_label_ = std::move(time_label);
    mark_dirty();
    mark_layout_dirty();
}

int ChatMessage::content_wrap_cols() const {
    const int body_pad_x = ui_user_bubble_insets().body_pad_x;
    return std::max(8, wrap_cols() - body_pad_x * 2);
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
        const int body_rows = std::max(1, static_cast<int>(lines.size()));
        const UiUserBubbleInsets chrome = ui_user_bubble_insets();
        const int top_rows = ui_corners_flat() ? 1 : 2;
        const int height = top_rows + body_rows + chrome.bottom_inset;
        const int w = chat_stream_row_preferred_width(bounds_.width > 0 ? bounds_.width : wrap_cols());
        return {w, height};
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
        ui_fill_widget_bounds(ctx, *this, kUiMainBackground);

        const int panel_w = ui_chat_stream_panel_width() > 0 ? ui_chat_stream_panel_width() : bounds_.width;
        const ChatStreamLayout column = chat_stream_layout_for_panel(panel_w);
        int box_x = column.offset_x;
        int box_w = column.content_width;
        if (bounds_.width > 0) {
            box_w = std::min(box_w, bounds_.width - box_x);
        }

        const tuinator::Rect panel{box_x, 0, box_w, bounds_.height};
        ui_paint_user_bubble_panel(ctx, panel, kUiQuickActionPillBg);

        const UiUserBubbleInsets chrome = ui_user_bubble_insets();
        const int pad_x = chrome.pad_x;
        const int body_pad_x = chrome.body_pad_x;
        const int bottom_inset = chrome.bottom_inset;

        tuinator::Style header = ui_resolve_main_text(style_);
        header.background_rgb = kUiQuickActionPillBg;
        header.bold = true;
        header.foreground_rgb = tuinator::rgb(235, 232, 245);

        tuinator::Style body_style = ui_resolve_main_text(style_);
        body_style.background_rgb = kUiQuickActionPillBg;
        body_style.bold = false;
        body_style.foreground_rgb = tuinator::rgb(220, 225, 240);

        tuinator::Style time_style = header;
        time_style.bold = false;
        time_style.foreground_rgb = kUiSecondaryText;

        const auto lines = content_lines();
        const int header_y = ui_corners_flat() ? 0 : 1;
        int y = header_y;

        if (y < bounds_.height) {
            const std::string who = std::string(kNfUser) + " You";
            ctx.canvas.draw_text({box_x + pad_x, y}, who, header);
            if (!time_label_.empty()) {
                const int time_w = tuinator::text_display_width(time_label_);
                const int time_x = std::max(box_x + pad_x, box_x + box_w - pad_x - time_w);
                ctx.canvas.draw_text({time_x, y}, time_label_, time_style);
            }
            ++y;
        }

        for (const auto& line : lines) {
            if (y >= bounds_.height - bottom_inset) {
                break;
            }
            paint_plain_line(ctx, box_x + body_pad_x, y, box_w - body_pad_x * 2, line, body_style);
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
