#include <tuinator/backend/memory_backend.hpp>
#include <tuinator/core/application.hpp>
#include <tuinator/core/event.hpp>
#include <tuinator/testing/app_driver.hpp>
#include <tuinator/testing/cell_styles.hpp>
#include <tuinator/testing/widget_query.hpp>

#include "chat_input_bar.hpp"
#include "message_composer.hpp"
#include "scenes/scenes.hpp"
#include "support/test_harness.hpp"
#include "ui_chat_column.hpp"
#include "ui_test_ids.hpp"
#include "ui_theme.hpp"

namespace {

tuinator::KeyPress enter_key() {
    tuinator::KeyPress key;
    key.key = tuinator::Key::Enter;
    return key;
}

tuinator::KeyPress down_key() {
    tuinator::KeyPress key;
    key.key = tuinator::Key::Down;
    return key;
}

struct PromptBarMetrics {
    int width = 0;
    int height = 0;
};

PromptBarMetrics prompt_bar_metrics(const tuinator::testing::AppDriver& driver) {
    const tuinator::Widget* widget =
        tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kComposerPromptInput);
    if (widget == nullptr) {
        return {};
    }
    const tuinator::Rect b = widget->bounds();
    return {b.width, b.height};
}

bool metrics_match(const PromptBarMetrics& a, const PromptBarMetrics& b) {
    return a.width == b.width && a.height == b.height;
}

int painted_prompt_cells(const tuinator::testing::AppDriver& driver) {
    const tuinator::Widget* widget =
        tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kComposerPromptInput);
    if (widget == nullptr) {
        return 0;
    }
    return tuinator::testing::analyze_region(driver.backend(), widget->bounds()).cells_sampled;
}

} // namespace

/// While the theme list is open (and while previewing another theme), the prompt row must not shrink or change copy.
TAIJI_TUI_TEST(theme_picker_keeps_prompt_bar_while_open_and_after_commit) {
    tuinator::Application bootstrap(tuinator::MemoryTerminalBackend::create({1, 1}));
    ui_apply_stylesheet(bootstrap, "dark");
    ui_set_chat_stream_panel_width(0);

    tuinator::testing::AppDriver driver({80, 20}, taiji::ui::test_scenes::make_composer_column());
    driver.present();

    MessageComposer* mc = dynamic_cast<MessageComposer*>(
        tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kComposerRoot));
    TAIJI_TUI_CHECK(mc != nullptr);
    ChatTextInput* input = mc->text_input();
    TAIJI_TUI_CHECK(input != nullptr);
    const std::string baseline_placeholder = input->placeholder();
    input->set_value("Hallo");
    driver.present();

    const PromptBarMetrics baseline = prompt_bar_metrics(driver);
    TAIJI_TUI_CHECK(baseline.width >= 32);
    TAIJI_TUI_CHECK(baseline.height >= 3);
    TAIJI_TUI_CHECK(input->value() == "Hallo");

    mc->open_theme_picker("dark");
    driver.present();

    TAIJI_TUI_CHECK(mc->is_theme_picker_open());
    TAIJI_TUI_CHECK(input->value() == "Hallo");
    TAIJI_TUI_CHECK(input->placeholder() == baseline_placeholder);
    TAIJI_TUI_CHECK(metrics_match(prompt_bar_metrics(driver), baseline));

    TAIJI_TUI_CHECK(mc->route_theme_picker_key(down_key()));
    driver.present();
    TAIJI_TUI_CHECK(metrics_match(prompt_bar_metrics(driver), baseline));
    TAIJI_TUI_CHECK(input->value() == "Hallo");

    TAIJI_TUI_CHECK(mc->route_theme_picker_key(enter_key()));
    driver.present();

    TAIJI_TUI_CHECK(!mc->is_theme_picker_open());
    TAIJI_TUI_CHECK(input->value() == "Hallo");
    TAIJI_TUI_CHECK(input->placeholder() == baseline_placeholder);
    TAIJI_TUI_CHECK(metrics_match(prompt_bar_metrics(driver), baseline));
}

TAIJI_TUI_TEST(theme_picker_keeps_prompt_bar_in_chat_input_column) {
    tuinator::Application bootstrap(tuinator::MemoryTerminalBackend::create({1, 1}));
    ui_apply_stylesheet(bootstrap, "dark");

    tuinator::testing::AppDriver driver({80, 14}, taiji::ui::test_scenes::make_composer_column());
    driver.present();

    MessageComposer* mc = dynamic_cast<MessageComposer*>(
        tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kComposerRoot));
    TAIJI_TUI_CHECK(mc != nullptr);

    const PromptBarMetrics baseline = prompt_bar_metrics(driver);
    TAIJI_TUI_CHECK(baseline.width >= 40);

    const tuinator::Widget* prompt =
        tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kComposerPromptInput);
    TAIJI_TUI_CHECK(prompt != nullptr);
    const int baseline_x = prompt->bounds().x;

    mc->open_theme_picker("dark");
    driver.present();
    TAIJI_TUI_CHECK(metrics_match(prompt_bar_metrics(driver), baseline));
    TAIJI_TUI_CHECK(prompt->bounds().x == baseline_x);

    mc->route_theme_picker_key(enter_key());
    driver.present();
    TAIJI_TUI_CHECK(!mc->is_theme_picker_open());
    TAIJI_TUI_CHECK(metrics_match(prompt_bar_metrics(driver), baseline));
    TAIJI_TUI_CHECK(prompt->bounds().x == baseline_x);
}

TAIJI_TUI_TEST(centered_chat_column_aligns_composer_to_stream_width) {
    tuinator::Application bootstrap(tuinator::MemoryTerminalBackend::create({1, 1}));
    ui_apply_stylesheet(bootstrap, "dark");
    ui_set_chat_stream_panel_width(100);

    tuinator::testing::AppDriver driver({80, 18}, taiji::ui::test_scenes::make_composer_column());
    driver.present();

    const tuinator::Widget* composer =
        tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kComposerRoot);
    TAIJI_TUI_CHECK(composer != nullptr);
    const ChatStreamLayout stream = chat_stream_layout_in_widget(80);
    TAIJI_TUI_CHECK(composer->bounds().width == stream.content_width);

    ui_set_chat_stream_panel_width(0);
}

/// Stale chat stream width (wide viewport) + narrow composer: theme picker must not clip the painted prompt.
TAIJI_TUI_TEST(theme_picker_prompt_paint_fills_width_with_stale_stream_panel) {
    tuinator::Application bootstrap(tuinator::MemoryTerminalBackend::create({1, 1}));
    ui_apply_stylesheet(bootstrap, "dark");
    ui_set_chat_stream_panel_width(100);

    tuinator::testing::AppDriver driver({80, 18}, taiji::ui::test_scenes::make_composer_column());
    driver.present();

    MessageComposer* mc = dynamic_cast<MessageComposer*>(
        tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kComposerRoot));
    TAIJI_TUI_CHECK(mc != nullptr);

    const int baseline_width = prompt_bar_metrics(driver).width;
    const int baseline_cells = painted_prompt_cells(driver);
    TAIJI_TUI_CHECK(baseline_cells >= baseline_width - 2);

    mc->open_theme_picker("dark");
    driver.present();
    TAIJI_TUI_CHECK(painted_prompt_cells(driver) >= baseline_width - 2);

    mc->route_theme_picker_key(enter_key());
    driver.present();
    TAIJI_TUI_CHECK(painted_prompt_cells(driver) >= baseline_width - 2);

    ui_set_chat_stream_panel_width(0);
}
