#include "ui_themes_bundled.hpp"

#include <cstring>

namespace ui_bundled_themes {

const char* theme_text_for_id(const char* theme_id) {
    if (theme_id == nullptr) {
        return nullptr;
    }
    for (std::size_t i = 0; i < kBundledThemeCount; ++i) {
        if (std::strcmp(kBundledThemes[i].id, theme_id) == 0) {
            return kBundledThemes[i].text;
        }
    }
    return nullptr;
}

} // namespace ui_bundled_themes
