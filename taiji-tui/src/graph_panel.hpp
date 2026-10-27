#pragma once

#include <tuinator/tuinator.hpp>

#include <string>
#include <vector>

class GraphPanel : public tuinator::Widget {
  public:
    enum class Layout { Sidebar, HorizontalStrip };

    explicit GraphPanel(tuinator::Theme theme);

    void set_layout(Layout layout);
    Layout layout() const { return layout_; }

    void set_from_json(const std::string& json);
    bool is_active() const { return active_; }
    void set_compact_mode(bool compact);
    bool compact_mode() const { return compact_; }
    void toggle_compact_mode();
    void set_tree_overlay(bool open);
    bool tree_overlay() const { return tree_overlay_; }
    void toggle_tree_overlay();
    void set_on_layout_changed(std::function<void()> callback);

    std::size_t successor_count() const { return successor_ids_.size(); }
    std::string successor_id(std::size_t index) const;
    std::string hotkey_hint_line() const;

    std::string_view widget_type_name() const override { return "GraphPanel"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    void collect_focusable(std::vector<tuinator::Widget*>& out) override;
    bool wants_hover() const override { return active_; }

  private:
    struct SuccessorExit {
        std::string id;
        std::string title_line;
    };

    void paint_radar(tuinator::PaintContext& ctx, int x, int y, int w, int max_h, tuinator::Style label,
                     tuinator::Style muted, tuinator::Style accent) const;
    void paint_entities(tuinator::PaintContext& ctx, int x, int y, int w, int max_h, tuinator::Style label,
                        tuinator::Style muted, tuinator::Style accent) const;
    void paint_horizontal_strip(tuinator::PaintContext& ctx, tuinator::Style label, tuinator::Style muted,
                                tuinator::Style accent) const;
    void paint_tree_overlay(tuinator::PaintContext& ctx) const;
    int content_height() const;

    tuinator::Theme theme_;
    Layout layout_ = Layout::Sidebar;
    bool active_ = false;
    bool compact_ = false;
    bool tree_overlay_ = false;
    std::string graph_name_;
    std::string current_title_;
    std::string node_state_;
    std::string exit_hint_;
    std::string breadcrumb_;
    int visible_turns_ = 0;
    int entity_count_ = 0;
    std::vector<std::string> predecessors_;
    std::vector<SuccessorExit> successors_;
    std::vector<std::string> here_;
    std::vector<std::string> party_;
    std::vector<std::string> node_tools_;
    std::vector<std::string> tree_lines_;
    std::vector<std::string> successor_ids_;
    std::function<void()> on_layout_changed_;
};
