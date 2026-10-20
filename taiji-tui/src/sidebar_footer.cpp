#include "sidebar_footer.hpp"

#include "ui_layout.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

namespace {

constexpr const char* kVersionLabel = "taiji-tui v0.1";

} // namespace

SidebarFooter::SidebarFooter(tuinator::Theme theme, std::function<void()> on_settings)
    : theme_(std::move(theme)), on_settings_(std::move(on_settings)) {}

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
    tuinator::Style settings = ui_accent_on_sidebar();
    if (settings_hover_) {
        settings.bold = true;
    }

    tuinator::Style version = ui_secondary_style(theme_);
    version.background_rgb = ui_sidebar_bg();
    version = ui_resolve_sidebar_text(ctx, *this, version);

    tuinator::Style sep = ui_secondary_style(theme_);
    sep.background_rgb = ui_sidebar_bg();
    sep = ui_resolve_sidebar_text(ctx, *this, sep);

    const std::string settings_label = std::string(kNfSettings) + " Settings";
    ctx.canvas.draw_text({pad, 0}, settings_label, settings);

    const int version_w = tuinator::text_display_width(kVersionLabel);
    const int sep_w = tuinator::text_display_width(" │ ");
    const int version_x = std::max(pad, bounds_.width - pad - version_w);
    const int sep_x = std::max(pad, version_x - sep_w);
    if (sep_x > pad + tuinator::text_display_width(settings_label) + 1) {
        ctx.canvas.draw_text({sep_x, 0}, " │ ", sep);
    }
    ctx.canvas.draw_text({version_x, 0}, kVersionLabel, version);
}

bool SidebarFooter::handle_event(const tuinator::Event& event) {
    const auto* mouse = std::get_if<tuinator::MouseEvent>(&event);
    if (mouse == nullptr) {
        return false;
    }
    const int pad = kUiPadX;
    const std::string settings_label = std::string(kNfSettings) + " Settings";
    const int settings_w = tuinator::text_display_width(settings_label);
    const tuinator::Point local{mouse->position.x - bounds_.x, mouse->position.y - bounds_.y};
    const bool in_settings =
        local.y == 0 && local.x >= pad && local.x < pad + settings_w && bounds_.contains(mouse->position);

    if (mouse->action == tuinator::MouseAction::Move) {
        if (in_settings != settings_hover_) {
            settings_hover_ = in_settings;
            mark_dirty();
        }
        return bounds_.contains(mouse->position);
    }

    if (mouse->button == tuinator::MouseButton::Left && mouse->action == tuinator::MouseAction::Release &&
        in_settings) {
        if (on_settings_) {
            on_settings_();
        }
        return true;
    }
    return bounds_.contains(mouse->position);
}

tuinator::Widget* SidebarFooter::hit_test(tuinator::Point point) {
    if (!bounds_.contains(point)) {
        return nullptr;
    }
    return this;
}
