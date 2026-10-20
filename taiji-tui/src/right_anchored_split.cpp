#include "right_anchored_split.hpp"

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

RightAnchoredSplit::RightAnchoredSplit(std::unique_ptr<tuinator::Widget> main,
                                       std::unique_ptr<tuinator::Widget> sidebar, int sidebar_width)
    : main_(std::move(main)), sidebar_(std::move(sidebar)), default_sidebar_width_(sidebar_width),
      sidebar_width_(sidebar_width) {
    attach_child_widget(main_.get());
    attach_child_widget(sidebar_.get());
}

void RightAnchoredSplit::set_sidebar_width(int width) {
    const int clamped = clamped_sidebar_width(width);
    if (sidebar_width_ == clamped) {
        return;
    }
    sidebar_width_ = clamped;
    mark_layout_dirty();
}

void RightAnchoredSplit::set_on_sidebar_resize(std::function<void(int width)> callback) {
    on_sidebar_resize_ = std::move(callback);
}

int RightAnchoredSplit::allowed_min_sidebar() const {
    return std::max(kSidebarAbsMin, default_sidebar_width_ - kSidebarAdjustDelta);
}

int RightAnchoredSplit::allowed_max_sidebar() const {
    return default_sidebar_width_ + kSidebarAdjustDelta;
}

int RightAnchoredSplit::clamped_sidebar_width(int /*width*/) const {
    const int target = default_sidebar_width_;
    if (bounds_.width <= 0) {
        return target;
    }

    const int layout_hi = bounds_.width - kMainAbsMin - kDividerCols;
    return std::clamp(target, kSidebarAbsMin, std::max(kSidebarAbsMin, layout_hi));
}

int RightAnchoredSplit::main_width() const {
    if (bounds_.width <= 0) {
        return kMainAbsMin;
    }
    return std::max(0, bounds_.width - clamped_sidebar_width(sidebar_width_) - kDividerCols);
}

tuinator::Rect RightAnchoredSplit::divider_bounds() const {
    const int sidebar_w = clamped_sidebar_width(sidebar_width_);
    const int divider_x = bounds_.x + bounds_.width - sidebar_w - kDividerCols;
    return {divider_x, bounds_.y, kDividerCols, bounds_.height};
}

tuinator::Rect RightAnchoredSplit::divider_hit_bounds() const {
    tuinator::Rect divider = divider_bounds();
    divider.x -= kDividerHitSlop;
    divider.width += kDividerHitSlop * 2 + kDividerCols;
    return divider;
}

bool RightAnchoredSplit::contains_divider(tuinator::Point point) const { return divider_hit_bounds().contains(point); }

void RightAnchoredSplit::update_sidebar_width_from_mouse(tuinator::Point global_position) {
    if (bounds_.width <= 0) {
        return;
    }

    const int local_x = global_position.x - bounds_.x;
    const int raw_width = bounds_.width - local_x - kDividerCols;
    const int clamped = clamped_sidebar_width(raw_width);
    if (clamped == sidebar_width_) {
        return;
    }

    sidebar_width_ = clamped;
    if (on_sidebar_resize_) {
        on_sidebar_resize_(sidebar_width_);
    }
    mark_layout_dirty();
}

void RightAnchoredSplit::begin_drag() {
    if (dragging_) {
        return;
    }
    dragging_ = true;
}

void RightAnchoredSplit::end_drag() { dragging_ = false; }

void RightAnchoredSplit::set_on_dirty(std::function<void(tuinator::Rect)> callback) {
    tuinator::Widget::set_on_dirty(std::move(callback));
    if (main_ != nullptr) {
        main_->set_on_dirty(on_dirty_);
    }
    if (sidebar_ != nullptr) {
        sidebar_->set_on_dirty(on_dirty_);
    }
}

void RightAnchoredSplit::set_on_layout(std::function<void()> callback) {
    tuinator::Widget::set_on_layout(std::move(callback));
    if (main_ != nullptr) {
        main_->set_on_layout(on_layout_);
    }
    if (sidebar_ != nullptr) {
        sidebar_->set_on_layout(on_layout_);
    }
}

tuinator::Size RightAnchoredSplit::preferred_size() const {
    const tuinator::Size main_size = main_ != nullptr ? main_->preferred_size() : tuinator::Size{};
    const tuinator::Size sidebar_size = sidebar_ != nullptr ? sidebar_->preferred_size() : tuinator::Size{};
    return {
        main_size.width + kDividerCols + std::max(sidebar_width_, sidebar_size.width),
        std::max(main_size.height, sidebar_size.height),
    };
}

