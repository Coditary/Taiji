#pragma once

#include <tuinator/tuinator.hpp>

#include <string>
#include <vector>

class ChatMessage : public tuinator::Widget {
  public:
    enum class Align { Left, Right };
    enum class Variant { Plain, UserPrompt };

    ChatMessage(std::string text, tuinator::Style style, Align align, Variant variant = Variant::Plain);

    void set_text(std::string text);
    void set_style(tuinator::Style style);
    void set_wrap_width(int width);
    void set_variant(Variant variant);
    void set_time_label(std::string time_label);
    std::string_view widget_type_name() const override { return "ChatMessage"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;

    int sticky_preview_height(int max_body_lines) const;
    void paint_sticky_preview(tuinator::PaintContext& ctx, int panel_width, int max_body_lines) const;

  private:
    std::vector<std::string> wrapped_lines() const;
    int wrap_cols() const;

    int content_wrap_cols() const;
    std::vector<std::string> content_lines() const;

    std::string text_;
    tuinator::Style style_;
    Align align_;
    Variant variant_ = Variant::Plain;
    int wrap_width_hint_ = 0;
    std::string time_label_;
};
