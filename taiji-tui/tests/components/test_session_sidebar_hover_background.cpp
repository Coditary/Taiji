#include <tuinator/testing/app_driver.hpp>
#include <tuinator/testing/cell_styles.hpp>
#include <tuinator/testing/widget_query.hpp>

#include "scenes/scenes.hpp"
#include "support/test_harness.hpp"
#include "ui_test_ids.hpp"

/// Regression guard: hovered session rows paint the relative-time suffix with the same
/// row background as the session title (see session_sidebar.cpp suffix_style).
/// Intentionally strict — fails until hover fill and suffix text share one background.
TAIJI_TUI_TEST(session_sidebar_hovered_row_background_is_uniform_across_timestamp) {
    tuinator::testing::AppDriver driver({40, 16}, taiji::ui::test_scenes::make_session_sidebar());
    driver.present();

    const std::string row_selector = taiji::ui::test_id::session_row("sess-beta");
    tuinator::Widget* row = tuinator::testing::find_by_selector(driver.root(), row_selector);
    TAIJI_TUI_CHECK(row != nullptr);

    driver.hover(row_selector);
    driver.present();

    const std::string snapshot = driver.snapshot_text();
    TAIJI_TUI_CHECK(snapshot.find("Beta chat") != std::string::npos);

    const tuinator::testing::RegionStyleAnalysis analysis = driver.analyze_selector(row_selector);
    TAIJI_TUI_CHECK(analysis.cells_sampled > 0);
    TAIJI_TUI_CHECK(analysis.left_background.has_value());
    TAIJI_TUI_CHECK(analysis.right_background.has_value());
    TAIJI_TUI_CHECK(analysis.uniform_background);
    TAIJI_TUI_CHECK(*analysis.left_background == *analysis.right_background);
}