void RightAnchoredSplit::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    sidebar_width_ = clamped_sidebar_width(sidebar_width_);
    const int sidebar_w = sidebar_width_;
    const int main_w = main_width();

    if (main_ != nullptr) {
        main_->layout({bounds_.x, bounds_.y, main_w, bounds_.height});
    }
    if (sidebar_ != nullptr) {
        sidebar_->layout({bounds_.x + main_w + kDividerCols, bounds_.y, sidebar_w, bounds_.height});
    }
}

void RightAnchoredSplit::paint(tuinator::PaintContext& ctx) const {
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

    if (main_ != nullptr) {
        const tuinator::Rect local{
            main_->bounds().x - bounds_.x,
            main_->bounds().y - bounds_.y,
            main_->bounds().width,
            main_->bounds().height,
        };
        paint_panel_background(ctx, local, ui_main_panel_style());
    }
    if (sidebar_ != nullptr) {
        const tuinator::Rect local{
            sidebar_->bounds().x - bounds_.x,
            sidebar_->bounds().y - bounds_.y,
            sidebar_->bounds().width,
            sidebar_->bounds().height,
        };
        paint_panel_background(ctx, local, ui_sidebar_panel_style());
    }

    paint_child(main_.get());
    paint_child(sidebar_.get());
}

tuinator::Widget* RightAnchoredSplit::hit_test(tuinator::Point point) {
    if (!bounds_.contains(point)) {
        return nullptr;
    }

    if (main_ != nullptr && main_->bounds().contains(point)) {
        if (tuinator::Widget* hit = main_->hit_test(point)) {
            return hit;
        }
    }

    if (sidebar_ != nullptr && sidebar_->bounds().contains(point)) {
        if (tuinator::Widget* hit = sidebar_->hit_test(point)) {
            return hit;
        }
    }

    return this;
}

bool RightAnchoredSplit::route_mouse_to_child(const tuinator::MouseEvent& mouse) {
    if (!bounds_.contains(mouse.position)) {
        return false;
    }

    if (main_ != nullptr && main_->bounds().contains(mouse.position)) {
        if (tuinator::Widget* hit = main_->hit_test(mouse.position)) {
            return hit->handle_event(mouse);
        }
    }

    if (sidebar_ != nullptr && sidebar_->bounds().contains(mouse.position)) {
        if (tuinator::Widget* hit = sidebar_->hit_test(mouse.position)) {
            return hit->handle_event(mouse);
        }
    }

    return false;
}

bool RightAnchoredSplit::handle_event(const tuinator::Event& event) {
    if (const auto* mouse = std::get_if<tuinator::MouseEvent>(&event)) {
        if (route_mouse_to_child(*mouse)) {
            return true;
        }
    }

    for (tuinator::Widget* pane : {main_.get(), sidebar_.get()}) {
        if (pane != nullptr && pane->has_focused_descendant() && pane->handle_event(event)) {
            return true;
        }
    }

    if (const auto* mouse = std::get_if<tuinator::MouseEvent>(&event)) {
        return route_mouse_to_child(*mouse);
    }

    return false;
}

tuinator::Widget* RightAnchoredSplit::hit_test_focusable(tuinator::Point point) {
    if (!bounds_.contains(point)) {
        return nullptr;
    }

    if (contains_divider(point)) {
        return this;
    }

    for (tuinator::Widget* pane : {main_.get(), sidebar_.get()}) {
        if (pane == nullptr) {
            continue;
        }
        if (tuinator::Widget* hit = pane->hit_test_focusable(point)) {
            return hit;
        }
    }

    return nullptr;
}

bool RightAnchoredSplit::has_focused_descendant() const {
    return (main_ != nullptr && main_->has_focused_descendant()) ||
           (sidebar_ != nullptr && sidebar_->has_focused_descendant());
}

void RightAnchoredSplit::collect_focusable(std::vector<tuinator::Widget*>& out) {
    if (main_ != nullptr) {
        main_->collect_focusable(out);
    }
    if (sidebar_ != nullptr) {
        sidebar_->collect_focusable(out);
    }
}

void RightAnchoredSplit::for_each_child(const std::function<void(tuinator::Widget*)>& visitor) {
    if (main_ != nullptr) {
        visitor(main_.get());
    }
    if (sidebar_ != nullptr) {
        visitor(sidebar_.get());
    }
}

void RightAnchoredSplit::for_each_descendant(const std::function<void(tuinator::Widget*)>& visitor) {
    if (main_ != nullptr) {
        visitor(main_.get());
        main_->for_each_descendant(visitor);
    }
    if (sidebar_ != nullptr) {
        visitor(sidebar_.get());
        sidebar_->for_each_descendant(visitor);
    }
}
