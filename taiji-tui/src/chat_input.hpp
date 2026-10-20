#pragma once

#include <tuinator/widgets/controls/text_input.hpp>

#include <functional>
#include <string>

class ChatTextInput : public tuinator::TextInput {
  public:
    using QuickKeyHandler = std::function<bool(const tuinator::KeyPress&)>;

    ChatTextInput(tuinator::TextInputOptions options, tuinator::Style style, tuinator::Style focused_style);

    void set_model_info(std::string model, std::string provider);
    void sync_frame_border_style();
    void sync_field_styles(tuinator::Style normal, tuinator::Style focused);
    void set_quick_key_handler(QuickKeyHandler handler);
    void set_chrome_busy(bool busy);
    void tick_chrome(int dt_ms);
    bool chrome_animating() const { return chrome_busy_; }
    bool needs_caret_animation() const { return is_focused(); }

    bool wants_initial_focus() const override { return true; }

    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;

  private:
    void paint_border_label(tuinator::PaintContext& ctx) const;
    void paint_chrome_animation(tuinator::PaintContext& ctx) const;
    void paint_caret(tuinator::PaintContext& ctx) const;
    void note_caret_activity();

    tuinator::TextInputOptions options_;
    int chrome_anim_ms_ = 0;
    bool chrome_busy_ = false;
    mutable bool caret_visible_ = true;
    mutable int caret_blink_ms_ = 0;
    mutable int caret_solid_ms_ = 0;
    std::string model_label_;
    std::string provider_label_;
    QuickKeyHandler quick_key_handler_;
};
