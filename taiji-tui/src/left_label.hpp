#pragma once

#include <tuinator/tuinator.hpp>

#include <string>

class LeftLabel : public tuinator::Widget {
  public:
    LeftLabel(std::string text, tuinator::Style style);

    void set_text(std::string text);
    void set_style(tuinator::Style style);

    std::string_view widget_type_name() const override { return "LeftLabel"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;

  private:
    std::string text_;
    tuinator::Style style_;
};
