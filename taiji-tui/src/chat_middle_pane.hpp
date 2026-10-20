#pragma once

#include <tuinator/tuinator.hpp>

#include <functional>
#include <memory>

class ChatMessage;

class ChatMiddlePane : public tuinator::Widget {
  public:
    ChatMiddlePane(std::unique_ptr<tuinator::Widget> empty_state, std::unique_ptr<tuinator::Widget> chat_scroll);

    tuinator::Widget* empty_state() const { return empty_state_.get(); }
    tuinator::Widget* chat_scroll() const { return chat_scroll_.get(); }

    void set_show_empty(bool show);
    void set_sticky_user_message_enabled(bool enabled);
    void set_sticky_user_resolver(std::function<const ChatMessage*()> resolver);
    bool shows_chat_log() const { return !show_empty_; }

    std::string_view widget_type_name() const override { return "ChatMiddlePane"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    tuinator::Widget* hit_test(tuinator::Point point) override;
    tuinator::Widget* hit_test_focusable(tuinator::Point point) override;
    void collect_focusable(std::vector<tuinator::Widget*>& out) override;

  private:
    int sticky_overlay_height() const;

    std::unique_ptr<tuinator::Widget> empty_state_;
    std::unique_ptr<tuinator::Widget> chat_scroll_;
    bool show_empty_ = true;
    bool sticky_user_message_enabled_ = true;
    std::function<const ChatMessage*()> sticky_user_resolver_;
};
