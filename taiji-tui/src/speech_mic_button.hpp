#pragma once

#include <tuinator/tuinator.hpp>

#include <functional>
#include <string>

class SpeechMicButton : public tuinator::Widget {
  public:
    SpeechMicButton(tuinator::Style normal_style, tuinator::Style recording_style, tuinator::Style hover_style,
                    std::function<void()> on_click);

    void set_recording(bool recording);
    void set_enabled(bool enabled);
    bool recording() const { return recording_; }

    std::string_view widget_type_name() const override { return "SpeechMicButton"; }
    bool wants_hover() const override { return enabled_; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    tuinator::Widget* hit_test(tuinator::Point point) override;

  private:
    std::string label_text() const;

    tuinator::Style normal_style_;
    tuinator::Style recording_style_;
    tuinator::Style hover_style_;
    std::function<void()> on_click_;
    bool recording_ = false;
    bool enabled_ = true;
    bool hover_ = false;
};
