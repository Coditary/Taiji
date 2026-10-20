#pragma once

#include <tuinator/tuinator.hpp>

class SidebarDivider : public tuinator::Widget {
  public:
    explicit SidebarDivider(tuinator::Theme theme);

    std::string_view widget_type_name() const override { return "SidebarDivider"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;

  private:
    tuinator::Theme theme_;
};
