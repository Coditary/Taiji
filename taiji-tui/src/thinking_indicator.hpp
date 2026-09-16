#pragma once

#include <tuinator/tuinator.hpp>

class ThinkingIndicator : public tuinator::HBox {
  public:
    ThinkingIndicator(tuinator::Style throbber_style, tuinator::Style label_style);

    void set_active(bool active);
    bool active() const { return active_; }
    void set_label(const std::string& text);
    void tick(int dt_ms);

    tuinator::Size preferred_size() const override;
    void paint(tuinator::PaintContext& ctx) const override;

  private:
    bool active_ = false;
    tuinator::Throbber* throbber_ = nullptr;
    tuinator::Label* label_ = nullptr;
};
