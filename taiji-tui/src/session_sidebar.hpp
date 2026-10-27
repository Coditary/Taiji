#pragma once

#include <tuinator/tuinator.hpp>

#include <cstdint>
#include <functional>
#include <memory>
#include <string>
#include <unordered_map>
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
    using AddFolderHandler = std::function<void()>;

    SessionSidebar(tuinator::Theme theme, SessionSelectHandler on_select, NewSessionHandler on_new,
                   SessionContextHandler on_context, std::function<void()> on_settings = nullptr,
                   AddFolderHandler on_add_folder = nullptr);

    void refresh(uint64_t taiji_handle);
    /// Load sidebar rows from the same JSON shape as `taiji_projects_list` (for tests and fixtures).
    void apply_list_json(const std::string& json);
    void expand_project(const std::string& project_root);
    void set_width_hint(int cols);
    void set_session_generating(const std::string& session_id, bool generating);
    void notify_session_generation_complete(const std::string& session_id);
    void tick_animations(int dt_ms);
    bool needs_animation_tick() const;

    std::string_view widget_type_name() const override { return "SessionSidebar"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    void collect_focusable(std::vector<tuinator::Widget*>& out) override;
    void for_each_descendant(const std::function<void(tuinator::Widget*)>& visitor) override;
    bool wants_hover() const override { return true; }

  private:
    struct Row {
        enum class Kind { SectionHeader, ProjectHeader, Session, EmptyHint } kind = Kind::Session;
        std::string section_key;
        std::string project_root;
        std::string session_id;
        std::string label;
        std::string suffix;
        bool active = false;
        bool section_add = false;
    };

    void rebuild_rows();
    bool is_project_collapsed(const std::string& project_root) const;
    void toggle_project_collapsed(const std::string& project_root);
    std::string format_relative_time(const std::string& iso_time) const;
    int row_at_y(int local_y) const;
    std::string truncate_label(const std::string& text, int max_cols) const;
    int trailing_action_button_x() const;
    bool is_trailing_action_click(int local_x, int local_y) const;
    void parse_project_groups(const std::string& json, const std::string& array_key,
                              std::vector<ProjectGroup>& out);
    bool is_settings_click(int local_x, int local_y) const;
    std::string settings_label() const;
    int footer_y() const { return std::max(0, bounds_.height - 1); }
    void sync_hit_regions();

    tuinator::Theme theme_;
    SessionSelectHandler on_select_;
    NewSessionHandler on_new_;
    SessionContextHandler on_context_;
    std::function<void()> on_settings_;
    AddFolderHandler on_add_folder_;
    std::vector<ProjectGroup> folders_;
    std::vector<ProjectGroup> projects_;
    std::vector<Row> rows_;
    std::unordered_set<std::string> collapsed_projects_;
    int width_hint_ = 26;
    int scroll_offset_ = 0;
    int hovered_row_ = -1;
    bool settings_hover_ = false;
    std::unordered_map<std::string, bool> session_generating_;
    std::unordered_map<std::string, int> session_complete_pulse_ms_;
    int anim_ms_ = 0;
    std::vector<std::unique_ptr<tuinator::Widget>> hit_regions_;
};
