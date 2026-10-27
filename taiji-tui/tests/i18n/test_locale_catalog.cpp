#include "locale_keys_used.gen.hpp"
#include "support/test_harness.hpp"
#include "ui_locale.hpp"

#include <string>

namespace {

std::string lookup(const std::string& locale_id, const std::string& module, const std::string& key) {
    ui_set_locale(locale_id);
    if (module == "shell") {
        return ui_shell(key);
    }
    return ui_tr(module, key);
}

} // namespace

TAIJI_TUI_TEST(all_referenced_locale_keys_resolve_in_english) {
    for (std::size_t i = 0; i < kTaijiLocaleKeyUsedCount; ++i) {
        const TaijiLocaleKeyRef& ref = kTaijiLocaleKeysUsed[i];
        const std::string value = lookup("en", ref.module, ref.key);
        TAIJI_TUI_CHECK(!value.empty());
    }
}

TAIJI_TUI_TEST(all_referenced_locale_keys_resolve_for_every_shipped_locale) {
    const std::vector<std::string> locales = ui_locale_ids();
    TAIJI_TUI_CHECK(!locales.empty());
    for (const std::string& locale_id : locales) {
        for (std::size_t i = 0; i < kTaijiLocaleKeyUsedCount; ++i) {
            const TaijiLocaleKeyRef& ref = kTaijiLocaleKeysUsed[i];
            const std::string value = lookup(locale_id, ref.module, ref.key);
            TAIJI_TUI_CHECK(!value.empty());
        }
    }
}

TAIJI_TUI_TEST(german_shell_strings_differ_from_english) {
    ui_set_locale("en");
    const std::string en_projects = ui_shell("sidebar.projects");
    const std::string en_settings = ui_tr("common", "app.sidebar_settings");

    ui_set_locale("de");
    const std::string de_projects = ui_shell("sidebar.projects");
    const std::string de_settings = ui_tr("common", "app.sidebar_settings");

    TAIJI_TUI_CHECK(!en_projects.empty() && !de_projects.empty());
    TAIJI_TUI_CHECK(en_projects != de_projects);
    TAIJI_TUI_CHECK(en_settings != de_settings);
    TAIJI_TUI_CHECK(de_settings.find("Einstellungen") != std::string::npos);
}
