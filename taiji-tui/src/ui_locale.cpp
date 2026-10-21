#include "ui_locale.hpp"

#include "ui_locales_embed.hpp"

#include <unordered_map>

namespace {

using Catalog = std::unordered_map<std::string, std::string>;

constexpr const char* kLocalePlaceholder = "...";

bool is_usable_locale_value(const std::string& value) {
    return !value.empty() && value != kLocalePlaceholder;
}

void ensure_loaded();
std::string catalog_key(const std::string& module, const std::string& key);

std::unordered_map<std::string, Catalog> g_by_locale;
std::string g_active_locale = "en";
bool g_loaded = false;

std::string catalog_key(const std::string& module, const std::string& key) {
    return module + "\x1f" + key;
}

void ensure_loaded() {
    if (g_loaded) {
        return;
    }
    for (std::size_t i = 0; i < taiji_locales_embed::kStringCount; ++i) {
        const taiji_locales_embed::LocaleString& entry = taiji_locales_embed::kStrings[i];
        if (entry.locale == nullptr || entry.module == nullptr || entry.key == nullptr || entry.value == nullptr) {
            continue;
        }
        g_by_locale[entry.locale][catalog_key(entry.module, entry.key)] = entry.value;
    }
    g_loaded = true;
}

std::string lookup_raw(const std::string& locale_id, const std::string& module, const std::string& key) {
    ensure_loaded();
    const auto locale_it = g_by_locale.find(locale_id);
    if (locale_it == g_by_locale.end()) {
        return {};
    }
    const auto value_it = locale_it->second.find(catalog_key(module, key));
    if (value_it == locale_it->second.end()) {
        return {};
    }
    return value_it->second;
}

const std::string& lookup(const std::string& locale_id, const std::string& module, const std::string& key) {
    ensure_loaded();
    static const std::string kEmpty;
    const auto locale_it = g_by_locale.find(locale_id);
    if (locale_it != g_by_locale.end()) {
        const auto value_it = locale_it->second.find(catalog_key(module, key));
        if (value_it != locale_it->second.end() && is_usable_locale_value(value_it->second)) {
            return value_it->second;
        }
    }
    if (locale_id != "en") {
        return lookup("en", module, key);
    }
    return kEmpty;
}

} // namespace

void ui_set_locale(const std::string& locale_id) {
    g_active_locale = ui_normalize_locale_id(locale_id);
}

const std::string& ui_locale() {
    return g_active_locale;
}

std::vector<std::string> ui_locale_ids() {
    ensure_loaded();
    std::vector<std::string> out;
    out.reserve(taiji_locales_embed::kLocaleIdCount);
    for (std::size_t i = 0; i < taiji_locales_embed::kLocaleIdCount; ++i) {
        out.emplace_back(taiji_locales_embed::kLocaleIds[i]);
    }
    return out;
}

std::string ui_normalize_locale_id(const std::string& locale_id) {
    ensure_loaded();
    if (locale_id.empty()) {
        return "en";
    }
    for (std::size_t i = 0; i < taiji_locales_embed::kLocaleIdCount; ++i) {
        if (locale_id == taiji_locales_embed::kLocaleIds[i]) {
            return locale_id;
        }
    }
    return "en";
}

std::string ui_tr(const std::string& module, const std::string& key) {
    return lookup(g_active_locale, module, key);
}

std::string ui_locale_display_name(const std::string& locale_id) {
    if (locale_id.empty()) {
        return {};
    }
    const std::string key = "locale_names." + locale_id;
    // Autonym: name in the language's own script (e.g. 日本語, Deutsch).
    std::string label = lookup_raw(locale_id, "common", key);
    if (!is_usable_locale_value(label)) {
        label = lookup_raw("en", "common", key);
    }
    if (!is_usable_locale_value(label)) {
        label = lookup_raw(g_active_locale, "common", key);
    }
    return label.empty() ? locale_id : label;
}

std::string ui_settings_row_label(const std::string& enum_key, const std::string& fallback) {
    if (enum_key.empty()) {
        return fallback;
    }
    const std::string translated = ui_tr("settings", "rows." + enum_key + ".label");
    return translated.empty() ? fallback : translated;
}

std::string ui_format(const std::string& text, const std::string& placeholder, const std::string& value) {
    if (text.empty() || placeholder.empty()) {
        return text;
    }
    const std::string token = "{" + placeholder + "}";
    std::string out = text;
    for (std::size_t pos = 0; (pos = out.find(token, pos)) != std::string::npos;) {
        out.replace(pos, token.size(), value);
        pos += value.size();
    }
    return out;
}

std::string ui_shell(const std::string& key) {
    return ui_tr("shell", key);
}

std::string ui_default_footer_shortcuts() {
    const std::string translated = ui_shell("footer.shortcuts_default");
    if (!translated.empty()) {
        return translated;
    }
    return "Tab: focus · Shift+Enter: newline · Enter: send · Esc: stop";
}

std::string ui_settings_row_hint(const std::string& enum_key, const std::string& fallback) {
    if (enum_key.empty()) {
        return fallback;
    }
    const std::string translated = ui_tr("settings", "rows." + enum_key + ".hint");
    return translated.empty() ? fallback : translated;
}

std::string ui_settings_option(const std::string& group, const std::string& value_id, const std::string& fallback) {
    if (group.empty() || value_id.empty()) {
        return fallback;
    }
    const std::string translated = ui_tr("settings", "options." + group + "." + value_id);
    return translated.empty() ? fallback : translated;
}
