#pragma once

#include "ui_chat_column.hpp"

#include <tuinator/tuinator.hpp>

#include <cstddef>
#include <functional>
#include <string>
#include <vector>

class TurnMetricsNote : public tuinator::Widget {
  public:
    enum class Action { Redo, Copy, Branch, PrevVersion, NextVersion };

    TurnMetricsNote(tuinator::Theme theme, std::function<void(Action)> on_action);

    void set_content(std::string metrics_text, bool actions_visible, std::size_t version_index,
                     std::size_t version_count);
    void clear();
    void set_wrap_width(int width);
    void set_right_inset(int columns);
    bool set_actions_hover(bool hovered);
    static int reserved_right_width(bool actions_visible, std::size_t version_count);

    std::string_view widget_type_name() const override { return "TurnMetricsNote"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    bool wants_hover() const override { return actions_enabled_; }

  private:
    std::vector<std::string> metric_lines() const;
    int actions_cluster_width() const;
    int actions_cluster_start_x() const;
    int redo_x() const;
    int copy_x() const;
    int branch_x() const;
    int primary_actions_width() const;
    int prev_x() const;
    int version_label_x() const;
    int next_x() const;
    std::string version_label() const;
    ChatStreamLayout chat_stream_layout() const;
    int content_right_x() const;
    int metrics_x() const;
    int metrics_max_width() const;
    bool show_primary_actions() const;

    tuinator::Theme theme_;
    tuinator::Style style_;
    std::function<void(Action)> on_action_;
    std::string metrics_text_;
    bool actions_enabled_ = false;
    bool actions_hover_ = false;
    std::size_t version_index_ = 0;
    std::size_t version_count_ = 0;
    int wrap_width_hint_ = 0;
    int right_inset_ = 1;
};
