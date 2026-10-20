#include "settings_gear.hpp"

#include <tuinator/render/text.hpp>

SettingsGearButton::SettingsGearButton(tuinator::Style normal_style, tuinator::Style hover_style,
                                       std::function<void()> on_click)
    : normal_style_(std::move(normal_style)), hover_style_(std::move(hover_style)), on_click_(std::move(on_click)) {}

tuinator::Size SettingsGearButton::preferred_size() const {
    const int w = tuinator::text_display_width(kLabel);
    return {w, 1};
}

void SettingsGearButton::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void SettingsGearButton::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }
    const tuinator::Style base = hover_ ? hover_style_ : normal_style_;
    paint_bounds_background(ctx, base);
    const tuinator::Style style = ctx.styles().text(*this, base);
    ctx.canvas.draw_text({0, 0}, kLabel, style);
}

bool SettingsGearButton::handle_event(const tuinator::Event& event) {
    if (const auto* mouse = std::get_if<tuinator::MouseEvent>(&event)) {
        if (mouse->action == tuinator::MouseAction::Move) {
            const bool inside = bounds_.contains(mouse->position);
            if (inside != hover_) {
                hover_ = inside;
                mark_dirty();
            }
            return inside;
        }
        if (mouse->button != tuinator::MouseButton::Left) {
            return bounds_.contains(mouse->position);
        }
        if ((mouse->action == tuinator::MouseAction::Press || mouse->action == tuinator::MouseAction::Click ||
             mouse->action == tuinator::MouseAction::Release) &&
            bounds_.contains(mouse->position)) {
            if (mouse->action == tuinator::MouseAction::Release) {
                if (on_click_) {
                    on_click_();
                }
            }
            return true;
        }
    }
    return false;
}

tuinator::Widget* SettingsGearButton::hit_test(tuinator::Point point) {
    if (!bounds_.contains(point)) {
        return nullptr;
    }
    return this;
}
