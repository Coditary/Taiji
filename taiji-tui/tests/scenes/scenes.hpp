#pragma once

#include "chat_input_bar.hpp"
#include "message_composer.hpp"
#include "session_sidebar.hpp"
#include "status_footer.hpp"
#include "ui_test_ids.hpp"
#include "ui_theme.hpp"

#include <tuinator/layout/box.hpp>
#include <tuinator/testing/app_driver.hpp>

#include <memory>
#include <string>

namespace taiji::ui::test_scenes {

inline tuinator::Theme scene_theme() {
    return tuinator::dark_theme();
}

inline std::unique_ptr<MessageComposer> make_message_composer() {
    tuinator::TextInputOptions opts{.min_width = 40, .placeholder = "Ask Taiji…"};
    tuinator::Style style = ui_composer_input_style();
    tuinator::Style focused = style;
    return std::make_unique<MessageComposer>(opts, style, focused, scene_theme());
}

inline std::unique_ptr<tuinator::Widget> make_composer_column() {
    auto bar = std::make_unique<ChatInputBar>();
    bar->set_composer(make_message_composer());
    return bar;
}

inline constexpr const char* kSidebarFixtureJson = R"json({
  "current_session": "sess-alpha",
  "projects": [{
    "root": "/home/demo/proj",
    "name": "Demo Project",
    "sessions": [{
      "id": "sess-alpha",
      "name": "Alpha chat",
      "created_at": "2025-10-01T12:00:00Z"
    }, {
      "id": "sess-beta",
      "name": "Beta chat",
      "created_at": "2025-10-02T08:30:00Z"
    }]
  }]
})json";

inline std::unique_ptr<SessionSidebar> make_session_sidebar() {
    auto sidebar = std::make_unique<SessionSidebar>(
        scene_theme(), [](const std::string&, const std::string&) {}, [](const std::string&) {},
        [](const std::string&, const std::string&, tuinator::Point) {}, []() {}, []() {});
    sidebar->set_width_hint(28);
    sidebar->apply_list_json(kSidebarFixtureJson);
    return sidebar;
}

inline std::unique_ptr<tuinator::Widget> make_sidebar_with_footer() {
    auto column = std::make_unique<tuinator::VBox>(tuinator::BoxOptions{.gap = 0});
    column->set_test_id("scene.sidebar-column");
    auto sidebar = make_session_sidebar();
    column->add_child(std::move(sidebar));
    tuinator::Style path_style;
    tuinator::Style status_style;
    auto footer = std::make_unique<StatusFooter>(path_style, status_style);
    footer->set_path("/home/demo/proj");
    footer->set_status("Ready");
    column->add_child(std::move(footer));
    return column;
}

inline std::unique_ptr<tuinator::Widget> build_scene_root(const std::string& scene_id) {
    if (scene_id == "composer" || scene_id.empty()) {
        return make_composer_column();
    }
    if (scene_id == "session-sidebar") {
        return make_session_sidebar();
    }
    if (scene_id == "sidebar-footer") {
        return make_sidebar_with_footer();
    }
    return make_composer_column();
}

} // namespace taiji::ui::test_scenes
