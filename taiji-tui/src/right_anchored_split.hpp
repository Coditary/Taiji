#pragma once

#include <tuinator/tuinator.hpp>

#include <functional>
#include <memory>

/// Chat area + right sidebar. Sidebar width is stored from the right edge and
/// kept stable when the outer (left) split changes.
class RightAnchoredSplit : public tuinator::Widget {
  public:
    RightAnchoredSplit(std::unique_ptr<tuinator::Widget> main, std::unique_ptr<tuinator::Widget> sidebar,
                       int sidebar_width = 28);

    tuinator::Widget* main() const { return main_.get(); }
    tuinator::Widget* sidebar() const { return sidebar_.get(); }
    int sidebar_width() const { return sidebar_width_; }
    void set_sidebar_width(int width);
    void set_on_sidebar_resize(std::function<void(int width)> callback);
    bool pointer_active() const override { return dragging_; }
    bool captures_pointer() const override { return dragging_; }

    std::string_view widget_type_name() const override { return "RightAnchoredSplit"; }
    void set_on_dirty(std::function<void(tuinator::Rect)> callback) override;
    void set_on_layout(std::function<void()> callback) override;
    tuinator::Size preferred_size() const override;
    void layout(tuinator::Rect bounds) override;
    void paint(tuinator::PaintContext& ctx) const override;
    bool handle_event(const tuinator::Event& event) override;
    tuinator::Widget* hit_test(tuinator::Point point) override;
    tuinator::Widget* hit_test_focusable(tuinator::Point point) override;
    bool has_focused_descendant() const override;
    void collect_focusable(std::vector<tuinator::Widget*>& out) override;
    void for_each_child(const std::function<void(tuinator::Widget*)>& visitor) override;
    void for_each_descendant(const std::function<void(tuinator::Widget*)>& visitor) override;

  private:
    static constexpr int kDividerCols = 1;
    static constexpr int kDividerHitSlop = 2;
    static constexpr int kSidebarAdjustDelta = 4;
    static constexpr int kMainAbsMin = 16;
    static constexpr int kSidebarAbsMin = 12;

    int allowed_min_sidebar() const;
    int allowed_max_sidebar() const;
    int clamped_sidebar_width(int width) const;
    int main_width() const;
    tuinator::Rect divider_bounds() const;
    tuinator::Rect divider_hit_bounds() const;
    bool contains_divider(tuinator::Point point) const;
    void update_sidebar_width_from_mouse(tuinator::Point global_position);
    void begin_drag();
    void end_drag();
    bool route_mouse_to_child(const tuinator::MouseEvent& mouse);

    std::unique_ptr<tuinator::Widget> main_;
    std::unique_ptr<tuinator::Widget> sidebar_;
    int default_sidebar_width_;
    int sidebar_width_;
    bool dragging_ = false;
    std::function<void(int width)> on_sidebar_resize_;
};
