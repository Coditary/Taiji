#include <tuinator/testing/app_driver.hpp>

#include "scenes/scenes.hpp"
#include "support/test_harness.hpp"
#include "ui_locale.hpp"

TAIJI_TUI_TEST(session_sidebar_renders_german_section_titles) {
    ui_set_locale("de");
    tuinator::testing::AppDriver driver({32, 14}, taiji::ui::test_scenes::make_session_sidebar());
    driver.present();

    const std::string snapshot = driver.snapshot_text();
    TAIJI_TUI_CHECK(snapshot.find("PROJEKTE") != std::string::npos);
    TAIJI_TUI_CHECK(snapshot.find("Einstellungen") != std::string::npos);
}

TAIJI_TUI_TEST(session_sidebar_english_chrome_uses_catalog_not_hardcoded_projects) {
    ui_set_locale("en");
    tuinator::testing::AppDriver driver({32, 14}, taiji::ui::test_scenes::make_session_sidebar());
    driver.present();

    const std::string snapshot = driver.snapshot_text();
    TAIJI_TUI_CHECK(snapshot.find(ui_shell("sidebar.projects")) != std::string::npos);
    TAIJI_TUI_CHECK(snapshot.find("PROJECTS") != std::string::npos);
}
