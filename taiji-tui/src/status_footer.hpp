#pragma once

#include <tuinator/tuinator.hpp>

#include <string>

class StatusFooter : public tuinator::Widget {
  public:
    StatusFooter(tuinator::Style path_style, tuinator::Style status_style);

    void set_path(std::string text);
    void set_status(std::string text);

    std::string_view widget_type_name() const override { return "StatusFooter"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;

  private:
    std::string path_;
    std::string status_;
    tuinator::Style path_style_;
    tuinator::Style status_style_;
};
