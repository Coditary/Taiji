#pragma once

#include <tuinator/tuinator.hpp>

#include <string>

enum class UiKeybindingStyle { Standard, VimModal };

enum class UiInputMode { Insert, Normal };

enum class UiFocusPane { Prompt, Chat, Projects, Context };

UiKeybindingStyle ui_parse_keybinding_style(const std::string& value);
std::string ui_keybinding_style_id(UiKeybindingStyle style);
std::string ui_keybinding_style_label(UiKeybindingStyle style);

void ui_set_navigation_chrome(bool vim_modal, UiFocusPane pane, UiInputMode mode);
bool ui_vim_modal_chrome_enabled();
UiFocusPane ui_focus_pane();
UiInputMode ui_input_mode();
void ui_paint_vim_focus_accent(tuinator::PaintContext& ctx, UiFocusPane pane, int height);

bool ui_key_ctrl_char(const tuinator::KeyPress& key, char ch, bool allow_shift = false);
bool ui_key_plain_char(const tuinator::KeyPress& key, char ch);

std::string ui_footer_shortcuts_for_state(bool vim_modal, UiInputMode mode);
