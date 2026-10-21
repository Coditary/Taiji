#pragma once

#include <cstddef>

namespace taiji_locales_embed {

struct LocaleString {
    const char* locale;
    const char* module;
    const char* key;
    const char* value;
};

extern const LocaleString kStrings[];
extern const std::size_t kStringCount;
extern const char* const kLocaleIds[];
extern const std::size_t kLocaleIdCount;

} // namespace taiji_locales_embed
