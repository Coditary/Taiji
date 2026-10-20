#include "ui_panel_box.hpp"

#include "ui_theme.hpp"

namespace {

void paint_box_children(tuinator::PaintContext& ctx, const tuinator::Widget& box) {
    for (const auto& child : box.children()) {
        const tuinator::Rect local{
            child->bounds().x - box.bounds().x,
            child->bounds().y - box.bounds().y,
            child->bounds().width,
            child->bounds().height,
        };
        ctx.with_clip(local, [&](tuinator::PaintContext& child_ctx) { child->paint(child_ctx); });
    }
}

} // namespace

MainPanelVBox::MainPanelVBox(tuinator::BoxOptions options) : VBox(options) {}

void MainPanelVBox::paint(tuinator::PaintContext& ctx) const {
    ui_fill_widget_bounds(ctx, *this, kUiMainBackground);
    paint_box_children(ctx, *this);
}

SidebarPanelVBox::SidebarPanelVBox(tuinator::BoxOptions options) : VBox(options) {}

void SidebarPanelVBox::paint(tuinator::PaintContext& ctx) const {
    ui_fill_widget_bounds(ctx, *this, kUiSidebarBackground);
    paint_box_children(ctx, *this);
}
