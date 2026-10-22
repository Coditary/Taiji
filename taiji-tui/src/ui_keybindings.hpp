#pragma once

#include <tuinator/tuinator.hpp>

#include <string>

/// Returns true when `key` matches a shortcut string such as "Ctrl+T".
bool ui_keybinding_matches(const std::string& shortcut, const tuinator::KeyPress& key);

/// Default shortcut when config leaves the binding empty.
std::string ui_keybinding_default_open_theme_picker();

bool ui_key_ctrl_navigation(const tuinator::KeyPress& key);
bool ui_key_theme_picker_list_prev(const tuinator::KeyPress& key);
bool ui_key_theme_picker_list_next(const tuinator::KeyPress& key);
