#include <tuinator/backend/memory_backend.hpp>
#include <tuinator/core/application.hpp>
#include <tuinator/core/event.hpp>
#include <tuinator/testing/app_driver.hpp>
#include <tuinator/testing/cell_styles.hpp>
#include <tuinator/testing/widget_query.hpp>

#include "chat_input.hpp"
#include "message_composer.hpp"
#include "scenes/scenes.hpp"
#include "support/test_harness.hpp"
#include "ui_chat_column.hpp"
#include "ui_keybindings.hpp"
#include "ui_test_ids.hpp"
#include "ui_theme.hpp"

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

class ThemePickerSystemShell : public tuinator::Widget {
  public:
    void set_child(std::unique_ptr<tuinator::Widget> child) {
        child_ = std::move(child);
        if (child_ != nullptr) {
            attach_child_widget(child_.get());
        }
    }

    void set_preprocess(std::function<bool(const tuinator::Event&)> handler) {
        preprocess_ = std::move(handler);
    }

    bool wants_full_screen() const override { return true; }

    tuinator::Size preferred_size() const override {
        if (bounds_.width > 0 && bounds_.height > 0) {
            return bounds_.size();
        }
        return child_ != nullptr ? child_->preferred_size() : tuinator::Size{80, 24};
    }

    void layout(tuinator::Rect bounds) override {
        bounds_ = bounds;
        if (child_ != nullptr) {
            child_->layout(bounds_);
        }
    }

    void paint(tuinator::PaintContext& ctx) const override {
        if (child_ != nullptr) {
            child_->paint(ctx);
        }
    }

    bool handle_event(const tuinator::Event& event) override {
        if (preprocess_ && preprocess_(event)) {
            mark_dirty();
            return true;
        }
        return child_ != nullptr && child_->handle_event(event);
    }

    tuinator::Widget* hit_test(tuinator::Point point) override {
        return child_ != nullptr ? child_->hit_test(point) : nullptr;
    }

    tuinator::Widget* hit_test_focusable(tuinator::Point point) override {
        return child_ != nullptr ? child_->hit_test_focusable(point) : nullptr;
    }

    void for_each_descendant(const std::function<void(tuinator::Widget*)>& visitor) override {
        if (child_ != nullptr) {
            visitor(child_.get());
            child_->for_each_descendant(visitor);
        }
        Widget::for_each_descendant(visitor);
    }

  private:
    std::unique_ptr<tuinator::Widget> child_;
    std::function<bool(const tuinator::Event&)> preprocess_;
};

struct ThemePickerSystemScene {
    MessageComposer* composer = nullptr;
    tuinator::Widget* layout_root = nullptr;
    std::string current_theme = "dark";
};

MessageComposer* find_message_composer(tuinator::Widget& root) {
    tuinator::Widget* node = tuinator::testing::find_by_selector(root, taiji::ui::test_id::kComposerRoot);
    return dynamic_cast<MessageComposer*>(node);
}

int prompt_input_height(const tuinator::testing::AppDriver& driver) {
    const tuinator::Widget* widget =
        tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kComposerPromptInput);
    return widget != nullptr ? widget->bounds().height : 0;
}

int prompt_input_width(const tuinator::testing::AppDriver& driver) {
    const tuinator::Widget* widget =
        tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kComposerPromptInput);
    return widget != nullptr ? widget->bounds().width : 0;
}

void wire_chat_app_theme_behavior(ThemePickerSystemScene& scene, tuinator::Application& app,
                                  ThemePickerSystemShell& shell) {
    scene.composer->set_on_palette_open_changed([&scene, &app](bool) {
        if (scene.composer != nullptr) {
            scene.composer->mark_layout_dirty();
        }
        if (scene.layout_root != nullptr) {
            scene.layout_root->mark_layout_dirty();
        }
        app.repaint_all();
    });
    scene.composer->set_on_theme_preview([&scene, &app](const std::string& theme_id) {
        ui_apply_stylesheet(app, theme_id);
        scene.current_theme = theme_id;
        if (scene.composer != nullptr) {
            scene.composer->sync_field_styles(ui_composer_input_style(), ui_composer_input_style());
        }
        app.repaint_all();
    });
    scene.composer->set_on_theme_picked([&scene, &app](const std::string& theme_id) {
        ui_apply_stylesheet(app, theme_id);
        scene.current_theme = theme_id;
        if (scene.composer != nullptr) {
            scene.composer->sync_field_styles(ui_composer_input_style(), ui_composer_input_style());
        }
        app.repaint_all();
    });

    shell.set_preprocess([&scene, &app](const tuinator::Event& event) {
        const auto* key = std::get_if<tuinator::KeyPress>(&event);
        if (key == nullptr || scene.composer == nullptr) {
            return false;
        }
        if (scene.composer->is_theme_picker_open()) {
            if (scene.composer->route_theme_picker_key(*key)) {
                scene.composer->mark_dirty();
                if (scene.layout_root != nullptr) {
                    scene.layout_root->mark_layout_dirty();
                }
                app.repaint_all();
                return true;
            }
            return false;
        }
        if (ui_keybinding_matches(ui_keybinding_default_open_theme_picker(), *key)) {
            scene.composer->open_theme_picker(scene.current_theme);
            scene.composer->mark_layout_dirty();
            if (scene.layout_root != nullptr) {
                scene.layout_root->mark_layout_dirty();
            }
            app.repaint_all();
            return true;
        }
        return false;
    });
}

} // namespace

