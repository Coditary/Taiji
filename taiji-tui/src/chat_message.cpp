#include "chat_message.hpp"

#include "chat_message_chrome.hpp"
#include "ui_chat_column.hpp"
#include "ui_locale.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/glyphs.hpp>
#include <tuinator/render/text.hpp>

#include <algorithm>
#include <cctype>
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

struct UserStreamBox {
    int x = 0;
    int w = 0;
};

UserStreamBox user_stream_box(const tuinator::Rect& bounds) {
    UserStreamBox out;
    if (bounds.width <= 0) {
        return out;
    }
    const ChatStreamLayout column = chat_stream_layout_in_widget(bounds.width);
    out.x = column.offset_x;
    out.w = std::max(0, std::min(column.content_width, bounds.width - out.x));
    return out;
}

int user_bubble_outer_width(int stream_w, int content_line_w, const UiChatRolePrefs& prefs, int horizontal_pad = 4) {
    if (stream_w <= 0) {
        return content_line_w + horizontal_pad;
    }
    return std::min(ui_chat_frame_max_cols(stream_w, prefs), content_line_w + horizontal_pad);
}

std::string repeat_glyph(const std::string& glyph, int count) {
    if (count <= 0) {
        return {};
    }
    std::string out;
    out.reserve(static_cast<std::size_t>(count) * glyph.size());
    for (int i = 0; i < count; ++i) {
        out += glyph;
    }
    return out;
}

void paint_horizontal_run(tuinator::PaintContext& ctx, int y, int x, int width, const tuinator::Style& style) {
    if (width <= 0) {
        return;
    }
    const tuinator::BorderGlyphs glyphs = ui_frame_border_glyphs();
    const std::string cell = glyphs.horizontal.empty() ? "-" : glyphs.horizontal;
    const int cell_w = std::max(1, tuinator::text_display_width(cell));
    int col = x;
    const int end = x + width;
    while (col + cell_w <= end) {
        ctx.canvas.draw_text({col, y}, cell, style);
        col += cell_w;
    }
    if (col < end) {
        ctx.canvas.draw_char({col, y}, '-', style);
    }
}

