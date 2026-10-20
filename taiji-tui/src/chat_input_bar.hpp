#pragma once

#include "ui_chat_column.hpp"

#include <tuinator/tuinator.hpp>

#include <memory>

class SpeechMicButton;

/// Composer plus speech mic overlaid at the bottom-right inside the prompt frame.
class ChatInputBar : public tuinator::Widget {
  public:
    void set_composer(std::unique_ptr<tuinator::Widget> composer);
    void set_mic(std::unique_ptr<SpeechMicButton> mic);

    std::string_view widget_type_name() const override { return "ChatInputBar"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    tuinator::Widget* hit_test(tuinator::Point point) override;
    tuinator::Widget* hit_test_focusable(tuinator::Point point) override;

  private:
    std::unique_ptr<CenteredChatColumn> column_;
    std::unique_ptr<SpeechMicButton> mic_;
};
