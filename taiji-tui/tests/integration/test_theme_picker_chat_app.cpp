#include <cstdlib>

#include <tuinator/backend/memory_backend.hpp>
#include <tuinator/core/application.hpp>
#include <tuinator/core/event.hpp>
#include <tuinator/testing/cell_styles.hpp>
#include <tuinator/testing/widget_query.hpp>

#include "chat_app.hpp"
#include "chat_input.hpp"
#include "support/test_harness.hpp"
#include "ui_test_ids.hpp"

namespace {

tuinator::KeyPress ctrl_t_key() {
    tuinator::KeyPress key;
    key.ctrl = true;
    key.character = 't';
    return key;
}

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

int painted_prompt_cells(const tuinator::MemoryTerminalBackend& memory, const tuinator::Widget& prompt) {
    return tuinator::testing::analyze_region(memory, prompt.bounds()).cells_sampled;
}

} // namespace

/// Full ChatApp: after theme pick, painted prompt width must match baseline without waiting for poll_updates.
TAIJI_TUI_TEST(theme_picker_ctrl_t_enter_keeps_prompt_in_full_chat_app) {
    setenv("TAIJI_LOCAL", "1", 1);

    tuinator::Application app(tuinator::MemoryTerminalBackend::create({120, 36}));
    ChatApp chat(app);
    chat.set_root();

    for (int i = 0; i < 400; ++i) {
        chat.poll_updates();
        app.step_frame();
    }

    tuinator::Widget* root = app.root();
    TAIJI_TUI_CHECK(root != nullptr);
    tuinator::Widget* prompt =
        tuinator::testing::find_by_selector(*root, taiji::ui::test_id::kComposerPromptInput);
    TAIJI_TUI_CHECK(prompt != nullptr);
    auto* chat_input = dynamic_cast<ChatTextInput*>(prompt);
    TAIJI_TUI_CHECK(chat_input != nullptr);
    chat_input->set_model_info("gemma4_26b-128k-nothink", "Llama Local");

    auto* memory = dynamic_cast<tuinator::MemoryTerminalBackend*>(&app.backend());
    TAIJI_TUI_CHECK(memory != nullptr);

    app.focus_widget(prompt);
    app.present();

    const int baseline_cells = painted_prompt_cells(*memory, *prompt);
    const int baseline_width = prompt->bounds().width;
    const int baseline_height = prompt->bounds().height;
    TAIJI_TUI_CHECK(baseline_width >= 40);
    TAIJI_TUI_CHECK(baseline_height >= 3);
    TAIJI_TUI_CHECK(baseline_cells >= 36);

    app.dispatch_event(ctrl_t_key());
    app.present();

    TAIJI_TUI_CHECK(painted_prompt_cells(*memory, *prompt) == baseline_cells);
    TAIJI_TUI_CHECK(prompt->bounds().width == baseline_width);

    app.dispatch_event(down_key());
    app.present();

    TAIJI_TUI_CHECK(painted_prompt_cells(*memory, *prompt) == baseline_cells);

    app.dispatch_event(enter_key());
    // No poll_updates here — reproduces the real key handler path (immediate repaint).
    app.present();

    TAIJI_TUI_CHECK(prompt->bounds().width == baseline_width);
    TAIJI_TUI_CHECK(prompt->bounds().height == baseline_height);
    TAIJI_TUI_CHECK(painted_prompt_cells(*memory, *prompt) == baseline_cells);

    const std::string screen = memory->snapshot();
    const std::size_t prompt_row = screen.rfind("Write a message");
    if (prompt_row != std::string::npos) {
        const std::string row = screen.substr(prompt_row, 120);
        TAIJI_TUI_CHECK(row.find("…-") == std::string::npos);
        TAIJI_TUI_CHECK(row.find("...-") == std::string::npos);
    }
}
