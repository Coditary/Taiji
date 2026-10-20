#pragma once

#include <tuinator/tuinator.hpp>

#include <string>

/// Returns true when `key` matches a shortcut string such as "Ctrl+T".
bool ui_keybinding_matches(const std::string& shortcut, const tuinator::KeyPress& key);

/// Default shortcut when config leaves the binding empty.
std::string ui_keybinding_default_open_theme_picker();
