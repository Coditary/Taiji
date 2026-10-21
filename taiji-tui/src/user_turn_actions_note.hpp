#pragma once

#include "ui_chat_column.hpp"

#include <tuinator/tuinator.hpp>

#include <functional>
#include <string>

class UserTurnActionsNote : public tuinator::Widget {
  public:
    enum class Action { Edit, Copy };

    UserTurnActionsNote(tuinator::Theme theme, std::function<void(Action)> on_action);

    void set_actions_visible(bool visible);
    void clear();
    void set_wrap_width(int width);
    bool set_actions_hover(bool hovered);
    static int reserved_right_width(bool actions_visible);

    std::string_view widget_type_name() const override { return "UserTurnActionsNote"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    bool wants_hover() const override { return actions_enabled_; }

  private:
    int actions_cluster_width() const;
    int actions_cluster_start_x() const;
    int edit_x() const;
    int copy_x() const;
    int primary_actions_width() const;
    bool show_primary_actions() const;
    ChatStreamLayout chat_stream_layout() const;
    int content_right_x() const;

    tuinator::Theme theme_;
    std::function<void(Action)> on_action_;
    bool actions_enabled_ = false;
    bool actions_hover_ = false;
    int wrap_width_hint_ = 0;
    int right_inset_ = 1;
};
