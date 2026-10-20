#pragma once

#include <tuinator/tuinator.hpp>

#include <memory>

constexpr int kChatStreamMaxCols = 88;
constexpr int kChatStreamPadX = 2;
constexpr int kChatStreamPadTop = 1;

struct ChatStreamLayout {
    int content_width = kChatStreamMaxCols;
    int offset_x = 0;
};

int chat_stream_content_width(int panel_width);
void ui_set_chat_stream_max_cols(int max_cols);
/// Shared reference width (scroll viewport) so chat rows and the prompt use the same column.
void ui_set_chat_stream_panel_width(int panel_width);
int ui_chat_stream_panel_width();
ChatStreamLayout chat_stream_layout_for_panel(int panel_width);
ChatStreamLayout chat_stream_layout_for_panel(int panel_width, int max_content_cols);

/// Stream column aligned inside a widget (avoids double-inset when the widget is narrower than the panel).
ChatStreamLayout chat_stream_layout_in_widget(int widget_width);

/// Preferred row width for chat log children (matches scroll viewport / prompt bar).
int chat_stream_row_preferred_width(int widget_bounds_width);

/// Full-width host; lays out its child in a centered max-width column (chat + input share the same inset).
class CenteredChatColumn : public tuinator::Widget {
  public:
    explicit CenteredChatColumn(std::unique_ptr<tuinator::Widget> content);

    tuinator::Widget* content() const { return content_.get(); }

    void set_max_content_width(int cols);
    int max_content_width() const { return max_content_width_; }

    /// When false, content uses the full host height (e.g. bottom prompt bar).
    void set_apply_top_pad(bool apply) { apply_top_pad_ = apply; }

    std::string_view widget_type_name() const override { return "CenteredChatColumn"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    tuinator::Widget* hit_test(tuinator::Point point) override;
    tuinator::Widget* hit_test_focusable(tuinator::Point point) override;
    void collect_focusable(std::vector<tuinator::Widget*>& out) override;

  private:
    std::unique_ptr<tuinator::Widget> content_;
    int max_content_width_ = kChatStreamMaxCols;
    int last_panel_width_ = 0;
    bool apply_top_pad_ = true;
};
