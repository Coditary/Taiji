#include "composer_palette.hpp"

#include "skim_fuzzy.hpp"
#include "ui_theme.hpp"

#include <tuinator/core/event.hpp>
#include <tuinator/render/glyphs.hpp>
#include <tuinator/render/text.hpp>

#include <algorithm>
#include <cctype>

namespace {

std::string repeat_utf8(std::string_view glyph, int count) {
    std::string out;
    for (int i = 0; i < count; ++i) {
        out += glyph;
    }
    return out;
}

constexpr int kPaletteMaxListRows = 10;

void sync_list_scroll(int selected, int max_visible, int total_visible, int& list_scroll) {
    if (max_visible <= 0) {
        list_scroll = 0;
        return;
    }
    if (selected < list_scroll) {
        list_scroll = selected;
    } else if (selected >= list_scroll + max_visible) {
        list_scroll = selected - max_visible + 1;
    }
    const int max_scroll = std::max(0, total_visible - max_visible);
    list_scroll = std::clamp(list_scroll, 0, max_scroll);
}

} // namespace

ComposerPalette::ComposerPalette(tuinator::Theme theme) : theme_(std::move(theme)) {}

void ComposerPalette::set_rows(std::vector<ComposerPaletteRow> rows) {
    rows_ = std::move(rows);
    rebuild_visible();
    selected_ = 0;
    flyout_parent_ = -1;
    mark_dirty();
    mark_layout_dirty();
}

void ComposerPalette::set_filter(std::string query) {
    if (filter_ == query) {
        return;
    }
    filter_ = std::move(query);
    rebuild_visible();
    selected_ = 0;
    flyout_parent_ = -1;
    mark_dirty();
}

void ComposerPalette::set_on_select(SelectHandler handler) {
    on_select_ = std::move(handler);
}

void ComposerPalette::set_on_close(std::function<void()> handler) {
    on_close_ = std::move(handler);
}

void ComposerPalette::open() {
    if (open_) {
        return;
    }
    open_ = true;
    open_anim_ms_ = 0;
    mark_dirty();
    mark_layout_dirty();
}

void ComposerPalette::close() {
    if (!open_) {
        return;
    }
    open_ = false;
    flyout_parent_ = -1;
    if (on_close_) {
        on_close_();
    }
    mark_dirty();
    mark_layout_dirty();
}

void ComposerPalette::tick(int dt_ms) {
    if (!open_) {
        return;
    }
    open_anim_ms_ += dt_ms;
    mark_dirty();
}

void ComposerPalette::rebuild_visible() {
    visible_.clear();
    std::vector<std::string> lines;
    std::vector<std::string> meta;
    lines.reserve(rows_.size());
    meta.reserve(rows_.size());
    for (const ComposerPaletteRow& row : rows_) {
        lines.push_back(row.title + "  " + row.subtitle + "  " + row.id);
        meta.push_back(row.id + " " + row.title + " " + row.subtitle + " " + row.section);
    }
    const std::vector<int> hits = skim_fuzzy_filter_indices(filter_, lines, meta, meta);
    for (int i : hits) {
        if (i >= 0 && i < static_cast<int>(rows_.size())) {
            visible_.push_back(i);
        }
    }
    if (selected_ >= static_cast<int>(visible_.size())) {
        selected_ = std::max(0, static_cast<int>(visible_.size()) - 1);
    }
    const int list_rows = std::min(kPaletteMaxListRows, static_cast<int>(visible_.size()));
    sync_list_scroll(selected_, list_rows, static_cast<int>(visible_.size()), list_scroll_);
}

int ComposerPalette::panel_height() const {
    if (!open_) {
        return 0;
    }
    const int row_count = std::max(1, static_cast<int>(visible_.empty() ? 1 : visible_.size()));
    const int visible = std::min(kPaletteMaxListRows, row_count);
    return std::min(kPaletteMaxListRows + 3, visible + 3);
}

tuinator::Size ComposerPalette::preferred_size() const {
    if (!open_) {
        return {0, 0};
    }
    return {42, panel_height()};
}

void ComposerPalette::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

tuinator::Rect ComposerPalette::main_panel_rect() const {
    const int h = panel_height();
    const int w = std::min(bounds_.width, 44);
    const int x = std::max(0, (bounds_.width - w) / 2);
    return {x, 0, w, h};
}

tuinator::Rect ComposerPalette::flyout_panel_rect() const {
    const tuinator::Rect main = main_panel_rect();
    if (flyout_parent_ < 0 || flyout_parent_ >= static_cast<int>(visible_.size())) {
        return {};
    }
    const int row_index = visible_[flyout_parent_];
    const ComposerPaletteRow& parent = rows_[static_cast<std::size_t>(row_index)];
    if (!parent.submenu) {
        return {};
    }
    int child_count = 0;
    for (const ComposerPaletteRow& row : rows_) {
        if (row.section == parent.id) {
            ++child_count;
        }
    }
    if (child_count <= 0) {
        return {};
    }
    const int fh = std::min(12, child_count + 2);
    const int fw = 28;
    int fx = main.x + main.width + 1;
    if (fx + fw > bounds_.width) {
        fx = std::max(0, main.x - fw - 1);
    }
    const int fy = main.y + 1 + flyout_parent_;
    return {fx, fy, fw, fh};
}

