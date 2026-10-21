#pragma once

#include <tuinator/tuinator.hpp>

#include <functional>
#include <string>

class SidebarFooter : public tuinator::Widget {
  public:
    explicit SidebarFooter(tuinator::Theme theme);

    std::string_view widget_type_name() const override { return "SidebarFooter"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    bool wants_hover() const override { return true; }
    tuinator::Widget* hit_test(tuinator::Point point) override;

  private:
    tuinator::Theme theme_;
};
