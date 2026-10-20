#pragma once

#include "ui_theme.hpp"

#include <tuinator/tuinator.hpp>

#include <functional>
#include <string>
#include <vector>

struct ChatFrameLayout {
    int frame_x = 0;
    int content_w = 0;
    int body_x = 0;
    int body_w = 0;
};

enum class ChatMessageRole { User, Assistant };

struct ChatChromeStyles {
    tuinator::Style accent{};
    tuinator::Style muted{};
    tuinator::Style body{};
    tuinator::Style body_on_fill{};
    tuinator::Style accent_on_fill{};
    tuinator::Style muted_on_fill{};
};

/// Border/title colors for user vs assistant bubbles (chrome uses main background, not panel fill).
ChatChromeStyles chat_chrome_styles(ChatMessageRole role, const tuinator::Style& user_base_style = {});

using ChatFramedBodyPainter = std::function<void(tuinator::PaintContext& ctx, const ChatFrameLayout& frame,
                                                   int body_top, int body_rows, const ChatChromeStyles& styles)>;

/// Shared header/cornered frame; invokes paint_body for the content rows between top strip and bottom corner.
int paint_chat_role_framed_message(tuinator::PaintContext& ctx, int y, int max_y, const ChatFrameLayout& frame,
                                     UiUserMessageStyle style, const std::string& title, const std::string& time_label,
                                     const ChatChromeStyles& styles, ChatMessageRole role, int body_rows,
                                     const ChatFramedBodyPainter& paint_body);

/// Layout width/position for a framed chat message (user or assistant).
ChatFrameLayout chat_frame_layout(int box_x, int box_w, int min_content_w, int body_text_w, const UiChatRolePrefs& prefs);

void paint_chat_header_strip(tuinator::PaintContext& ctx, int y, const ChatFrameLayout& frame, const std::string& title,
                             const std::string& time_label, const tuinator::Style& accent, const tuinator::Style& muted,
                             ChatMessageRole role);

void paint_chat_cornered_bottom(tuinator::PaintContext& ctx, int y, const ChatFrameLayout& frame,
                                const tuinator::Style& accent, ChatMessageRole role);

void paint_chat_body_lines(tuinator::PaintContext& ctx, int y, int max_y, const ChatFrameLayout& frame,
                           const std::vector<std::string>& lines, const tuinator::Style& body);

int chat_header_strip_min_width(const std::string& title, const std::string& time_label);

bool chat_message_fill_background(ChatMessageRole role);
tuinator::Rgb chat_message_panel_bg(ChatMessageRole role);

/// Solid panel fill for framed chrome (header, body, bottom corner) when fill is enabled for the role.
void paint_chat_framed_panel_fill(tuinator::PaintContext& ctx, const ChatFrameLayout& frame, int y, int row_count,
                                  ChatMessageRole role);