void ComposerPalette::paint(tuinator::PaintContext& ctx) const {
    if (!open_ || bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    const tuinator::Rect panel = main_panel_rect();
    if (panel.width <= 0 || panel.height <= 0) {
        return;
    }

    const tuinator::BorderGlyphs border = tuinator::unicode_rounded_border_glyphs();
    tuinator::Style frame = theme_.label;
    frame.background_rgb = ui_input_chrome_fill();
    frame.foreground_rgb = ui_border_chrome();
    ctx.canvas.draw_box(panel, frame, border);

    tuinator::Style search = ui_resolve_main_text(ui_secondary_style(theme_));
    search.background_rgb = ui_input_chrome_fill();
    tuinator::Style row_style = ui_resolve_main_text(theme_.label);
    row_style.background_rgb = ui_input_chrome_fill();
    tuinator::Style row_active = row_style;
    row_active.background_rgb = ui_list_active_bg();
    row_active.foreground_rgb = ui_list_active_fg();
    tuinator::Style hint = ui_resolve_main_text(ui_secondary_style(theme_));
    hint.background_rgb = ui_input_chrome_fill();

    int y = panel.y + 1;
    const int inner_x = panel.x + 1;
    const int inner_w = panel.width - 2;

    std::string search_line = "󰍉 Search skills, files, context…";
    if (!filter_.empty()) {
        search_line = "󰍉 " + filter_;
    }
    const std::size_t search_bytes = tuinator::text_byte_length_for_width(search_line, inner_w - 1);
    ctx.canvas.draw_text({inner_x, y}, search_line.substr(0, search_bytes), search);
    ++y;

    ctx.canvas.draw_text({inner_x, y}, repeat_utf8(u8"─", inner_w), hint);
    ++y;

    const int max_list_rows = std::min(kPaletteMaxListRows, panel.height - 3);
    for (int row = 0; row < max_list_rows; ++row) {
        const int vi = list_scroll_ + row;
        if (vi >= static_cast<int>(visible_.size()) || y >= panel.y + panel.height - 1) {
            break;
        }
        const ComposerPaletteRow& palette_row = rows_[static_cast<std::size_t>(visible_[vi])];
        const tuinator::Style& style = vi == selected_ ? row_active : row_style;

        if (!palette_row.section.empty() &&
            (vi == 0 || rows_[static_cast<std::size_t>(visible_[vi - 1])].section != palette_row.section)) {
            if (y > panel.y + 2) {
                ctx.canvas.draw_text({inner_x, y}, std::string(u8"├") + repeat_utf8(u8"─", std::max(0, inner_w - 1)), hint);
                ++y;
            }
        }

        std::string prefix = palette_row.icon.empty() ? "  " : palette_row.icon + " ";
        std::string line = prefix + palette_row.title;
        if (!palette_row.subtitle.empty()) {
            line += "  ";
        }
        const int title_max = inner_w - 8;
        const std::size_t title_bytes = tuinator::text_byte_length_for_width(line, title_max);
        ctx.canvas.draw_text({inner_x, y}, line.substr(0, title_bytes), style);

        if (!palette_row.subtitle.empty()) {
            const std::string sub = palette_row.subtitle;
            const int sub_x = inner_x + inner_w - tuinator::text_display_width(sub) - (palette_row.submenu ? 2 : 0);
            tuinator::Style sub_style = hint;
            sub_style.background = style.background;
            sub_style.background_rgb = style.background_rgb;
            ctx.canvas.draw_text({sub_x, y}, sub, sub_style);
        }
        if (palette_row.submenu) {
            ctx.canvas.draw_text({inner_x + inner_w - 2, y}, "▸", style);
        } else if (palette_row.checked) {
            ctx.canvas.draw_text({inner_x + inner_w - 2, y}, "✓", style);
        } else if (!palette_row.hint.empty()) {
            const int hx = inner_x + inner_w - tuinator::text_display_width(palette_row.hint);
            tuinator::Style hs = hint;
            hs.background_rgb = style.background_rgb;
            ctx.canvas.draw_text({hx, y}, palette_row.hint, hs);
        }
        ++y;
    }

    if (!visible_.empty()) {
        std::string scroll_text = std::to_string(selected_ + 1) + "/" + std::to_string(visible_.size());
        if (static_cast<int>(visible_.size()) > max_list_rows) {
            scroll_text += "  ↑↓";
        }
        const int caption_y = panel.y + panel.height - 1;
        const int text_w = tuinator::text_display_width(scroll_text);
        const int caption_x = panel.x + panel.width - 2 - text_w;
        tuinator::Style caption = frame;
        caption.dim = true;
        caption.foreground_rgb = ui_secondary_text();
        caption.background_rgb = ui_input_chrome_fill();
        ctx.canvas.fill_rect({caption_x - 1, caption_y, text_w + 2, 1}, ' ', frame);
        ctx.canvas.draw_text({caption_x, caption_y}, scroll_text, caption);
    }

    const tuinator::Rect fly = flyout_panel_rect();
    if (fly.width > 2 && fly.height > 2) {
        ctx.canvas.draw_box(fly, frame, border);
        int fy = fly.y + 1;
        const int row_index = visible_[flyout_parent_];
        const std::string section = rows_[static_cast<std::size_t>(row_index)].id;
        for (const ComposerPaletteRow& row : rows_) {
            if (row.section != section) {
                continue;
            }
            if (fy >= fly.y + fly.height - 1) {
                break;
            }
            std::string line = row.icon + " " + row.title;
            if (row.checked) {
                line = "[x] " + row.title;
            }
            const std::size_t bytes = tuinator::text_byte_length_for_width(line, fly.width - 2);
            ctx.canvas.draw_text({fly.x + 1, fy}, line.substr(0, bytes), row_style);
            ++fy;
        }
    }
}

bool ComposerPalette::handle_keypress(const tuinator::KeyPress& key) {
    if (!open_) {
        return false;
    }

    if (key.key == tuinator::Key::Escape) {
        close();
        return true;
    }
    if (key.key == tuinator::Key::Up) {
        selected_ = std::max(0, selected_ - 1);
        sync_list_scroll(selected_, std::min(kPaletteMaxListRows, static_cast<int>(visible_.size())),
                         static_cast<int>(visible_.size()), list_scroll_);
        mark_dirty();
        return true;
    }
    if (key.key == tuinator::Key::Down) {
        selected_ = std::min(static_cast<int>(visible_.size()) - 1, selected_ + 1);
        sync_list_scroll(selected_, std::min(kPaletteMaxListRows, static_cast<int>(visible_.size())),
                         static_cast<int>(visible_.size()), list_scroll_);
        mark_dirty();
        return true;
    }
    if (key.key == tuinator::Key::Right) {
        flyout_parent_ = selected_;
        mark_dirty();
        return true;
    }
    if (key.key == tuinator::Key::Left) {
        flyout_parent_ = -1;
        mark_dirty();
        return true;
    }
    if (key.key == tuinator::Key::Enter) {
        if (flyout_parent_ >= 0) {
            return true;
        }
        if (selected_ >= 0 && selected_ < static_cast<int>(visible_.size())) {
            const ComposerPaletteRow& row = rows_[static_cast<std::size_t>(visible_[selected_])];
            if (row.submenu) {
                flyout_parent_ = selected_;
                mark_dirty();
                return true;
            }
            if (on_select_) {
                on_select_(row.id);
            }
            return true;
        }
        return true;
    }
    if (key.key == tuinator::Key::Backspace) {
        if (!filter_.empty()) {
            filter_.pop_back();
            rebuild_visible();
            mark_dirty();
        }
        return true;
    }
    const std::string insert = tuinator::keypress_insert_text(key);
    if (!insert.empty() && insert[0] >= 32) {
        filter_ += insert;
        rebuild_visible();
        mark_dirty();
        return true;
    }
    return false;
}

bool ComposerPalette::handle_event(const tuinator::Event& event) {
    if (!open_) {
        return false;
    }

    if (const auto* key = std::get_if<tuinator::KeyPress>(&event)) {
        if (handle_keypress(*key)) {
            return true;
        }
    }

    const auto* mouse = std::get_if<tuinator::MouseEvent>(&event);
    if (mouse != nullptr && mouse->button == tuinator::MouseButton::Left &&
        mouse->action == tuinator::MouseAction::Release) {
        const tuinator::Point local{mouse->position.x - bounds_.x, mouse->position.y - bounds_.y};
        const tuinator::Rect fly = flyout_panel_rect();
        if (fly.contains(local)) {
            const int row_index = visible_[flyout_parent_];
            const std::string section = rows_[static_cast<std::size_t>(row_index)].id;
            int row_y = fly.y + 1;
            for (const ComposerPaletteRow& row : rows_) {
                if (row.section != section) {
                    continue;
                }
                if (local.y == row_y && on_select_) {
                    on_select_(row.id);
                    return true;
                }
                ++row_y;
            }
        }
        const tuinator::Rect panel = main_panel_rect();
        if (panel.contains(local)) {
            const int row_y = local.y - (panel.y + 2);
            if (row_y >= 0 && row_y < std::min(kPaletteMaxListRows, static_cast<int>(visible_.size()))) {
                selected_ = list_scroll_ + row_y;
                const ComposerPaletteRow& row = rows_[static_cast<std::size_t>(visible_[selected_])];
                if (row.submenu) {
                    flyout_parent_ = selected_;
                } else if (on_select_) {
                    on_select_(row.id);
                }
                mark_dirty();
                return true;
            }
        }
        close();
        return true;
    }

    return open_;
}

tuinator::Widget* ComposerPalette::hit_test(tuinator::Point point) {
    if (!open_ || !bounds_.contains(point)) {
        return nullptr;
    }
    return this;
}
