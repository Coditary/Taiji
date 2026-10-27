#include "ui_panel_box.hpp"

#include "ui_theme.hpp"

namespace {

bool vbox_child_occupies_space(const tuinator::Widget& child) { return child.preferred_size().height > 0; }

int vbox_gap_after(const tuinator::VBox& box, std::size_t child_index) {
    const auto& children = box.children();
    if (child_index + 1 >= children.size()) {
        return 0;
    }
    if (!vbox_child_occupies_space(*children[child_index]) ||
        !vbox_child_occupies_space(*children[child_index + 1])) {
        return 0;
    }
    return box.gap();
}

int collapsed_vbox_total_gap(const tuinator::VBox& box) {
    const auto& children = box.children();
    if (children.size() <= 1) {
        return 0;
    }
    int total = 0;
    for (std::size_t i = 0; i + 1 < children.size(); ++i) {
        total += vbox_gap_after(box, i);
    }
    return total;
}

void layout_main_panel_vertical(MainPanelVBox& box, tuinator::Rect bounds) {
    const int gap = box.gap();
    const int padding = box.padding();
    const int content_width = std::max(0, bounds.width - padding * 2);
    const int content_height = std::max(0, bounds.height - padding * 2);

    struct Row {
        tuinator::Widget* widget = nullptr;
        tuinator::Size preferred{};
        int flex = 0;
    };

    const auto& children = box.children();
    std::vector<Row> rows;
    rows.reserve(children.size());
    for (const auto& child : children) {
        rows.push_back({child.get(), child->preferred_size(), child->flex()});
    }

    int fixed_height = 0;
    int flex_total = 0;
    for (const Row& row : rows) {
        if (row.flex > 0) {
            flex_total += row.flex;
        } else {
            fixed_height += row.preferred.height;
        }
    }

    const int remaining = content_height - fixed_height - collapsed_vbox_total_gap(box);
    const int extra_per_flex = flex_total > 0 ? std::max(0, remaining) / flex_total : 0;
    int leftover = flex_total > 0 ? std::max(0, remaining) % flex_total : 0;

    int y = bounds.y + padding;
    for (std::size_t row_index = 0; row_index < rows.size(); ++row_index) {
        const Row& row = rows[row_index];
        int height = row.preferred.height;
        if (row.flex > 0) {
            height = std::max(row.preferred.height, extra_per_flex * row.flex);
            if (leftover > 0) {
                height += 1;
                leftover -= 1;
            }
        }

        row.widget->layout({bounds.x + padding, y, content_width, height});
        y += height;
        if (row_index + 1 < rows.size()) {
            y += vbox_gap_after(box, row_index);
        }
    }
}

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

tuinator::Size MainPanelVBox::preferred_size() const {
    int width = 0;
    int height = padding() * 2;
    const auto& children = this->children();
    for (std::size_t i = 0; i < children.size(); ++i) {
        const tuinator::Size child_size = children[i]->preferred_size();
        width = std::max(width, child_size.width);
        height += child_size.height;
        if (i + 1 < children.size()) {
            height += vbox_gap_after(*this, i);
        }
    }
    return {width + padding() * 2, height};
}

void MainPanelVBox::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
    layout_main_panel_vertical(*this, bounds);
}

void MainPanelVBox::paint(tuinator::PaintContext& ctx) const {
    ui_fill_widget_bounds(ctx, *this, ui_main_bg());
    paint_box_children(ctx, *this);
}

SidebarPanelVBox::SidebarPanelVBox(tuinator::BoxOptions options) : VBox(options) {}

void SidebarPanelVBox::paint(tuinator::PaintContext& ctx) const {
    ui_fill_widget_bounds(ctx, *this, ui_sidebar_bg());
    paint_box_children(ctx, *this);
}
