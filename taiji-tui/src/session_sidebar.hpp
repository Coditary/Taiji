#pragma once

#include <tuinator/tuinator.hpp>

#include <cstdint>
#include <functional>
#include <string>
#include <unordered_set>
#include <vector>

class SessionSidebar : public tuinator::Widget {
  public:
    struct SessionItem {
        std::string project_root;
        std::string session_id;
        std::string name;
        std::string created_at;
        bool active = false;
    };

    struct ProjectGroup {
        std::string root;
        std::string name;
        std::vector<SessionItem> sessions;
    };

    using SessionSelectHandler = std::function<void(const std::string& project_root, const std::string& session_id)>;
    using NewSessionHandler = std::function<void(const std::string& project_root)>;
    using SessionContextHandler = std::function<void(const std::string& project_root, const std::string& session_id,
                                                     tuinator::Point position)>;

    SessionSidebar(tuinator::Theme theme, SessionSelectHandler on_select, NewSessionHandler on_new,
                   SessionContextHandler on_context, std::function<void()> on_settings = nullptr);

    void refresh(uint64_t taiji_handle);
    void expand_project(const std::string& project_root);
    void set_width_hint(int cols);

    std::string_view widget_type_name() const override { return "SessionSidebar"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    void collect_focusable(std::vector<tuinator::Widget*>& out) override;
    bool wants_hover() const override { return true; }

  private:
    struct Row {
        enum class Kind { ProjectHeader, Session } kind = Kind::Session;
        std::string project_root;
        std::string session_id;
        std::string label;
        std::string suffix;
        bool active = false;
    };

    void rebuild_rows();
    bool is_project_collapsed(const std::string& project_root) const;
    void toggle_project_collapsed(const std::string& project_root);
    std::string format_relative_time(const std::string& iso_time) const;
    int row_at_y(int local_y) const;
    std::string truncate_label(const std::string& text, int max_cols) const;
    int project_new_button_x() const;
    bool is_project_new_click(int local_x, int local_y) const;
    bool is_settings_click(int local_x, int local_y) const;
    std::string settings_label() const;
    int footer_y() const { return std::max(0, bounds_.height - 1); }

    tuinator::Theme theme_;
    SessionSelectHandler on_select_;
    NewSessionHandler on_new_;
    SessionContextHandler on_context_;
    std::function<void()> on_settings_;
    std::vector<ProjectGroup> projects_;
    std::vector<Row> rows_;
    std::unordered_set<std::string> collapsed_projects_;
    int width_hint_ = 26;
    int scroll_offset_ = 0;
    int hovered_row_ = -1;
    bool settings_hover_ = false;
};
