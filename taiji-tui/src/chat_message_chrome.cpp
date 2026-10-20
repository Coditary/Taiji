#include "chat_message_chrome.hpp"

#include "ui_chat_column.hpp"

#include <tuinator/render/text.hpp>

#include <algorithm>

namespace {

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

} // namespace

int chat_header_strip_min_width(const std::string& title, const std::string& time_label) {
    const tuinator::BorderGlyphs glyphs = ui_frame_border_glyphs();
    const std::string left = glyphs.top_left + glyphs.horizontal + " ";
    const std::string right_cap = " " + glyphs.horizontal + glyphs.top_right;
    int width = tuinator::text_display_width(left) + tuinator::text_display_width(title);
    if (!time_label.empty()) {
        width += tuinator::text_display_width(" " + glyphs.horizontal + " ");
        width += tuinator::text_display_width(time_label);
    }
    width += tuinator::text_display_width(right_cap);
    return width;
}

ChatFrameLayout chat_frame_layout(int box_x, int box_w, int min_content_w, int body_text_w,
                                  const UiChatRolePrefs& prefs) {
    ChatFrameLayout out;
    if (box_w <= 0) {
        return out;
    }
    const int max_w = ui_chat_frame_max_cols(box_w, prefs);
    int content_w = std::max(min_content_w, body_text_w + 4);
    content_w = std::min(box_w, std::min(max_w, content_w));
    out.content_w = content_w;
    out.frame_x = ui_chat_frame_origin(box_x, box_w, content_w, prefs);
    out.body_x = out.frame_x + 2;
    out.body_w = std::max(1, content_w - 4);
    return out;
}

void paint_chat_header_strip(tuinator::PaintContext& ctx, int y, const ChatFrameLayout& frame, const std::string& title,
                             const std::string& time_label, const tuinator::Style& accent,
                             const tuinator::Style& muted) {
    const tuinator::BorderGlyphs glyphs = ui_frame_border_glyphs();
    const std::string left = glyphs.top_left + glyphs.horizontal + " ";
    const std::string before_time = time_label.empty() ? "" : " " + glyphs.horizontal + " ";
    const std::string right_cap = " " + glyphs.horizontal + glyphs.top_right;

    const int strip_width = frame.content_w;

    const int left_w = tuinator::text_display_width(left);
    const int title_w = tuinator::text_display_width(title);
    const int before_time_w = tuinator::text_display_width(before_time);
    const int time_w = time_label.empty() ? 0 : tuinator::text_display_width(time_label);
    const int right_w = tuinator::text_display_width(right_cap);

    const int fixed_w = left_w + title_w + before_time_w + time_w + right_w;
    int fill = std::max(0, strip_width - fixed_w);

    int cx = frame.frame_x;
    ctx.canvas.draw_text({cx, y}, left, accent);
    cx += left_w;
    ctx.canvas.draw_text({cx, y}, title, accent);
    cx += title_w;
    if (fill > 0) {
        paint_horizontal_run(ctx, y, cx, fill, accent);
        cx += fill;
    }
    if (!time_label.empty()) {
        ctx.canvas.draw_text({cx, y}, before_time, accent);
        cx += before_time_w;
        ctx.canvas.draw_text({cx, y}, time_label, muted);
        cx += time_w;
    }
    ctx.canvas.draw_text({cx, y}, right_cap, accent);
}

void paint_chat_cornered_bottom(tuinator::PaintContext& ctx, int y, const ChatFrameLayout& frame,
                                const tuinator::Style& accent) {
    if (frame.content_w <= 0) {
        return;
    }
    const tuinator::BorderGlyphs glyphs = ui_frame_border_glyphs();
    const std::string left_corner = glyphs.bottom_left;
    const std::string right_corner = glyphs.bottom_right;
    const int left_w = tuinator::text_display_width(left_corner);
    const int right_w = tuinator::text_display_width(right_corner);
    if (frame.content_w <= left_w + right_w) {
        paint_plain_line(ctx, frame.frame_x, y, frame.content_w, left_corner + right_corner, accent);
        return;
    }
    ctx.canvas.draw_text({frame.frame_x, y}, left_corner, accent);
    const int mid_w = frame.content_w - left_w - right_w;
    paint_horizontal_run(ctx, y, frame.frame_x + left_w, mid_w, accent);
    ctx.canvas.draw_text({frame.frame_x + frame.content_w - right_w, y}, right_corner, accent);
}

void paint_chat_framed_body_fill(tuinator::PaintContext& ctx, const ChatFrameLayout& frame, int y, int row_count) {
    if (!ui_message_fill_background() || row_count <= 0 || frame.content_w <= 0) {
        return;
    }
    tuinator::Style fill;
    fill.background_rgb = ui_message_panel_bg();
    ctx.canvas.fill_rect({frame.frame_x, y, frame.content_w, row_count}, ' ', fill);
}

void paint_chat_body_lines(tuinator::PaintContext& ctx, int y, int max_y, const ChatFrameLayout& frame,
                           const std::vector<std::string>& lines, const tuinator::Style& body) {
    for (const auto& line : lines) {
        if (y >= max_y) {
            break;
        }
        paint_plain_line(ctx, frame.body_x, y, frame.body_w, line, body);
        ++y;
    }
}