void paint_centered_rule(tuinator::PaintContext& ctx, int y, int x, int width, const std::string& label,
                         const tuinator::Style& style) {
    if (width <= 0) {
        return;
    }
    const int label_w = tuinator::text_display_width(label);
    if (label_w >= width) {
        paint_plain_line(ctx, x, y, width, label, style);
        return;
    }
    const int dash_total = width - label_w;
    const int left = dash_total / 2;
    const int right = dash_total - left;
    paint_horizontal_run(ctx, y, x, left, style);
    ctx.canvas.draw_text({x + left, y}, label, style);
    paint_horizontal_run(ctx, y, x + left + label_w, right, style);
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

void ChatMessage::set_author_label(std::string label) {
    if (author_label_ == label) {
        return;
    }
    author_label_ = std::move(label);
    mark_dirty();
    mark_layout_dirty();
}

void ChatMessage::set_compact_summary(bool compact) {
    if (compact_summary_ == compact) {
        return;
    }
    compact_summary_ = compact;
    mark_dirty();
    mark_layout_dirty();
}

std::string ChatMessage::speaker_name() const {
    if (author_label_.empty()) {
        const std::string you = ui_shell("chat.author_you");
        return you.empty() ? "You" : you;
    }
    return author_label_;
}

std::string ChatMessage::log_speaker_name() const {
    std::string name = speaker_name();
    for (char& ch : name) {
        ch = static_cast<char>(std::tolower(static_cast<unsigned char>(ch)));
    }
    return name;
}

int ChatMessage::content_wrap_cols() const {
    const UserStreamBox box = user_stream_box(bounds_.width > 0 ? bounds_ : tuinator::Rect{0, 0, wrap_cols(), 1});
    const int stream_w = box.w > 0 ? box.w : wrap_cols();
    if (compact_summary_) {
        return std::max(8, stream_w);
    }
    const UiChatRolePrefs prefs = ui_user_chat_prefs();
    switch (prefs.style) {
        case UiUserMessageStyle::Block: {
            const int body_pad_x = ui_user_bubble_insets().body_pad_x;
            return std::max(8, ui_chat_frame_max_cols(stream_w, prefs) - body_pad_x * 2);
        }
        case UiUserMessageStyle::Messenger:
        case UiUserMessageStyle::Cornered:
            return std::max(8, ui_chat_frame_max_cols(stream_w, prefs) - 4);
        case UiUserMessageStyle::Terminal:
            return std::max(8, stream_w - 2);
        case UiUserMessageStyle::Log: {
            const std::string prefix = "[" + time_label_ + "] " + log_speaker_name() + "> ";
            return std::max(8, stream_w - tuinator::text_display_width(prefix));
        }
        case UiUserMessageStyle::Card:
            return std::max(8, std::min(48, stream_w - 4));
        default:
            return std::max(8, stream_w);
    }
}

std::vector<std::string> ChatMessage::content_lines() const {
    return wrap_text(text_, content_wrap_cols());
}

std::vector<std::string> ChatMessage::wrapped_lines() const {
    return wrap_text(text_, wrap_cols());
}

int ChatMessage::sticky_user_top_rows() const {
    switch (ui_user_chat_prefs().style) {
        case UiUserMessageStyle::Block:
            return ui_corners_flat() ? 1 : 2;
        case UiUserMessageStyle::Plain:
            return time_label_.empty() ? 1 : 2;
        case UiUserMessageStyle::Messenger:
            return 2;
        case UiUserMessageStyle::Header:
        case UiUserMessageStyle::Cornered:
            return 2;
        case UiUserMessageStyle::Terminal:
        case UiUserMessageStyle::Log:
        case UiUserMessageStyle::Card:
            return 1;
        case UiUserMessageStyle::Section:
            return 2;
        default:
            return 1;
    }
}

int ChatMessage::block_body_wrap_cols(int stream_w) const {
    const int body_pad_x = ui_user_bubble_insets().body_pad_x;
    return std::max(8, stream_w - body_pad_x * 2);
}

void ChatMessage::paint_user_block(tuinator::PaintContext& ctx, const tuinator::Rect& layout,
                                   int max_body_lines) const {
    const UserStreamBox stream = user_stream_box(layout);
    const int box_x = stream.x;
    const int box_w = stream.w;
    if (box_w <= 0 || layout.height <= 0) {
        return;
    }

    const auto all_lines = wrap_text(text_, block_body_wrap_cols(box_w));
    const int body_limit =
        max_body_lines > 0 ? std::min(max_body_lines, static_cast<int>(all_lines.size()))
                           : static_cast<int>(all_lines.size());
    if (body_limit <= 0 && text_.empty()) {
        return;
    }

    const std::string name = speaker_name();
    const bool default_you = author_label_.empty() || name == "You";

    tuinator::Style accent = ui_resolve_main_text(style_);
    accent.bold = true;
    accent.foreground_rgb = ui_user_bubble();
    const tuinator::Rgb panel_bg = ui_user_message_panel_bg();
    accent.background_rgb = panel_bg;

    tuinator::Style body = ui_resolve_main_text(style_);
    body.bold = false;
    body.foreground_rgb = ui_text();
    body.background_rgb = panel_bg;

    tuinator::Style muted = body;
    muted.foreground_rgb = ui_secondary_text();

    const tuinator::Rect panel{box_x, 0, box_w, layout.height};
    ui_paint_user_bubble_panel(ctx, panel, ui_quick_action_pill_bg());

    const UiUserBubbleInsets chrome = ui_user_bubble_insets();
    const int header_y = ui_corners_flat() ? 0 : 1;
    int y = header_y;
    if (y < layout.height) {
        const std::string who = (default_you ? std::string(kNfUser) + " " : "") + name;
        ctx.canvas.draw_text({box_x + chrome.pad_x, y}, who, accent);
        if (!time_label_.empty()) {
            const int time_w = tuinator::text_display_width(time_label_);
            const int time_x = std::max(box_x + chrome.pad_x, box_x + box_w - chrome.pad_x - time_w);
            ctx.canvas.draw_text({time_x, y}, time_label_, muted);
        }
        ++y;
    }
    for (int i = 0; i < body_limit; ++i) {
        if (y >= layout.height - chrome.bottom_inset) {
            break;
        }
        paint_plain_line(ctx, box_x + chrome.body_pad_x, y, box_w - chrome.body_pad_x * 2,
                         all_lines[static_cast<std::size_t>(i)], body);
        ++y;
    }
}

tuinator::Size ChatMessage::preferred_size_user() const {
    const auto lines = content_lines();
    const int body_rows = std::max(1, static_cast<int>(lines.size()));
    const UiUserBubbleInsets chrome = ui_user_bubble_insets();
    int height = 0;
    if (compact_summary_) {
        height = 2 + body_rows;
        return {wrap_cols(), height};
    }
    switch (ui_user_chat_prefs().style) {
        case UiUserMessageStyle::Block:
            height = sticky_user_top_rows() + body_rows + chrome.bottom_inset;
            break;
        case UiUserMessageStyle::Plain:
            height = 1 + body_rows;
            break;
        case UiUserMessageStyle::Messenger:
            height = 1 + 2 + body_rows + (time_label_.empty() ? 1 : 2);
            break;
        case UiUserMessageStyle::Header:
            height = 1 + body_rows;
            break;
        case UiUserMessageStyle::Cornered:
            height = 1 + body_rows + 1;
            break;
        case UiUserMessageStyle::Terminal:
            height = body_rows;
            break;
        case UiUserMessageStyle::Log:
            height = body_rows;
            break;
        case UiUserMessageStyle::Card: {
            const int inner_h = body_rows + 2;
            height = inner_h + (time_label_.empty() ? 0 : 0);
            break;
        }
        case UiUserMessageStyle::Section:
            height = 1 + body_rows + 1;
            break;
        default:
            height = body_rows;
            break;
    }
    const int w = chat_stream_row_preferred_width(bounds_.width > 0 ? bounds_.width : wrap_cols());
    return {w, height};
}

tuinator::Size ChatMessage::preferred_size() const {
    if (text_.empty()) {
        return {0, 0};
    }

    if (variant_ == Variant::UserPrompt) {
        return preferred_size_user();
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

void ChatMessage::paint_user_message(tuinator::PaintContext& ctx, const tuinator::Rect& layout, int max_body_lines) const {
    ui_fill_widget_bounds(ctx, *this, ui_main_bg());

    const UserStreamBox stream = user_stream_box(layout);
    const int box_x = stream.x;
    const int box_w = stream.w;
    if (box_w <= 0) {
        return;
    }

    auto lines = content_lines();
    if (max_body_lines > 0 && static_cast<int>(lines.size()) > max_body_lines) {
        lines.resize(static_cast<std::size_t>(max_body_lines));
    }
    const std::string name = speaker_name();
    const bool default_you = author_label_.empty() || name == "You";

    const ChatChromeStyles chrome = chat_chrome_styles(ChatMessageRole::User, style_);
    const tuinator::Style& accent = chrome.accent;
    const tuinator::Style& body = chrome.body;
    const tuinator::Style& muted = chrome.muted;
    const tuinator::Style& body_on_fill = chrome.body_on_fill;
    const tuinator::Style& muted_on_fill = chrome.muted_on_fill;

    int y = 0;

    if (compact_summary_) {
        std::string label = " " + log_speaker_name();
        if (!time_label_.empty()) {
            label += " · " + time_label_;
        }
        label += " ";
        if (y < layout.height) {
            paint_centered_rule(ctx, y, box_x, box_w, label, muted);
            ++y;
        }
        if (y < layout.height) {
            ++y;
        }
        for (const auto& line : lines) {
            if (y >= layout.height) {
                break;
            }
            paint_plain_line(ctx, box_x, y, box_w, line, body);
            ++y;
        }
        return;
    }

    const UiChatRolePrefs prefs = ui_user_chat_prefs();
    switch (prefs.style) {
        case UiUserMessageStyle::Block: {
            int max_line_w = 0;
            for (const auto& line : lines) {
                max_line_w = std::max(max_line_w, tuinator::text_display_width(line));
            }
            const ChatFrameLayout frame =
                chat_frame_layout(box_x, box_w, ui_user_bubble_insets().body_pad_x * 4, max_line_w, prefs);
            paint_user_block(ctx, {frame.frame_x, 0, frame.content_w, layout.height}, max_body_lines);
            return;
        }
        case UiUserMessageStyle::Plain: {
            std::string meta = name;
            if (!time_label_.empty()) {
                meta += " · " + time_label_;
            }
            if (y < layout.height) {
                ctx.canvas.draw_text({box_x, y}, meta, accent);
                ++y;
            }
            for (const auto& line : lines) {
                if (y >= layout.height) {
                    break;
                }
                paint_plain_line(ctx, box_x, y, box_w, line, body);
                ++y;
            }
            return;
        }
        case UiUserMessageStyle::Messenger: {
            const std::string title = (default_you ? std::string(kNfUser) + " " : "") + name;
            int max_line_w = 0;
            for (const auto& line : lines) {
                max_line_w = std::max(max_line_w, tuinator::text_display_width(line));
            }
            if (!time_label_.empty()) {
                max_line_w = std::max(max_line_w, tuinator::text_display_width(time_label_));
            }
            const int header_min = chat_header_strip_min_width(title, time_label_);
            const ChatFrameLayout frame = chat_frame_layout(box_x, box_w, header_min, max_line_w, prefs);
            const int block_top = y;
            const bool has_title = y < layout.height;
            if (has_title) {
                ++y;
            }
            const int body_rows = static_cast<int>(lines.size());
            const int bubble_h = std::max(3, body_rows + 2 + (time_label_.empty() ? 0 : 1));
            const int bubble_x = frame.frame_x;
            const int bubble_y = y;
            const int block_h = (bubble_y + bubble_h <= layout.height) ? (bubble_y + bubble_h - block_top) : 0;
            if (chat_message_fill_background(ChatMessageRole::User) && block_h > 0) {
                paint_chat_framed_panel_fill(ctx, frame, block_top, block_h, ChatMessageRole::User);
                if (has_title) {
                    const int label_w = tuinator::text_display_width(title);
                    const int label_x = frame.frame_x + std::max(0, frame.content_w - label_w);
                    ctx.canvas.draw_text({label_x, block_top}, title, chrome.accent_on_fill);
                }
            } else if (has_title) {
                const int label_w = tuinator::text_display_width(title);
                const int label_x = frame.frame_x + std::max(0, frame.content_w - label_w);
                ctx.canvas.draw_text({label_x, block_top}, title, accent);
            }
            const int bubble_w = frame.content_w;
            if (bubble_y + bubble_h <= layout.height) {
                const tuinator::Rect bubble{bubble_x, bubble_y, bubble_w, bubble_h};
                ui_paint_user_bubble_panel(ctx, bubble, ui_quick_action_pill_bg());
                int by = bubble_y + 1;
                for (const auto& line : lines) {
                    if (by >= bubble_y + bubble_h - (time_label_.empty() ? 1 : 2)) {
                        break;
                    }
                    paint_plain_line(ctx, bubble_x + 2, by, bubble_w - 4, line, body_on_fill);
                    ++by;
                }
                if (!time_label_.empty() && by < bubble_y + bubble_h) {
                    const int time_w = tuinator::text_display_width(time_label_);
                    const int time_x = bubble_x + std::max(2, bubble_w - 2 - time_w);
                    ctx.canvas.draw_text({time_x, by}, time_label_, muted_on_fill);
                }
            }
            return;
        }
        case UiUserMessageStyle::Header:
        case UiUserMessageStyle::Cornered: {
            const std::string title = (default_you ? std::string(kNfUser) + " " : "") + name;
            int max_body_w = 0;
            for (const auto& line : lines) {
                max_body_w = std::max(max_body_w, tuinator::text_display_width(line));
            }
            const int header_min = chat_header_strip_min_width(title, time_label_);
            const ChatFrameLayout frame = chat_frame_layout(box_x, box_w, header_min, max_body_w, prefs);
            const int body_rows = static_cast<int>(lines.size());
            const UiUserMessageStyle framed = prefs.style;
            paint_chat_role_framed_message(
                ctx, y, layout.height, frame, framed, title, time_label_, chrome, ChatMessageRole::User, body_rows,
                [&](tuinator::PaintContext& body_ctx, const ChatFrameLayout& body_frame, int body_top, int rows,
                    const ChatChromeStyles& styles) {
                    const tuinator::Style& text_style =
                        chat_message_fill_background(ChatMessageRole::User) ? styles.body_on_fill : styles.body;
                    if (framed == UiUserMessageStyle::Header) {
                        paint_chat_body_lines(body_ctx, body_top, layout.height, body_frame, lines, text_style);
                    } else {
                        int row_y = body_top;
                        for (const auto& line : lines) {
                            if (row_y >= layout.height) {
                                break;
                            }
                            paint_plain_line(body_ctx, body_frame.body_x, row_y, body_frame.body_w, line, text_style);
                            ++row_y;
                        }
                    }
                });
            return;
        }
        case UiUserMessageStyle::Terminal: {
            for (std::size_t i = 0; i < lines.size(); ++i) {
                if (y >= layout.height) {
                    break;
                }
                const std::string prefix = i == 0 ? "❯ " : "  ";
                tuinator::Style prompt = accent;
                ctx.canvas.draw_text({box_x, y}, prefix, prompt);
                const int prefix_w = tuinator::text_display_width(prefix);
                paint_plain_line(ctx, box_x + prefix_w, y, box_w - prefix_w, lines[i], body);
                ++y;
            }
            return;
        }
        case UiUserMessageStyle::Log: {
            const std::string prefix =
                "[" + (time_label_.empty() ? std::string("--:--") : time_label_) + "] " + log_speaker_name() + "> ";
            const int prefix_w = tuinator::text_display_width(prefix);
            for (std::size_t i = 0; i < lines.size(); ++i) {
                if (y >= layout.height) {
                    break;
                }
                if (i == 0) {
                    ctx.canvas.draw_text({box_x, y}, prefix, muted);
                    paint_plain_line(ctx, box_x + prefix_w, y, box_w - prefix_w, lines[i], body);
                } else {
                    paint_plain_line(ctx, box_x + prefix_w, y, box_w - prefix_w, lines[i], body);
                }
                ++y;
            }
            return;
        }
        case UiUserMessageStyle::Card: {
            int max_line_w = 0;
            for (const auto& line : lines) {
                max_line_w = std::max(max_line_w, tuinator::text_display_width(line));
            }
            const ChatFrameLayout frame = chat_frame_layout(box_x, box_w, 8, max_line_w, prefs);
            const int card_w = frame.content_w;
            const int card_h = std::max(3, static_cast<int>(lines.size()) + 2);
            const int card_x = frame.frame_x;
            const tuinator::Rect card{card_x, 0, card_w, std::min(card_h, layout.height)};
            ui_paint_user_bubble_panel(ctx, card, ui_quick_action_pill_bg());
            int cy = 1;
            for (const auto& line : lines) {
                if (cy >= card.height - 1) {
                    break;
                }
                paint_plain_line(ctx, card_x + 2, cy, card_w - 4, line, body_on_fill);
                ++cy;
            }
            if (!time_label_.empty()) {
                const int time_w = tuinator::text_display_width(time_label_);
                const int time_x = card_x + card_w + 2;
                const int time_y = card.y + std::max(0, (card.height - 1) / 2);
                if (time_x + time_w <= box_x + box_w) {
                    ctx.canvas.draw_text({time_x, time_y}, time_label_, muted);
                }
            }
            return;
        }
        // Section: full-width rule + centered label. Default is "you"; set_author_label() for
        // summaries ("summary"), agent roles, compaction markers, etc.
        case UiUserMessageStyle::Section: {
            std::string label = " " + log_speaker_name();
            if (!time_label_.empty()) {
                label += " · " + time_label_;
            }
            label += " ";
            if (y < layout.height) {
                paint_centered_rule(ctx, y, box_x, box_w, label, muted);
                ++y;
            }
            if (y < layout.height) {
                ++y;
            }
            for (const auto& line : lines) {
                if (y >= layout.height) {
                    break;
                }
                paint_plain_line(ctx, box_x, y, box_w, line, body);
                ++y;
            }
            return;
        }
        default:
            break;
    }
}

void ChatMessage::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0 || text_.empty()) {
        return;
    }

    if (variant_ == Variant::UserPrompt) {
        paint_user_message(ctx, bounds_);
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

int ChatMessage::sticky_preview_height_for_style(int max_body_lines) const {
    const auto lines = content_lines();
    if (lines.empty()) {
        return 0;
    }
    const int body_rows = std::min(max_body_lines, static_cast<int>(lines.size()));
    const UiUserBubbleInsets chrome = ui_user_bubble_insets();
    if (compact_summary_) {
        return 2 + body_rows;
    }
    switch (ui_user_chat_prefs().style) {
        case UiUserMessageStyle::Block:
            return sticky_user_top_rows() + body_rows + chrome.bottom_inset;
        case UiUserMessageStyle::Messenger:
            return 1 + 2 + body_rows + (time_label_.empty() ? 1 : 2);
        case UiUserMessageStyle::Cornered:
            return 1 + body_rows + 1;
        case UiUserMessageStyle::Header:
            return 1 + body_rows;
        case UiUserMessageStyle::Plain:
            return 1 + body_rows;
        default:
            return sticky_user_top_rows() + body_rows;
    }
}

int ChatMessage::sticky_preview_height(int max_body_lines) const {
    if (variant_ != Variant::UserPrompt || text_.empty() || max_body_lines <= 0) {
        return 0;
    }
    return sticky_preview_height_for_style(max_body_lines);
}

void ChatMessage::paint_sticky_preview(tuinator::PaintContext& ctx, int panel_width, int max_body_lines) const {
    if (variant_ != Variant::UserPrompt || text_.empty() || max_body_lines <= 0 || panel_width <= 0) {
        return;
    }

    const int height = sticky_preview_height(max_body_lines);
    if (height <= 0) {
        return;
    }

    ctx.with_clip({0, 0, panel_width, height}, [&](tuinator::PaintContext& clip_ctx) {
        tuinator::Style fill;
        fill.background_rgb = ui_main_bg();
        clip_ctx.canvas.fill_rect({0, 0, panel_width, height}, ' ', fill);
        paint_user_message(clip_ctx, {0, 0, panel_width, height}, max_body_lines);
    });
}
