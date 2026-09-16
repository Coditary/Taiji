#pragma once

#include <tuinator/tuinator.hpp>

#include <functional>
#include <string>

class QueueStatusBar : public tuinator::Widget {
  public:
    QueueStatusBar(tuinator::Theme theme, std::function<void()> on_promote, std::function<void()> on_delete);

    void set_preview(std::string text, std::size_t extra_waiting = 0);
    void set_wrap_width(int width);

    std::string_view widget_type_name() const override { return "QueueStatusBar"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    bool wants_hover() const override { return visible_; }

  private:
    std::string display_text() const;
    int action_area_width() const;
    int promote_x() const;
    int delete_x() const;

    tuinator::Theme theme_;
    std::function<void()> on_promote_;
    std::function<void()> on_delete_;
    std::string preview_;
    std::size_t extra_waiting_ = 0;
    int wrap_width_hint_ = 0;
    bool visible_ = false;
};
