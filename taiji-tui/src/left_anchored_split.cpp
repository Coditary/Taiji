#include "left_anchored_split.hpp"

#include "ui_theme.hpp"

#include <algorithm>

namespace {

void paint_panel_background(tuinator::PaintContext& ctx, const tuinator::Rect& local_bounds, const tuinator::Style& bg) {
    if (local_bounds.width <= 0 || local_bounds.height <= 0) {
        return;
    }
    ctx.canvas.fill_rect(local_bounds, ' ', bg);
}

} // namespace

LeftAnchoredSplit::LeftAnchoredSplit(std::unique_ptr<tuinator::Widget> left, std::unique_ptr<tuinator::Widget> right,
                                     int left_width)
    : left_(std::move(left)), right_(std::move(right)), default_left_width_(left_width), left_width_(left_width) {
    attach_child_widget(left_.get());
    attach_child_widget(right_.get());
}

void LeftAnchoredSplit::set_left_width(int width) {
    const int clamped = clamped_left_width(width);
    if (left_width_ == clamped) {
        return;
    }
    left_width_ = clamped;
    mark_layout_dirty();
}

void LeftAnchoredSplit::set_on_left_resize(std::function<void(int width)> callback) {
    on_left_resize_ = std::move(callback);
}

int LeftAnchoredSplit::allowed_min_left() const {
    return std::max(16, default_left_width_ - kLeftAdjustDelta);
}

int LeftAnchoredSplit::allowed_max_left() const {
    return default_left_width_ + kLeftAdjustDelta;
}

int LeftAnchoredSplit::clamped_left_width(int /*width*/) const {
    const int target = default_left_width_;
    if (bounds_.width <= 0) {
        return target;
    }

    const int layout_max = std::max(16, bounds_.width - kRestMinCols - kDividerCols);
    return std::min(target, layout_max);
}

int LeftAnchoredSplit::right_width() const {
    if (bounds_.width <= 0) {
        return kRestMinCols;
    }
    return std::max(0, bounds_.width - clamped_left_width(left_width_) - kDividerCols);
}

tuinator::Rect LeftAnchoredSplit::divider_bounds() const {
    const int left_w = clamped_left_width(left_width_);
    return {bounds_.x + left_w, bounds_.y, kDividerCols, bounds_.height};
}

tuinator::Rect LeftAnchoredSplit::divider_hit_bounds() const {
    tuinator::Rect divider = divider_bounds();
    divider.x -= kDividerHitSlop;
    divider.width += kDividerHitSlop * 2 + kDividerCols;
    return divider;
}

bool LeftAnchoredSplit::contains_divider(tuinator::Point point) const { return divider_hit_bounds().contains(point); }

void LeftAnchoredSplit::update_left_width_from_mouse(tuinator::Point global_position) {
    if (bounds_.width <= 0) {
        return;
    }

    const int local_x = global_position.x - bounds_.x;
    const int clamped = clamped_left_width(local_x);
    if (clamped == left_width_) {
        return;
    }

    left_width_ = clamped;
    if (on_left_resize_) {
        on_left_resize_(left_width_);
    }
    mark_layout_dirty();
}

void LeftAnchoredSplit::begin_drag() {
    if (dragging_) {
        return;
    }
    dragging_ = true;
}

void LeftAnchoredSplit::end_drag() { dragging_ = false; }

void LeftAnchoredSplit::set_on_dirty(std::function<void(tuinator::Rect)> callback) {
    tuinator::Widget::set_on_dirty(std::move(callback));
    if (left_ != nullptr) {
        left_->set_on_dirty(on_dirty_);
    }
    if (right_ != nullptr) {
        right_->set_on_dirty(on_dirty_);
    }
}

void LeftAnchoredSplit::set_on_layout(std::function<void()> callback) {
    tuinator::Widget::set_on_layout(std::move(callback));
    if (left_ != nullptr) {
        left_->set_on_layout(on_layout_);
    }
    if (right_ != nullptr) {
        right_->set_on_layout(on_layout_);
    }
}

tuinator::Size LeftAnchoredSplit::preferred_size() const {
    const tuinator::Size left_size = left_ != nullptr ? left_->preferred_size() : tuinator::Size{};
    const tuinator::Size right_size = right_ != nullptr ? right_->preferred_size() : tuinator::Size{};
    return {
        std::max(left_width_, left_size.width) + kDividerCols + right_size.width,
        std::max(left_size.height, right_size.height),
    };
}

