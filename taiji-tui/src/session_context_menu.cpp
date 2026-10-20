#include "session_context_menu.hpp"

#include "ui_theme.hpp"

#include <algorithm>

namespace {

int text_width(std::string_view text) {
    return static_cast<int>(text.size());
}

} // namespace

SessionContextMenu::SessionContextMenu(tuinator::Theme theme) : theme_(std::move(theme)) {}

void SessionContextMenu::set_items(std::vector<Item> items) {
    items_ = std::move(items);
    close();
}

void SessionContextMenu::set_on_open_changed(std::function<void(bool open)> callback) {
    on_open_changed_ = std::move(callback);
}

tuinator::Point SessionContextMenu::to_local(tuinator::Point point) const {
    return {point.x - bounds_.x, point.y - bounds_.y};
}

void SessionContextMenu::show(tuinator::Point position) {
    anchor_ = to_local(position);
    open_ = true;
    active_item_ = 0;
    int row = 0;
    if (hit_panel(anchor_, row) && row >= 0 && row < static_cast<int>(items_.size())) {
        active_item_ = row;
    }
    if (on_open_changed_) {
        on_open_changed_(true);
    }
    mark_layout_dirty();
}

void SessionContextMenu::close() {
    if (!open_) {
        return;
    }
    open_ = false;
    active_item_ = 0;
    if (on_open_changed_) {
        on_open_changed_(false);
    }
    mark_layout_dirty();
}

SessionContextMenu::PanelLayout SessionContextMenu::panel_layout() const {
    PanelLayout panel;
    if (!open_ || items_.empty()) {
        return panel;
    }

    int width = 10;
    for (const Item& item : items_) {
        width = std::max(width, text_width(item.label) + 4);
    }
    panel.width = width + 2;
    panel.height = static_cast<int>(items_.size()) + 2;
    panel.x = anchor_.x;
    panel.y = anchor_.y;

    if (bounds_.width > 0 && panel.x + panel.width > bounds_.width) {
        panel.x = std::max(0, bounds_.width - panel.width);
    }
    if (bounds_.height > 0 && panel.y + panel.height > bounds_.height) {
        panel.y = std::max(0, bounds_.height - panel.height);
    }
    return panel;
}

tuinator::Rect SessionContextMenu::panel_bounds() const {
    const PanelLayout panel = panel_layout();
    if (panel.width <= 0 || panel.height <= 0) {
        return {};
    }
    return {bounds_.x + panel.x, bounds_.y + panel.y, panel.width, panel.height};
}

tuinator::Size SessionContextMenu::preferred_size() const {
    const PanelLayout panel = panel_layout();
    if (!open_) {
        return {0, 0};
    }
    return {panel.width, panel.height};
}

void SessionContextMenu::paint(tuinator::PaintContext& ctx) const {
    if (!open_ || items_.empty()) {
        return;
    }

    const PanelLayout panel = panel_layout();
    if (panel.width <= 0 || panel.height <= 0) {
        return;
    }

    tuinator::Canvas& canvas = ctx.canvas;
    const tuinator::Rect box{panel.x, panel.y, panel.width, panel.height};

    const tuinator::Style panel_bg = ui_overlay_panel_style();

    tuinator::Style border_style = panel_bg;
    border_style.foreground_rgb = ui_border_chrome();

    canvas.with_clip(box, [&](tuinator::Canvas& clipped) {
        const tuinator::Rect inner{0, 0, panel.width, panel.height};
        clipped.fill_rect(inner, ' ', panel_bg);
        clipped.draw_box(inner, border_style, ui_frame_border_glyphs());

        const int inner_width = std::max(0, panel.width - 2);
        for (int row = 0; row < static_cast<int>(items_.size()); ++row) {
            const Item& item = items_[static_cast<std::size_t>(row)];
            const int y = 1 + row;
            const bool selected = row == active_item_;

            tuinator::Style row_style = ui_list_selection_style(selected);
            if (item.label == "Delete") {
                row_style.foreground_rgb = ui_danger();
            }
            if (inner_width > 0) {
                clipped.fill_rect({1, y, inner_width, 1}, ' ', row_style);
            }

            const std::string prefix = selected ? "> " : "  ";
            clipped.draw_text({1, y}, prefix + item.label, row_style);
        }
    });
}

bool SessionContextMenu::hit_panel(tuinator::Point local, int& row) const {
    const PanelLayout panel = panel_layout();
    if (local.x >= panel.x && local.x < panel.x + panel.width && local.y >= panel.y &&
        local.y < panel.y + panel.height) {
        row = local.y - panel.y - 1;
        return true;
    }
    return false;
}

void SessionContextMenu::set_active_item(int index) {
    if (index < 0 || index >= static_cast<int>(items_.size()) || index == active_item_) {
        return;
    }
    active_item_ = index;
    mark_panel_dirty();
}

void SessionContextMenu::mark_panel_dirty() {
    const tuinator::Rect panel = panel_bounds();
    if (panel.width <= 0 || panel.height <= 0) {
        mark_dirty();
        return;
    }
    if (on_dirty_) {
        on_dirty_(panel);
        return;
    }
    mark_dirty();
}

tuinator::Widget* SessionContextMenu::hit_test(tuinator::Point point) {
    if (!open_ || !bounds_.contains(point)) {
        return nullptr;
    }
    return this;
}

void SessionContextMenu::activate_item(int index) {
    if (index < 0 || index >= static_cast<int>(items_.size())) {
        return;
    }
    const auto action = items_[static_cast<std::size_t>(index)].action;
    close();
    if (action) {
        action();
    }
}

bool SessionContextMenu::handle_event(const tuinator::Event& event) {
    if (!open_) {
        return false;
    }

    if (const auto* mouse = std::get_if<tuinator::MouseEvent>(&event)) {
        const tuinator::Point local = to_local(mouse->position);

        if (mouse->action == tuinator::MouseAction::Move) {
            int row = 0;
            if (hit_panel(local, row) && row >= 0 && row < static_cast<int>(items_.size())) {
                set_active_item(row);
            }
            return true;
        }

        if (mouse->action != tuinator::MouseAction::Click && mouse->action != tuinator::MouseAction::Release) {
            return true;
        }

        int row = 0;
        if (!hit_panel(local, row)) {
            close();
            return true;
        }

        if (row >= 0 && row < static_cast<int>(items_.size())) {
            set_active_item(row);
            activate_item(row);
        }
        return true;
    }

    const auto* key = std::get_if<tuinator::KeyPress>(&event);
    if (!key) {
        return true;
    }

    if (key->key == tuinator::Key::Escape) {
        close();
        return true;
    }
    if (key->key == tuinator::Key::Up) {
        if (!items_.empty()) {
            set_active_item((active_item_ - 1 + static_cast<int>(items_.size())) % static_cast<int>(items_.size()));
        }
        return true;
    }
    if (key->key == tuinator::Key::Down) {
        if (!items_.empty()) {
            set_active_item((active_item_ + 1) % static_cast<int>(items_.size()));
        }
        return true;
    }
    if (key->key == tuinator::Key::Enter) {
        activate_item(active_item_);
        return true;
    }

    return true;
}
