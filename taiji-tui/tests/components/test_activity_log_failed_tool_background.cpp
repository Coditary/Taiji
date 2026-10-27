#include <tuinator/testing/app_driver.hpp>
#include <tuinator/testing/cell_styles.hpp>

#include "activity_log.hpp"
#include "support/test_harness.hpp"
#include "ui_chat_spacing.hpp"
#include "ui_test_ids.hpp"
#include "ui_theme.hpp"

namespace {

constexpr tuinator::Rgb kFailedToolPanelBg{55, 38, 42};
constexpr tuinator::Rgb kFailedToolBadgeBg{78, 48, 48};
constexpr tuinator::Rgb kFailedToolLabelFg{250, 200, 200};

std::unique_ptr<ActivityLog> make_failed_remember_log() {
    tuinator::Style muted;
    muted.foreground_rgb = tuinator::rgb(140, 145, 160);
    tuinator::Style active;
    active.foreground_rgb = tuinator::rgb(220, 225, 240);
    auto log = std::make_unique<ActivityLog>(active, muted);
    ui_set_tool_expand_display(true, true, "pretty", "text");
    log->set_wrap_width(64);
    log->start_tool("memory_remember", R"({"kind":"preference","text":"Leonard"})", false, false, "", "");
    log->finish_tool("memory_remember", "error: recall_text is required", "", 0);
    return log;
}

void click_activity_log_header(tuinator::testing::AppDriver& driver) {
    tuinator::Widget& widget = driver.require(taiji::ui::test_id::kActivityLog);
    const tuinator::Rect bounds = widget.bounds();
    const tuinator::Point at{bounds.x + 6, bounds.y};
    tuinator::MouseEvent press;
    press.position = at;
    press.button = tuinator::MouseButton::Left;
    press.action = tuinator::MouseAction::Press;
    press.left_pressed = true;
    driver.app().dispatch_event(press);
    tuinator::MouseEvent release = press;
    release.action = tuinator::MouseAction::Release;
    release.left_pressed = false;
    driver.app().dispatch_event(release);
}

bool uses_only_main_background(const tuinator::testing::RegionStyleAnalysis& analysis, tuinator::Rgb main_bg) {
    if (!analysis.uniform_background) {
        return false;
    }
    for (const tuinator::Rgb& bg : analysis.distinct_backgrounds) {
        if (bg != main_bg) {
            return false;
        }
    }
    return analysis.left_background.has_value() && *analysis.left_background == main_bg &&
           analysis.right_background.has_value() && *analysis.right_background == main_bg;
}

} // namespace

/// Failed tool rows (header + expanded parameters/result) should use the normal chat panel
/// background, not the red failure tint (activity_log failed_style_ / failed_body_style_).
TAIJI_TUI_TEST(activity_log_failed_tool_uses_main_background_when_expanded) {
    auto log = make_failed_remember_log();
    log->layout({0, 0, 72, 20});
    tuinator::testing::AppDriver driver({72, 20}, std::move(log));
    ui_apply_stylesheet(driver.app(), "dark");
    const tuinator::Rgb main_bg = ui_main_bg();
    click_activity_log_header(driver);
    driver.present();

    const std::string snapshot = driver.snapshot_text();
    TAIJI_TUI_CHECK(snapshot.find("Parameters") != std::string::npos);
    TAIJI_TUI_CHECK(snapshot.find("recall_text") != std::string::npos);

    const tuinator::testing::RegionStyleAnalysis analysis =
        driver.analyze_selector(taiji::ui::test_id::kActivityLog);
    TAIJI_TUI_CHECK(analysis.cells_sampled > 0);
    TAIJI_TUI_CHECK(uses_only_main_background(analysis, main_bg));

    for (const tuinator::Rgb& bg : analysis.distinct_backgrounds) {
        TAIJI_TUI_CHECK(bg != kFailedToolPanelBg);
        TAIJI_TUI_CHECK(bg != kFailedToolBadgeBg);
    }
}

/// Tree branch glyphs (├─, │) stay muted; failure color is only on ✗ and the tool title text.
TAIJI_TUI_TEST(activity_log_failed_tool_tree_branches_are_not_failure_colored) {
    auto log = make_failed_remember_log();
    log->layout({0, 0, 72, 4});
    tuinator::testing::AppDriver driver({72, 4}, std::move(log));
    ui_apply_stylesheet(driver.app(), "dark");
    driver.present();

    const tuinator::Widget& widget = driver.require(taiji::ui::test_id::kActivityLog);
    const tuinator::Rect bounds = widget.bounds();
    const tuinator::Rect branch_column{bounds.x, bounds.y, 2, 1};
    const tuinator::testing::RegionStyleAnalysis branch =
        tuinator::testing::analyze_region(driver.backend(), branch_column);
    TAIJI_TUI_CHECK(branch.cells_sampled > 0);
    for (const tuinator::Rgb& fg : branch.distinct_foregrounds) {
        TAIJI_TUI_CHECK(fg != kFailedToolLabelFg);
    }
}

TAIJI_TUI_TEST(activity_log_truncates_huge_search_result_when_expanded) {
    tuinator::Style muted;
    muted.foreground_rgb = tuinator::rgb(140, 145, 160);
    tuinator::Style active;
    active.foreground_rgb = tuinator::rgb(220, 225, 240);
    auto log = std::make_unique<ActivityLog>(active, muted);
    ui_set_tool_expand_display(true, true, "pretty", "text");
    log->set_wrap_width(64);
    log->layout({0, 0, 72, 24});
    std::string huge(200000, 'a');
    huge += "\nunique-tail-marker";
    log->start_tool("workspace_find", "{}", false, false, "", "");
    log->finish_tool("workspace_find", huge, "", 12);
    tuinator::testing::AppDriver driver({72, 24}, std::move(log));
    ui_apply_stylesheet(driver.app(), "dark");
    click_activity_log_header(driver);
    driver.present();
    const std::string snapshot = driver.snapshot_text();
    TAIJI_TUI_CHECK(snapshot.find("bytes truncated for display") != std::string::npos);
    TAIJI_TUI_CHECK(snapshot.find("unique-tail-marker") == std::string::npos);
}