void LeftAnchoredSplit::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    left_width_ = clamped_left_width(left_width_);
    const int left_w = left_width_;
    const int right_w = right_width();

    if (left_ != nullptr) {
        left_->layout({bounds_.x, bounds_.y, left_w, bounds_.height});
    }
    if (right_ != nullptr) {
        right_->layout({bounds_.x + left_w + kDividerCols, bounds_.y, right_w, bounds_.height});
    }
}

void LeftAnchoredSplit::paint(tuinator::PaintContext& ctx) const {
    auto paint_child = [&](const tuinator::Widget* child) {
        if (child == nullptr) {
            return;
        }

        const tuinator::Rect local{
            child->bounds().x - bounds_.x,
            child->bounds().y - bounds_.y,
            child->bounds().width,
            child->bounds().height,
        };
        ctx.with_clip(local, [&](tuinator::PaintContext& child_ctx) { child->paint(child_ctx); });
    };

    if (left_ != nullptr) {
        const tuinator::Rect local{
            left_->bounds().x - bounds_.x,
            left_->bounds().y - bounds_.y,
            left_->bounds().width,
            left_->bounds().height,
        };
        paint_panel_background(ctx, local, ui_sidebar_panel_style());
    }
    if (right_ != nullptr) {
        const tuinator::Rect local{
            right_->bounds().x - bounds_.x,
            right_->bounds().y - bounds_.y,
            right_->bounds().width,
            right_->bounds().height,
        };
        paint_panel_background(ctx, local, ui_main_panel_style());
    }

    paint_child(left_.get());
    paint_child(right_.get());
}

tuinator::Widget* LeftAnchoredSplit::hit_test(tuinator::Point point) {
    if (!bounds_.contains(point)) {
        return nullptr;
    }

    if (left_ != nullptr && left_->bounds().contains(point)) {
        if (tuinator::Widget* hit = left_->hit_test(point)) {
            return hit;
        }
    }

    // Keep the nested split as the mouse target so its divider handling runs
    // before chat scroll/input widgets absorb the event.
    if (right_ != nullptr && right_->bounds().contains(point)) {
        return right_.get();
    }

    return this;
}

bool LeftAnchoredSplit::route_mouse_to_child(const tuinator::MouseEvent& mouse) {
    if (!bounds_.contains(mouse.position)) {
        return false;
    }

    if (left_ != nullptr && left_->bounds().contains(mouse.position)) {
        if (tuinator::Widget* hit = left_->hit_test(mouse.position)) {
            return hit->handle_event(mouse);
        }
    }

    if (right_ != nullptr && right_->bounds().contains(mouse.position)) {
        return right_->handle_event(mouse);
    }

    return false;
}

bool LeftAnchoredSplit::handle_event(const tuinator::Event& event) {
    if (const auto* mouse = std::get_if<tuinator::MouseEvent>(&event)) {
        if (route_mouse_to_child(*mouse)) {
            return true;
        }
    }

    for (tuinator::Widget* pane : {left_.get(), right_.get()}) {
        if (pane != nullptr && pane->has_focused_descendant() && pane->handle_event(event)) {
            return true;
        }
    }

    if (const auto* mouse = std::get_if<tuinator::MouseEvent>(&event)) {
        return route_mouse_to_child(*mouse);
    }

    return false;
}

tuinator::Widget* LeftAnchoredSplit::hit_test_focusable(tuinator::Point point) {
    if (!bounds_.contains(point)) {
        return nullptr;
    }

    if (contains_divider(point)) {
        return this;
    }

    for (tuinator::Widget* pane : {left_.get(), right_.get()}) {
        if (pane == nullptr) {
            continue;
        }
        if (tuinator::Widget* hit = pane->hit_test_focusable(point)) {
            return hit;
        }
    }

    return nullptr;
}

bool LeftAnchoredSplit::has_focused_descendant() const {
    return (left_ != nullptr && left_->has_focused_descendant()) ||
           (right_ != nullptr && right_->has_focused_descendant());
}

void LeftAnchoredSplit::collect_focusable(std::vector<tuinator::Widget*>& out) {
    if (left_ != nullptr) {
        left_->collect_focusable(out);
    }
    if (right_ != nullptr) {
        right_->collect_focusable(out);
    }
}

void LeftAnchoredSplit::for_each_child(const std::function<void(tuinator::Widget*)>& visitor) {
    if (left_ != nullptr) {
        visitor(left_.get());
    }
    if (right_ != nullptr) {
        visitor(right_.get());
    }
}

void LeftAnchoredSplit::for_each_descendant(const std::function<void(tuinator::Widget*)>& visitor) {
    if (left_ != nullptr) {
        visitor(left_.get());
        left_->for_each_descendant(visitor);
    }
    if (right_ != nullptr) {
        visitor(right_.get());
        right_->for_each_descendant(visitor);
    }
}
