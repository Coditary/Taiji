#include "sidebar_divider.hpp"

#include "ui_layout.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

SidebarDivider::SidebarDivider(tuinator::Theme theme) : theme_(std::move(theme)) {}

tuinator::Size SidebarDivider::preferred_size() const {
    return {8, 1};
}

void SidebarDivider::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void SidebarDivider::paint(tuinator::PaintContext& ctx) const {
    if (bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    paint_bounds_background(ctx, ui_sidebar_panel_style());

    const int pad = kUiPadX;
    const int line_w = std::max(0, bounds_.width - pad * 2);
    const std::string dash = u8"─";
    const int dash_w = std::max(1, tuinator::text_display_width(dash));
    std::string line;
    while (tuinator::text_display_width(line) + dash_w <= line_w) {
        line += dash;
    }
    tuinator::Style line_style = ui_secondary_style(theme_);
    line_style.background_rgb = kUiSidebarBackground;
    line_style.dim = true;
    ctx.canvas.draw_text({pad, 0}, line, ui_resolve_sidebar_text(ctx, *this, line_style));
}
