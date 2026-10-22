#pragma once

#include "ui_navigation.hpp"

#include <tuinator/tuinator.hpp>

#include <string>

class StatusFooter : public tuinator::Widget {
  public:
    StatusFooter(tuinator::Style path_style, tuinator::Style status_style);

    void set_path(std::string text);
    void set_status(std::string text);
    void set_vim_mode_badge(bool show, UiInputMode mode);

    std::string_view widget_type_name() const override { return "StatusFooter"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;

  private:
    std::string path_;
    std::string status_;
    bool show_mode_badge_ = false;
    UiInputMode mode_badge_ = UiInputMode::Insert;
    tuinator::Style path_style_;
    tuinator::Style status_style_;
};
