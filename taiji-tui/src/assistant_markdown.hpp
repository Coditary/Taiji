#pragma once

#include <tuinator/widgets/display/markdown_view.hpp>

#include <string>
#include <string_view>

/// MarkdownView sized for a vertical chat log (full content height, outer scroll).
class AssistantMarkdown : public tuinator::Widget {
  public:
    explicit AssistantMarkdown(tuinator::MarkdownViewOptions options = {});

    void append(std::string_view chunk);
    void set_markdown(std::string text);
    void clear();
    bool is_empty() const;
    void set_wrap_width(int width);
    void sync_from(const std::string& text, bool clear_first);
    void tick_stream_glow(int dt_ms);
    void set_markdown_styles(tuinator::MarkdownStyleSet styles);
    bool has_stream_animation() const { return stream_glow_ms_ > 0; }

    std::string_view widget_type_name() const override { return "AssistantMarkdown"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;

  private:
    void relayout_content();
    void refresh_measure_cache(int width) const;

    tuinator::MarkdownView view_;
    int wrap_width_hint_ = 0;
    int content_indent_ = 2;
    mutable int cached_width_ = 0;
    mutable int cached_height_ = 1;
    int stream_glow_ms_ = 0;
};
