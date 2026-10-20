#pragma once

#include <cstddef>

namespace ui_bundled_themes {

struct BundledTheme {
    const char* id;
    const char* text;
};

extern const BundledTheme kBundledThemes[];
extern const std::size_t kBundledThemeCount;

const char* theme_text_for_id(const char* theme_id);

} // namespace ui_bundled_themes
