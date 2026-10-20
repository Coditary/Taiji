#include "ui_keybindings.hpp"

#include <cctype>

namespace {

char normalize_key(char ch) { return static_cast<char>(std::tolower(static_cast<unsigned char>(ch))); }

bool match_shortcut(const std::string& shortcut, const tuinator::KeyPress& key) {
    if (shortcut.empty()) {
        return false;
    }

    bool wants_ctrl = false;
    bool wants_alt = false;
    char target = '\0';

    std::string token;
    for (std::size_t i = 0; i <= shortcut.size(); ++i) {
        if (i == shortcut.size() || shortcut[i] == '+') {
            if (token.empty()) {
                continue;
            }

            if (token == "Ctrl" || token == "Control") {
                wants_ctrl = true;
            } else if (token == "Alt") {
                wants_alt = true;
            } else if (token.size() == 1) {
                target = normalize_key(token[0]);
            } else if (token == "Enter") {
                return key.key == tuinator::Key::Enter && !wants_ctrl && !wants_alt;
            } else if (token == "Esc" || token == "Escape") {
                return key.key == tuinator::Key::Escape && !wants_ctrl && !wants_alt;
            }
            token.clear();
            continue;
        }
        token.push_back(shortcut[i]);
    }

    if (target == '\0') {
        return false;
    }

    if (wants_ctrl != key.ctrl || wants_alt != key.alt) {
        return false;
    }

    if (key.character != '\0') {
        return normalize_key(key.character) == target;
    }

    return false;
}

} // namespace

bool ui_keybinding_matches(const std::string& shortcut, const tuinator::KeyPress& key) {
    return match_shortcut(shortcut, key);
}

std::string ui_keybinding_default_open_theme_picker() {
    return "Ctrl+T";
}
