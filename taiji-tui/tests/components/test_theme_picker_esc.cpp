#include <tuinator/core/event.hpp>
#include <tuinator/testing/app_driver.hpp>
#include <tuinator/testing/widget_query.hpp>
#include <tuinator/widgets/controls/text_input.hpp>

#include "message_composer.hpp"
#include "scenes/scenes.hpp"
#include "support/test_harness.hpp"
#include "ui_test_ids.hpp"

namespace {

tuinator::KeyPress escape_key() {
    tuinator::KeyPress key;
    key.key = tuinator::Key::Escape;
    return key;
}

tuinator::KeyPress enter_key() {
    tuinator::KeyPress key;
    key.key = tuinator::Key::Enter;
    return key;
}

int selector_height(const tuinator::testing::AppDriver& driver, const char* selector) {
    const tuinator::Widget* widget = tuinator::testing::find_by_selector(driver.root(), selector);
    if (widget == nullptr) {
        return 0;
    }
    return widget->bounds().height;
}

} // namespace

TAIJI_TUI_TEST(theme_picker_esc_restores_draft_and_prompt_layout) {
    auto composer = taiji::ui::test_scenes::make_message_composer();
    MessageComposer* mc = composer.get();
    tuinator::testing::AppDriver driver({80, 24}, std::move(composer));
    driver.present();

    driver.type_into(taiji::ui::test_id::kComposerPromptInput, "Hallo");
    mc->open_theme_picker("dark");
    driver.present();

    TAIJI_TUI_CHECK(mc->is_theme_picker_open());
    TAIJI_TUI_CHECK(selector_height(driver, taiji::ui::test_id::kComposerPromptInput) > 0);

    TAIJI_TUI_CHECK(mc->route_theme_picker_key(escape_key()));
    driver.present();

    TAIJI_TUI_CHECK(!mc->is_theme_picker_open());
    tuinator::Widget* field =
        tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kComposerPromptInput);
    auto* input = dynamic_cast<tuinator::TextInput*>(field);
    TAIJI_TUI_CHECK(input != nullptr);
    TAIJI_TUI_CHECK(input->value() == "Hallo");
    TAIJI_TUI_CHECK(selector_height(driver, taiji::ui::test_id::kComposerPromptInput) > 0);
}

TAIJI_TUI_TEST(theme_picker_esc_keeps_prompt_in_chat_input_bar_column) {
    tuinator::testing::AppDriver driver({80, 16}, taiji::ui::test_scenes::make_composer_column());
    driver.present();
    tuinator::Widget* composer =
        tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kComposerRoot);
    auto* mc = dynamic_cast<MessageComposer*>(composer);
    TAIJI_TUI_CHECK(mc != nullptr);

    mc->open_theme_picker("dark");
    driver.present();
    TAIJI_TUI_CHECK(mc->route_theme_picker_key(escape_key()));
    driver.present();

    TAIJI_TUI_CHECK(selector_height(driver, taiji::ui::test_id::kComposerPromptInput) > 0);
    const std::string snap = driver.snapshot_text();
    TAIJI_TUI_CHECK(snap.find("Ask Taiji") != std::string::npos || snap.find("Write") != std::string::npos ||
                    snap.find("message") != std::string::npos);
}

TAIJI_TUI_TEST(theme_picker_confirm_then_esc_keeps_prompt_visible_in_short_terminal) {
    auto composer = taiji::ui::test_scenes::make_message_composer();
    MessageComposer* mc = composer.get();
    tuinator::testing::AppDriver driver({80, 12}, std::move(composer));
    driver.present();

    mc->open_theme_picker("dark");
    driver.present();
    TAIJI_TUI_CHECK(selector_height(driver, taiji::ui::test_id::kComposerPromptInput) > 0);

    TAIJI_TUI_CHECK(mc->route_theme_picker_key(enter_key()));
    driver.present();
    TAIJI_TUI_CHECK(!mc->is_theme_picker_open());

    mc->route_theme_picker_key(escape_key());
    driver.present();

    TAIJI_TUI_CHECK(selector_height(driver, taiji::ui::test_id::kComposerPromptInput) > 0);
    TAIJI_TUI_CHECK(tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kComposerPromptInput) !=
                    nullptr);
}
