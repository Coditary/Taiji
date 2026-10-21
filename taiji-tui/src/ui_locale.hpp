#pragma once

#include <string>
#include <vector>

/// Active UI locale id (e.g. en, de). Falls back to English for missing keys.
void ui_set_locale(const std::string& locale_id);
const std::string& ui_locale();
std::vector<std::string> ui_locale_ids();
/// Maps unknown/disabled ids (e.g. removed bn) to "en".
std::string ui_normalize_locale_id(const std::string& locale_id);

/// Translate within a module (common, settings). Key uses dotted paths from JSON.
std::string ui_tr(const std::string& module, const std::string& key);
/// Language name for the settings picker (uses active UI locale, safe for limited fonts).
std::string ui_locale_display_name(const std::string& locale_id);
inline std::string ui_tr(const std::string& key) { return ui_tr("common", key); }

std::string ui_settings_row_label(const std::string& enum_key, const std::string& fallback);
std::string ui_settings_row_hint(const std::string& enum_key, const std::string& fallback);
/// Localized label for a settings enum value (`options.<group>.<value_id>` in settings.json).
std::string ui_settings_option(const std::string& group, const std::string& value_id, const std::string& fallback);

/// Replace `{name}` placeholders in a translated shell string.
std::string ui_format(const std::string& text, const std::string& placeholder, const std::string& value);
std::string ui_shell(const std::string& key);
std::string ui_default_footer_shortcuts();
