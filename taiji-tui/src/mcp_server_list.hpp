#pragma once

#include <tuinator/tuinator.hpp>

#include <string>
#include <vector>

class McpServerList : public tuinator::Widget {
  public:
    explicit McpServerList(tuinator::Theme theme);

    void set_servers(std::vector<std::string> names);
    void set_status_line(std::string text);
    void set_active_server(const std::string& name);
    void clear_active_server();
    void tick(int dt_ms);
    bool has_active_animation() const { return !active_server_.empty(); }

    std::string_view widget_type_name() const override { return "McpServerList"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;

  private:
    int row_at_y(int local_y) const;

    tuinator::Theme theme_;
    std::vector<std::string> servers_;
    std::string status_line_;
    int hovered_row_ = -1;
    int selected_row_ = -1;
    std::string active_server_;
    int active_pulse_ms_ = 0;
};
