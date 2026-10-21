#include "sidebar_footer.hpp"

#include "ui_layout.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

namespace {

constexpr const char* kVersionLabel = "taiji-tui v0.1";

} // namespace

SidebarFooter::SidebarFooter(tuinator::Theme theme) : theme_(std::move(theme)) {}

tuinator::Size SidebarFooter::preferred_size() const {
    return {12, 1};
}

void SidebarFooter::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void SidebarFooter::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    paint_bounds_background(ctx, ui_sidebar_panel_style());

    const int pad = kUiPadX;
    tuinator::Style version = ui_secondary_style(theme_);
    version.background_rgb = ui_sidebar_bg();
    version = ui_resolve_sidebar_text(ctx, *this, version);

    const int version_w = tuinator::text_display_width(kVersionLabel);
    const int version_x = std::max(pad, bounds_.width - pad - version_w);
    ctx.canvas.draw_text({version_x, 0}, kVersionLabel, version);
}

bool SidebarFooter::handle_event(const tuinator::Event& event) {
    const auto* mouse = std::get_if<tuinator::MouseEvent>(&event);
    if (mouse == nullptr) {
        return false;
    }
    return bounds_.contains(mouse->position);
}

tuinator::Widget* SidebarFooter::hit_test(tuinator::Point point) {
    if (!bounds_.contains(point)) {
        return nullptr;
    }
    return this;
}
