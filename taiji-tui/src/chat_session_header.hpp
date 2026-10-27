#pragma once

#include <tuinator/tuinator.hpp>

#include <string>

class ChatSessionHeader : public tuinator::Widget {
  public:
    explicit ChatSessionHeader(tuinator::Theme theme);

    void set_project_name(std::string name);
    void set_session_name(std::string name);
    void set_template_persona(std::string label, const std::string& theme_color_hex);
    void tick_persona_pulse(int interval_ms);
    void trigger_response_ready_pulse();
    bool needs_animation_tick() const { return persona_pulse_ms_ > 0 || ready_pulse_ms_ > 0; }

    std::string_view widget_type_name() const override { return "ChatSessionHeader"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;

  private:
    tuinator::Theme theme_;
    std::string project_;
    std::string session_;
    std::string template_persona_;
    tuinator::Rgb template_persona_rgb_{203, 166, 247};
    bool template_persona_rgb_set_ = false;
    int persona_pulse_ms_ = 0;
    int ready_pulse_ms_ = 0;
};