/// System-style repro: prompt starts normal, Ctrl+T opens themes, Enter confirms — prompt must keep full height/width.
TAIJI_TUI_TEST(theme_picker_ctrl_t_enter_keeps_prompt_bar_layout) {
    constexpr int kTermCols = 120;
    constexpr int kTermRows = 32;
    constexpr int kSimulatedChatViewportCols = 96;

    ui_set_chat_stream_panel_width(0);

    tuinator::Application bootstrap(tuinator::MemoryTerminalBackend::create({1, 1}));
    ui_apply_stylesheet(bootstrap, "dark");

    ThemePickerSystemScene scene;
    auto shell = std::make_unique<ThemePickerSystemShell>();
    shell->set_test_id(taiji::ui::test_id::kAppShell);
    auto column = taiji::ui::test_scenes::make_composer_column();
    scene.layout_root = column.get();
    scene.composer = find_message_composer(*scene.layout_root);
    TAIJI_TUI_CHECK(scene.composer != nullptr);
    shell->set_child(std::move(column));

    ThemePickerSystemShell* shell_ptr = shell.get();
    tuinator::testing::AppDriver driver({kTermCols, kTermRows}, std::move(shell));
    ui_apply_stylesheet(driver.app(), scene.current_theme);
    scene.composer->sync_field_styles(ui_composer_input_style(), ui_composer_input_style());
    wire_chat_app_theme_behavior(scene, driver.app(), *shell_ptr);

    tuinator::Widget* prompt =
        tuinator::testing::find_by_selector(driver.root(), taiji::ui::test_id::kComposerPromptInput);
    auto* chat_input = dynamic_cast<ChatTextInput*>(prompt);
    TAIJI_TUI_CHECK(chat_input != nullptr);
    chat_input->set_model_info("gemma4_26b-128k-nothink", "Llama Local");

    driver.focus(taiji::ui::test_id::kComposerPromptInput);
    driver.present();

    const int baseline_width = prompt_input_width(driver);
    const int baseline_height = prompt_input_height(driver);
    TAIJI_TUI_CHECK(baseline_width >= 40);
    TAIJI_TUI_CHECK(baseline_height >= 1);

    ui_set_chat_stream_panel_width(kSimulatedChatViewportCols);
    scene.layout_root->mark_layout_dirty();
    driver.present();
    TAIJI_TUI_CHECK(prompt_input_height(driver) >= baseline_height);

    driver.app().dispatch_event(ctrl_t_key());
    driver.present();
    TAIJI_TUI_CHECK(scene.composer->is_theme_picker_open());

    driver.app().dispatch_event(enter_key());
    driver.present();
    TAIJI_TUI_CHECK(!scene.composer->is_theme_picker_open());

    const int after_width = prompt_input_width(driver);
    const int after_height = prompt_input_height(driver);
    TAIJI_TUI_CHECK(after_height >= baseline_height);
    TAIJI_TUI_CHECK(after_width >= baseline_width);

    const tuinator::testing::RegionStyleAnalysis after_style =
        driver.analyze_selector(taiji::ui::test_id::kComposerPromptInput);
    TAIJI_TUI_CHECK(after_style.cells_sampled >= 24);
    TAIJI_TUI_CHECK(after_style.uniform_background);

    const std::string screen = driver.snapshot_text();
    const std::size_t prompt_row = screen.rfind("Write a message");
    if (prompt_row != std::string::npos) {
        const std::string row = screen.substr(prompt_row, 120);
        TAIJI_TUI_CHECK(row.find("…-") == std::string::npos);
        TAIJI_TUI_CHECK(row.find("...-") == std::string::npos);
    }

    ui_set_chat_stream_panel_width(0);
}
