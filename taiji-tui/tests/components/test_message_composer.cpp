#include <tuinator/testing/app_driver.hpp>
#include <tuinator/testing/widget_query.hpp>
#include <tuinator/widgets/controls/text_input.hpp>

#include "scenes/scenes.hpp"
#include "support/test_harness.hpp"
#include "ui_test_ids.hpp"

TAIJI_TUI_TEST(composer_prompt_input_is_in_widget_tree) {
    tuinator::testing::AppDriver driver({72, 24}, taiji::ui::test_scenes::make_composer_column());
    driver.present();

    TAIJI_TUI_CHECK(tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kChatInputBar) != nullptr);
    TAIJI_TUI_CHECK(tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kComposerRoot) != nullptr);
    TAIJI_TUI_CHECK(tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kComposerPromptInput) !=
                    nullptr);
}

TAIJI_TUI_TEST(composer_prompt_input_exposes_text_field) {
    tuinator::testing::AppDriver driver({72, 24}, taiji::ui::test_scenes::make_message_composer());
    tuinator::Widget* field =
        tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kComposerPromptInput);
    auto* input = dynamic_cast<tuinator::TextInput*>(field);
    TAIJI_TUI_CHECK(input != nullptr);
    input->set_value("hello");
    driver.present();
    TAIJI_TUI_CHECK(input->value() == "hello");
}
