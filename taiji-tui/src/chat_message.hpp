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
    /// Display name in messenger/log/section (empty = "You"). Future: per-turn agent roles.
    void set_author_label(std::string label);
    /// Compacted history summaries always render as a section row (label e.g. "summary").
    void set_compact_summary(bool compact);
    bool is_compact_summary() const { return compact_summary_; }
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
    std::string speaker_name() const;
    std::string log_speaker_name() const;
    tuinator::Size preferred_size_user() const;
    void paint_user_message(tuinator::PaintContext& ctx, const tuinator::Rect& layout, int max_body_lines = 0) const;
    int sticky_preview_height_for_style(int max_body_lines) const;
    void paint_user_block(tuinator::PaintContext& ctx, const tuinator::Rect& layout, int max_body_lines) const;
    int block_body_wrap_cols(int stream_w) const;
    int sticky_user_top_rows() const;

    std::string text_;
    tuinator::Style style_;
    Align align_;
    Variant variant_ = Variant::Plain;
    int wrap_width_hint_ = 0;
    std::string time_label_;
    std::string author_label_;
    bool compact_summary_ = false;
};
