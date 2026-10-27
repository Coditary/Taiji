#include <tuinator/testing/app_driver.hpp>
#include <tuinator/testing/widget_query.hpp>

#include "scenes/scenes.hpp"
#include "support/test_harness.hpp"
#include "ui_test_ids.hpp"

TAIJI_TUI_TEST(integration_sidebar_column_contains_sidebar_and_footer) {
    tuinator::testing::AppDriver driver({32, 16}, taiji::ui::test_scenes::make_sidebar_with_footer());
    driver.present();

    TAIJI_TUI_CHECK(tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kSessionSidebar) !=
                    nullptr);
    TAIJI_TUI_CHECK(tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kStatusFooter) !=
                    nullptr);

    const std::string snapshot = driver.snapshot_text();
    TAIJI_TUI_CHECK(snapshot.find("Demo Project") != std::string::npos);
    TAIJI_TUI_CHECK(snapshot.find("Ready") != std::string::npos);
}
