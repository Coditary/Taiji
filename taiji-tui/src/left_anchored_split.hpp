#pragma once

#include <tuinator/tuinator.hpp>

#include <functional>
#include <memory>

class LeftAnchoredSplit : public tuinator::Widget {
  public:
    LeftAnchoredSplit(std::unique_ptr<tuinator::Widget> left, std::unique_ptr<tuinator::Widget> right,
                      int left_width = 26);

    tuinator::Widget* left() const { return left_.get(); }
    tuinator::Widget* right() const { return right_.get(); }
    int left_width() const { return left_width_; }
    void set_left_width(int width);
    void set_on_left_resize(std::function<void(int width)> callback);
    bool pointer_active() const override { return dragging_; }
    bool captures_pointer() const override { return dragging_; }

    std::string_view widget_type_name() const override { return "LeftAnchoredSplit"; }
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
    static constexpr int kLeftAdjustDelta = 4;
    static constexpr int kRestMinCols = 29;

    int allowed_min_left() const;
    int allowed_max_left() const;
    int clamped_left_width(int width) const;
    int right_width() const;
    tuinator::Rect divider_bounds() const;
    tuinator::Rect divider_hit_bounds() const;
    bool contains_divider(tuinator::Point point) const;
    void update_left_width_from_mouse(tuinator::Point global_position);
    void begin_drag();
    void end_drag();
    bool route_mouse_to_child(const tuinator::MouseEvent& mouse);

    std::unique_ptr<tuinator::Widget> left_;
    std::unique_ptr<tuinator::Widget> right_;
    int default_left_width_;
    int left_width_;
    bool dragging_ = false;
    std::function<void(int width)> on_left_resize_;
};
