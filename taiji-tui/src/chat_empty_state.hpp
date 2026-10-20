#pragma once

#include <tuinator/core/event.hpp>
#include <tuinator/tuinator.hpp>

#include <functional>
#include <string>

class ChatEmptyState : public tuinator::Widget {
  public:
    using QuickActionHandler = std::function<void(int index)>;

    explicit ChatEmptyState(tuinator::Theme theme);

    void set_visible(bool visible);
    void tick(int dt_ms);
    void set_workspace_path(std::string path);
    void set_on_quick_action(QuickActionHandler handler);
    bool is_visible() const { return visible_; }

    std::string_view widget_type_name() const override { return "ChatEmptyState"; }

    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    bool try_quick_action(const tuinator::KeyPress& key);

  private:
    tuinator::Theme theme_;
    bool visible_ = true;
    std::string workspace_path_;
    QuickActionHandler on_quick_action_;
    int reveal_ms_ = 0;
    int anim_clock_ms_ = 0;
};
