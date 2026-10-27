#include "chat_message_chrome.hpp"

#include "ui_anim.hpp"
#include "ui_chat_column.hpp"

#include <tuinator/render/text.hpp>

#include <algorithm>

namespace {

tuinator::Style chat_border_style(tuinator::Style style, ChatMessageRole role) {
    style.background_rgb =
        chat_message_fill_background(role) ? chat_message_panel_bg(role) : ui_main_bg();
    return style;
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

bool chat_message_fill_background(ChatMessageRole role) {
    return role == ChatMessageRole::User ? ui_user_message_fill_background() : ui_assistant_message_fill_background();
}

tuinator::Rgb chat_message_panel_bg(ChatMessageRole role) {
    return role == ChatMessageRole::User ? ui_user_message_panel_bg() : ui_assistant_message_panel_bg();
}

ChatChromeStyles chat_chrome_styles(ChatMessageRole role, const tuinator::Style& user_base_style) {
    ChatChromeStyles out;
    const tuinator::Rgb panel_bg = chat_message_panel_bg(role);

    if (role == ChatMessageRole::User) {
        out.accent = ui_resolve_main_text(user_base_style);
        out.accent.bold = true;
        out.accent.foreground_rgb = ui_user_bubble();
    } else {
        out.accent = ui_thinking_style();
        out.accent.foreground_rgb = ui_assistant_chrome_accent_rgb();
        out.body = ui_resolve_main_text(out.accent);
        out.body.bold = false;
        if (ui_narrative_presentation()) {
            out.body.foreground_rgb = ui_fade_toward(ui_text(), ui_secondary_text(), 0.12);
        } else {
            out.body.foreground_rgb = ui_text();
        }
    }

    if (role == ChatMessageRole::User) {
        out.body = ui_resolve_main_text(user_base_style);
        out.body.bold = false;
        out.body.foreground_rgb = ui_text();
    }

    const tuinator::Rgb chrome_bg = chat_message_fill_background(role) ? panel_bg : ui_main_bg();
    out.accent.background_rgb = chrome_bg;
    out.body.background_rgb = chrome_bg;

    out.muted = out.body;
    out.muted.foreground_rgb = ui_secondary_text();
    out.muted.background_rgb = chrome_bg;

    out.body_on_fill = out.body;
    out.body_on_fill.background_rgb = panel_bg;
    out.accent_on_fill = out.accent;
    out.accent_on_fill.background_rgb = panel_bg;
    out.muted_on_fill = out.muted;
    out.muted_on_fill.background_rgb = panel_bg;

    return out;
}

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
                             const tuinator::Style& muted, ChatMessageRole role) {
    if (chat_message_fill_background(role) && frame.content_w > 0) {
        paint_chat_framed_panel_fill(ctx, frame, y, 1, role);
    }
    const tuinator::Style border = chat_border_style(accent, role);
    const tuinator::Style time_style = chat_border_style(muted, role);
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
    const tuinator::Style title_style = chat_border_style(accent, role);
    ctx.canvas.draw_text({cx, y}, left, border);
    cx += left_w;
    ctx.canvas.draw_text({cx, y}, title, title_style);
    cx += title_w;
    if (fill > 0) {
        paint_horizontal_run(ctx, y, cx, fill, border);
        cx += fill;
    }
    if (!time_label.empty()) {
        ctx.canvas.draw_text({cx, y}, before_time, border);
        cx += before_time_w;
        ctx.canvas.draw_text({cx, y}, time_label, time_style);
        cx += time_w;
    }
    ctx.canvas.draw_text({cx, y}, right_cap, border);
}

void paint_chat_cornered_bottom(tuinator::PaintContext& ctx, int y, const ChatFrameLayout& frame,
                                const tuinator::Style& accent, ChatMessageRole role) {
    if (frame.content_w <= 0) {
        return;
    }
    const tuinator::Style border = chat_border_style(accent, role);
    const tuinator::BorderGlyphs glyphs = ui_frame_border_glyphs();
    const std::string left_corner = glyphs.bottom_left;
    const std::string right_corner = glyphs.bottom_right;
    const int left_w = tuinator::text_display_width(left_corner);
    const int right_w = tuinator::text_display_width(right_corner);
    if (frame.content_w <= left_w + right_w) {
        paint_plain_line(ctx, frame.frame_x, y, frame.content_w, left_corner + right_corner, border);
        return;
    }
    ctx.canvas.draw_text({frame.frame_x, y}, left_corner, border);
    const int mid_w = frame.content_w - left_w - right_w;
    paint_horizontal_run(ctx, y, frame.frame_x + left_w, mid_w, border);
    ctx.canvas.draw_text({frame.frame_x + frame.content_w - right_w, y}, right_corner, border);
}

int paint_chat_role_framed_message(tuinator::PaintContext& ctx, int y, int max_y, const ChatFrameLayout& frame,
                                 UiUserMessageStyle style, const std::string& title, const std::string& time_label,
                                 const ChatChromeStyles& styles, ChatMessageRole role, int body_rows,
                                 const ChatFramedBodyPainter& paint_body) {
    if (body_rows < 1) {
        body_rows = 1;
    }

    switch (style) {
        case UiUserMessageStyle::Header: {
            const int panel_top = y;
            const bool has_header = y < max_y;
            if (has_header) {
                ++y;
            }
            const int body_top = y;
            const int panel_rows = (has_header ? 1 : 0) + body_rows;
            paint_chat_framed_panel_fill(ctx, frame, panel_top, panel_rows, role);
            if (has_header) {
                const tuinator::Style& hdr =
                    chat_message_fill_background(role) ? styles.accent_on_fill : styles.accent;
                const tuinator::Style& mut =
                    chat_message_fill_background(role) ? styles.muted_on_fill : styles.muted;
                paint_chat_header_strip(ctx, panel_top, frame, title, time_label, hdr, mut, role);
            }
            if (paint_body) {
                paint_body(ctx, frame, body_top, body_rows, styles);
            }
            return body_top + body_rows;
        }
        case UiUserMessageStyle::Cornered: {
            const int panel_top = y;
            const bool has_header = y < max_y;
            if (has_header) {
                ++y;
            }
            const int body_top = y;
            const bool has_footer = body_top + body_rows < max_y;
            const int panel_rows = (has_header ? 1 : 0) + body_rows + (has_footer ? 1 : 0);
            paint_chat_framed_panel_fill(ctx, frame, panel_top, panel_rows, role);
            if (has_header) {
                const tuinator::Style& hdr =
                    chat_message_fill_background(role) ? styles.accent_on_fill : styles.accent;
                const tuinator::Style& mut =
                    chat_message_fill_background(role) ? styles.muted_on_fill : styles.muted;
                paint_chat_header_strip(ctx, panel_top, frame, title, time_label, hdr, mut, role);
            }
            if (paint_body) {
                paint_body(ctx, frame, body_top, body_rows, styles);
            }
            y = body_top + body_rows;
            if (has_footer) {
                const tuinator::Style& foot =
                    chat_message_fill_background(role) ? styles.accent_on_fill : styles.accent;
                paint_chat_cornered_bottom(ctx, y, frame, foot, role);
                ++y;
            }
            return y;
        }
        default:
            return y;
    }
}

void paint_chat_framed_panel_fill(tuinator::PaintContext& ctx, const ChatFrameLayout& frame, int y, int row_count,
                                  ChatMessageRole role) {
    if (!chat_message_fill_background(role) || row_count <= 0 || frame.content_w <= 0) {
        return;
    }
    tuinator::Style fill;
    fill.background_rgb = chat_message_panel_bg(role);
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
