#include "composer_image_modal.hpp"

#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

#include <algorithm>
#include <cmath>

namespace {

tuinator::Rect centered_rect(tuinator::Size size, const tuinator::Rect& bounds) {
    const int x = bounds.x + std::max(0, (bounds.width - size.width) / 2);
    const int y = bounds.y + std::max(0, (bounds.height - size.height) / 2);
    return {x, y, size.width, size.height};
}

} // namespace

ComposerImageModal::ComposerImageModal(tuinator::Theme theme) : theme_(std::move(theme)) {}

void ComposerImageModal::open(const tuinator::TerminalImage& image, const std::string& caption) {
    image_ = image;
    caption_ = caption;
    open_ = true;
    mark_layout_dirty();
    mark_dirty();
}

void ComposerImageModal::close() {
    if (!open_) {
        return;
    }
    open_ = false;
    image_ = {};
    caption_.clear();
    image_bounds_ = {};
    mark_dirty();
}

tuinator::Size ComposerImageModal::display_cells_for(const tuinator::TerminalImage& image, int max_cols, int max_rows) {
    if (image.empty() || max_cols <= 0 || max_rows <= 0) {
        return {0, 0};
    }

    const int cap_w = std::max(4, max_cols);
    const int cap_h = std::max(3, max_rows);
    const double pixel_w = static_cast<double>(image.width());
    const double pixel_h = static_cast<double>(image.height());
    const double cell_aspect = 0.5; // terminal cells are roughly twice as tall as wide in pixels

    int cols = cap_w;
    int rows = std::max(1, static_cast<int>(std::lround((pixel_h / pixel_w) * cols / cell_aspect)));
    if (rows > cap_h) {
        rows = cap_h;
        cols = std::max(4, static_cast<int>(std::lround((pixel_w / pixel_h) * rows * cell_aspect)));
    }
    cols = std::min(cols, cap_w);
    rows = std::min(rows, cap_h);
    return {cols, rows};
}

tuinator::Size ComposerImageModal::preferred_size() const {
    if (bounds_.width > 0 && bounds_.height > 0) {
        return bounds_.size();
    }
    return {80, 24};
}

void ComposerImageModal::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
    if (!open_ || image_.empty()) {
        image_bounds_ = {};
        return;
    }

    const int reserved_caption = caption_.empty() ? 0 : 2;
    const tuinator::Size cells =
        display_cells_for(image_, std::max(8, bounds.width - 4), std::max(6, bounds.height - 4 - reserved_caption));
    const int total_h = cells.height + reserved_caption;
    image_bounds_ = centered_rect({cells.width, total_h}, bounds);
}

tuinator::Rect ComposerImageModal::image_bounds() const {
    return image_bounds_;
}

void ComposerImageModal::paint(tuinator::PaintContext& ctx) const {
    if (!open_ || image_.empty() || bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    ctx.canvas.apply_dimming_scrim(bounds_, image_bounds_, 0.82f);

    tuinator::Style caption_style = theme_.label;
    caption_style.foreground_rgb = ui_secondary_text();
    caption_style.background_rgb = ui_main_bg();

    int y = image_bounds_.y;
    if (!caption_.empty()) {
        const std::string line = " " + caption_ + " ";
        ctx.canvas.draw_text({image_bounds_.x, y}, line, caption_style);
        y += 1;
    }

    const tuinator::Rect draw_area{image_bounds_.x, y, image_bounds_.width, image_bounds_.height - (y - image_bounds_.y)};
    if (draw_area.width > 0 && draw_area.height > 0) {
        ctx.with_clip({draw_area.x - bounds_.x, draw_area.y - bounds_.y, draw_area.width, draw_area.height},
                      [&](tuinator::PaintContext& clipped) {
                          clipped.canvas.draw_image({draw_area.x - bounds_.x, draw_area.y - bounds_.y},
                                                    {draw_area.width, draw_area.height}, image_);
                      });
    }

    tuinator::Style hint = caption_style;
    hint.dim = true;
    const std::string hint_text = " Esc / q / click outside to close ";
    const int hint_w = tuinator::text_display_width(hint_text);
    const int hint_x = bounds_.x + std::max(0, (bounds_.width - hint_w) / 2);
    const int hint_y = bounds_.y + bounds_.height - 1;
    if (hint_y >= bounds_.y) {
        ctx.canvas.draw_text({hint_x, hint_y}, hint_text, hint);
    }
}

bool ComposerImageModal::handle_event(const tuinator::Event& event) {
    if (!open_) {
        return false;
    }

    if (const auto* key = std::get_if<tuinator::KeyPress>(&event)) {
        if (key->key == tuinator::Key::Escape) {
            close();
            return true;
        }
        if (!key->ctrl && key->character == 'q') {
            close();
            return true;
        }
    }

    if (const auto* mouse = std::get_if<tuinator::MouseEvent>(&event)) {
        if (mouse->button == tuinator::MouseButton::Left && mouse->action == tuinator::MouseAction::Release) {
            if (!image_bounds_.contains(mouse->position)) {
                close();
                return true;
            }
        }
        return bounds_.contains(mouse->position);
    }

    return true;
}

tuinator::Widget* ComposerImageModal::hit_test(tuinator::Point point) {
    if (!open_ || !bounds_.contains(point)) {
        return nullptr;
    }
    return this;
}

tuinator::Widget* ComposerImageModal::hit_test_focusable(tuinator::Point point) {
    return hit_test(point);
}
