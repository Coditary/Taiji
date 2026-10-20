#pragma once

#include "ui_theme.hpp"

#include <tuinator/tuinator.hpp>

#include <string>
#include <vector>

struct ChatFrameLayout {
    int frame_x = 0;
    int content_w = 0;
    int body_x = 0;
    int body_w = 0;
};

/// Layout width/position for a framed chat message (user or assistant).
ChatFrameLayout chat_frame_layout(int box_x, int box_w, int min_content_w, int body_text_w, const UiChatRolePrefs& prefs);

void paint_chat_header_strip(tuinator::PaintContext& ctx, int y, const ChatFrameLayout& frame, const std::string& title,
                             const std::string& time_label, const tuinator::Style& accent, const tuinator::Style& muted);

void paint_chat_cornered_bottom(tuinator::PaintContext& ctx, int y, const ChatFrameLayout& frame,
                                const tuinator::Style& accent);

void paint_chat_body_lines(tuinator::PaintContext& ctx, int y, int max_y, const ChatFrameLayout& frame,
                           const std::vector<std::string>& lines, const tuinator::Style& body);

int chat_header_strip_min_width(const std::string& title, const std::string& time_label);

/// Solid panel fill for the body rows between top strip and bottom corner (when enabled in settings).
void paint_chat_framed_body_fill(tuinator::PaintContext& ctx, const ChatFrameLayout& frame, int y, int row_count);
