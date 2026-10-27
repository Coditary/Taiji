#include <tuinator/testing/app_driver.hpp>
#include <tuinator/testing/widget_query.hpp>

#include "scenes/scenes.hpp"
#include "session_sidebar.hpp"
#include "support/test_harness.hpp"
#include "ui_test_ids.hpp"

TAIJI_TUI_TEST(session_sidebar_lists_fixture_sessions) {
    tuinator::testing::AppDriver driver({32, 14}, taiji::ui::test_scenes::make_session_sidebar());
    driver.present();

    const std::string snapshot = driver.snapshot_text();
    TAIJI_TUI_CHECK(snapshot.find("Alpha chat") != std::string::npos);
    TAIJI_TUI_CHECK(snapshot.find("Beta chat") != std::string::npos);
}

TAIJI_TUI_TEST(session_sidebar_exposes_row_and_action_selectors) {
    tuinator::testing::AppDriver driver({32, 14}, taiji::ui::test_scenes::make_session_sidebar());
    driver.present();

    TAIJI_TUI_CHECK(tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::session_row("sess-alpha")) !=
                    nullptr);
    TAIJI_TUI_CHECK(tuinator::testing::find_by_selector(
                        driver.root(), taiji::ui::test_id::project_new_button("/home/demo/proj")) != nullptr);
    TAIJI_TUI_CHECK(tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kSessionSidebarSettings) !=
                    nullptr);
}

TAIJI_TUI_TEST(session_sidebar_lists_empty_folder_group) {
    const char* json = R"json({
  "current_session": "",
  "folders": [{
    "root": "/home/Leodoras",
    "name": "Leodoras",
    "sessions": []
  }],
  "projects": []
})json";
    auto nav = taiji::ui::test_scenes::make_session_sidebar();
    SessionSidebar* sidebar = nav.get();
    tuinator::testing::AppDriver driver({36, 12}, std::move(nav));
    sidebar->apply_list_json(json);
    driver.present();

    const std::string snapshot = driver.snapshot_text();
    TAIJI_TUI_CHECK(snapshot.find("Leodoras") != std::string::npos);
    TAIJI_TUI_CHECK(snapshot.find("(no chats yet)") != std::string::npos);
}

TAIJI_TUI_TEST(session_sidebar_active_row_is_style_probeable) {
    tuinator::testing::AppDriver driver({32, 14}, taiji::ui::test_scenes::make_session_sidebar());
    driver.present();

    const tuinator::testing::RegionStyleAnalysis analysis =
        driver.analyze_selector(taiji::ui::test_id::session_row("sess-alpha"));
    TAIJI_TUI_CHECK(analysis.left_background.has_value());
}
