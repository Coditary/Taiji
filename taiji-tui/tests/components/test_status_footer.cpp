#include <tuinator/testing/app_driver.hpp>
#include <tuinator/testing/widget_query.hpp>

#include "status_footer.hpp"
#include "support/test_harness.hpp"
#include "ui_test_ids.hpp"

TAIJI_TUI_TEST(status_footer_renders_path_and_status) {
    tuinator::Style path_style;
    tuinator::Style status_style;
    auto footer = std::make_unique<StatusFooter>(path_style, status_style);
    footer->set_path("/workspace/taiji");
    footer->set_status("Tab: focus");
    footer->set_vim_mode_badge(true, UiInputMode::Insert);

    tuinator::testing::AppDriver driver({120, 3}, std::move(footer));
    driver.present();

    TAIJI_TUI_CHECK(tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kStatusFooter) != nullptr);
    const std::string snapshot = driver.snapshot_text();
    TAIJI_TUI_CHECK(snapshot.find("INSERT") != std::string::npos);
}
