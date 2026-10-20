#pragma once

#include <tuinator/tuinator.hpp>

#include <functional>
#include <string>

class SettingsGearButton : public tuinator::Widget {
  public:
    SettingsGearButton(tuinator::Style normal_style, tuinator::Style hover_style,
                       std::function<void()> on_click);

    std::string_view widget_type_name() const override { return "SettingsGearButton"; }
    bool wants_hover() const override { return true; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    tuinator::Widget* hit_test(tuinator::Point point) override;

  private:
    static constexpr const char* kLabel = " ⚙ Settings ";

    tuinator::Style normal_style_;
    tuinator::Style hover_style_;
    std::function<void()> on_click_;
    bool hover_ = false;
};
