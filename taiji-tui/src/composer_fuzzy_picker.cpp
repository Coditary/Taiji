#include "composer_fuzzy_picker.hpp"

#include "skim_fuzzy.hpp"
#include "ui_keybindings.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/glyphs.hpp>
#include <tuinator/render/text.hpp>

#include <algorithm>

namespace {

constexpr int kPanelMaxWidth = 52;
constexpr int kPanelMaxListRows = 10;

void draw_bottom_border_caption(tuinator::PaintContext& ctx, const tuinator::Rect& panel, const std::string& text,
                                const tuinator::Style& frame) {
    if (panel.width <= 4 || text.empty()) {
        return;
    }
    const int y = panel.y + panel.height - 1;
    const int text_w = tuinator::text_display_width(text);
    const int x = panel.x + panel.width - 2 - text_w;
    tuinator::Style erase = frame;
    erase.background_rgb = ui_input_chrome_fill();
    ctx.canvas.fill_rect({x - 1, y, text_w + 2, 1}, ' ', erase);
    tuinator::Style label = frame;
    label.dim = true;
    label.foreground_rgb = ui_secondary_text();
    label.background_rgb = ui_input_chrome_fill();
    ctx.canvas.draw_text({x, y}, text, label);
}

} // namespace

ComposerFuzzyPicker::ComposerFuzzyPicker(tuinator::Theme theme) : theme_(std::move(theme)) {}

void ComposerFuzzyPicker::set_view(ComposerFuzzyPickerView view) {
    view_ = std::move(view);
    if (!view_.format_row_label) {
        view_.format_row_label = [](const ComposerFuzzyPickerItem& item) {
            return item.label.empty() ? item.line : item.label;
        };
    }
}

void ComposerFuzzyPicker::set_items(std::vector<ComposerFuzzyPickerItem> items) {
    items_ = std::move(items);
    if (open_) {
        rebuild_filter();
    }
}

void ComposerFuzzyPicker::rebuild_filter(bool preview) {
    filtered_.clear();
    std::vector<std::string> lines;
    std::vector<std::string> meta;
    lines.reserve(items_.size());
    meta.reserve(items_.size());
    for (const ComposerFuzzyPickerItem& item : items_) {
        lines.push_back(item.line);
        meta.push_back(item.meta);
    }
    filtered_ = skim_fuzzy_filter_indices(filter_, lines, meta, meta);
    auto mention_rank = [](const std::string& id) {
        if (id.rfind("actor.", 0) == 0) {
            return 0;
        }
        if (id.rfind("pack.", 0) == 0) {
            return 1;
        }
        if (id.rfind("file.", 0) == 0) {
            return 2;
        }
        return 1;
    };
    std::stable_sort(filtered_.begin(), filtered_.end(), [&](int a, int b) {
        const std::string& id_a = items_[static_cast<std::size_t>(a)].id;
        const std::string& id_b = items_[static_cast<std::size_t>(b)].id;
        return mention_rank(id_a) < mention_rank(id_b);
    });
    if (selected_ >= static_cast<int>(filtered_.size())) {
        selected_ = std::max(0, static_cast<int>(filtered_.size()) - 1);
    }
    sync_scroll(visible_list_rows(panel_height()));
    if (preview) {
        preview_selection();
    }
}

void ComposerFuzzyPicker::sync_scroll(int max_visible) {
    if (max_visible <= 0) {
        list_scroll_ = 0;
        return;
    }
    if (selected_ < list_scroll_) {
        list_scroll_ = selected_;
    } else if (selected_ >= list_scroll_ + max_visible) {
        list_scroll_ = selected_ - max_visible + 1;
    }
    const int max_scroll = std::max(0, static_cast<int>(filtered_.size()) - max_visible);
    list_scroll_ = std::clamp(list_scroll_, 0, max_scroll);
}

void ComposerFuzzyPicker::preview_selection() {
    if (!open_ || filtered_.empty() || selected_ < 0 || selected_ >= static_cast<int>(filtered_.size())) {
        return;
    }
    const ComposerFuzzyPickerItem& item = items_[static_cast<std::size_t>(filtered_[selected_])];
    if (on_preview_) {
        on_preview_(item.id);
    }
}

void ComposerFuzzyPicker::open() {
    open_ = true;
    filter_.clear();
    selected_ = 0;
    list_scroll_ = 0;
    rebuild_filter(false);
}

void ComposerFuzzyPicker::close() {
    open_ = false;
    filter_.clear();
    filtered_.clear();
    selected_ = 0;
    list_scroll_ = 0;
}

void ComposerFuzzyPicker::set_initial_selection_id(const std::string& id) {
    if (id.empty()) {
        return;
    }
    for (int row = 0; row < static_cast<int>(filtered_.size()); ++row) {
        const ComposerFuzzyPickerItem& item = items_[static_cast<std::size_t>(filtered_[row])];
        if (item.id == id) {
            selected_ = row;
            break;
        }
    }
    sync_scroll(visible_list_rows(panel_height()));
    preview_selection();
}

void ComposerFuzzyPicker::set_filter(const std::string& query) {
    if (filter_ == query) {
        return;
    }
    filter_ = query;
    selected_ = 0;
    list_scroll_ = 0;
    rebuild_filter();
}

const std::string& ComposerFuzzyPicker::selected_id() const {
    static const std::string kEmpty;
    if (!open_ || filtered_.empty() || selected_ < 0 || selected_ >= static_cast<int>(filtered_.size())) {
        return kEmpty;
    }
    return items_[static_cast<std::size_t>(filtered_[selected_])].id;
}

int ComposerFuzzyPicker::visible_list_rows(int panel_h) const {
    if (panel_h <= 0) {
        return 1;
    }
    return std::max(1, panel_h - 2 - view_.header_rows);
}

int ComposerFuzzyPicker::panel_height(int max_rows) const {
    if (!open_) {
        return 0;
    }
    const int row_count = std::max(1, static_cast<int>(filtered_.empty() ? 1 : filtered_.size()));
    const int visible = std::min(std::min(max_rows, kPanelMaxListRows), row_count);
    const int inner = view_.header_rows + visible;
    return std::min(kPanelMaxListRows + view_.header_rows + 2, inner + 2);
}

tuinator::Rect ComposerFuzzyPicker::panel_rect() const {
    const int h = panel_height();
    const int w = bounds_.width <= 0 ? kPanelMaxWidth : std::min(bounds_.width, kPanelMaxWidth);
    const int x = bounds_.width <= 0 ? 0 : std::max(0, (bounds_.width - w) / 2);
    return {x, 0, w, h};
}

int ComposerFuzzyPicker::list_start_y(const tuinator::Rect& panel) const {
    return panel.y + 1 + view_.header_rows;
}

tuinator::Size ComposerFuzzyPicker::preferred_size(int width, int max_rows) const {
    if (!open_) {
        return {0, 0};
    }
    return {width, panel_height(max_rows)};
}

void ComposerFuzzyPicker::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void ComposerFuzzyPicker::paint(tuinator::PaintContext& ctx) const {
    if (!open_ || bounds_.width <= 0 || bounds_.height <= 0) {
        return;
    }

    const tuinator::Rect panel = panel_rect();
    if (panel.width <= 2 || panel.height <= 2) {
        return;
    }

    const tuinator::BorderGlyphs border = tuinator::unicode_rounded_border_glyphs();
    tuinator::Style frame = theme_.label;
    frame.background_rgb = ui_input_chrome_fill();
    frame.foreground_rgb = ui_border_chrome();
    ctx.canvas.draw_box(panel, frame, border);

    const int inner_x = panel.x + 1;
    const int inner_y = panel.y + 1;
    const int inner_w = panel.width - 2;

    tuinator::Style panel_fill = ui_overlay_panel_style();
    panel_fill.background_rgb = ui_input_chrome_fill();
    ctx.canvas.fill_rect({inner_x, inner_y, inner_w, panel.height - 2}, ' ', panel_fill);

    tuinator::Style caption = ui_resolve_main_text(theme_.heading);
    caption.bold = true;
    caption.background_rgb = ui_input_chrome_fill();
    const std::string& picker_title = view_.title;
    ctx.canvas.draw_text({panel.x + 2, panel.y}, " " + picker_title + " ", caption);

    tuinator::Style row_style = ui_resolve_main_text(theme_.label);
    row_style.background_rgb = ui_input_chrome_fill();
    tuinator::Style row_active = row_style;
    row_active.background_rgb = ui_list_active_bg();
    row_active.foreground_rgb = ui_list_active_fg();

    int y = panel.y + 1;
    if (view_.paint_header) {
        view_.paint_header(ctx, {inner_x, y, inner_w, view_.header_rows}, y, inner_w, panel_fill);
        y += view_.header_rows;
    }

    const int max_rows = visible_list_rows(panel.height);
    const int list_y = y;

    if (filtered_.empty()) {
        tuinator::Style empty = row_style;
        empty.dim = true;
        ctx.canvas.fill_rect({inner_x, list_y, inner_w, 1}, ' ', panel_fill);
        ctx.canvas.draw_text({inner_x, list_y}, " " + view_.no_matches_text, empty);
        return;
    }

    for (int row = 0; row < max_rows; ++row) {
        const int idx = list_scroll_ + row;
        if (idx >= static_cast<int>(filtered_.size())) {
            break;
        }
        const ComposerFuzzyPickerItem& item = items_[static_cast<std::size_t>(filtered_[idx])];
        const bool active = idx == selected_;
        const tuinator::Style row_bg = active ? row_active : panel_fill;
        ctx.canvas.fill_rect({inner_x, list_y + row, inner_w, 1}, ' ', row_bg);

        tuinator::Style label_style = active ? row_active : row_style;
        const std::string label = view_.format_row_label(item);
        const std::size_t label_bytes = tuinator::text_byte_length_for_width(label, inner_w);
        ctx.canvas.draw_text({inner_x, list_y + row}, label.substr(0, label_bytes), label_style);
    }

    std::string scroll_text = std::to_string(selected_ + 1) + "/" + std::to_string(filtered_.size());
    if (static_cast<int>(filtered_.size()) > max_rows) {
        scroll_text += "  ↑↓";
    }
    scroll_text += view_.footer_suffix;
    draw_bottom_border_caption(ctx, panel, scroll_text, frame);
}

bool ComposerFuzzyPicker::move_selection(int delta) {
    if (!open_ || filtered_.empty()) {
        return false;
    }
    const int n = static_cast<int>(filtered_.size());
    selected_ = std::clamp(selected_ + delta, 0, n - 1);
    sync_scroll(visible_list_rows(panel_height()));
    preview_selection();
    return true;
}

bool ComposerFuzzyPicker::move_page(int rows) {
    return move_selection(rows);
}

bool ComposerFuzzyPicker::jump_selection(bool to_end) {
    if (!open_ || filtered_.empty()) {
        return false;
    }
    selected_ = to_end ? static_cast<int>(filtered_.size()) - 1 : 0;
    sync_scroll(visible_list_rows(panel_height()));
    preview_selection();
    return true;
}

bool ComposerFuzzyPicker::select_visible_row(int visible_index) {
    if (!open_ || visible_index < 0) {
        return false;
    }
    const int idx = list_scroll_ + visible_index;
    if (idx < 0 || idx >= static_cast<int>(filtered_.size())) {
        return false;
    }
    selected_ = idx;
    preview_selection();
    return true;
}

bool ComposerFuzzyPicker::hit_test_row(tuinator::Point local, int& visible_index) const {
    if (!open_ || bounds_.width <= 0 || bounds_.height <= 0) {
        return false;
    }
    const tuinator::Rect panel = panel_rect();
    if (local.x < panel.x || local.x >= panel.x + panel.width || local.y < panel.y ||
        local.y >= panel.y + panel.height) {
        return false;
    }
    const int list_y = list_start_y(panel);
    const int max_rows = visible_list_rows(panel.height);
    if (local.y < list_y || local.y >= list_y + max_rows) {
        return false;
    }
    visible_index = local.y - list_y;
    return true;
}

void ComposerFuzzyPicker::commit_selection() {
    const std::string& id = selected_id();
    if (id.empty()) {
        return;
    }
    if (on_commit_) {
        on_commit_(id);
    }
}

bool composer_fuzzy_picker_route_key(const tuinator::KeyPress& key, ComposerFuzzyPicker& picker,
                                   const std::function<void()>& on_escape,
                                   const std::function<void()>& on_confirm) {
    if (!picker.is_open()) {
        return false;
    }
    if (key.key == tuinator::Key::Escape) {
        if (on_escape) {
            on_escape();
        }
        return true;
    }
    if (key.key == tuinator::Key::Enter || (key.key == tuinator::Key::Tab && !key.shift)) {
        if (on_confirm) {
            on_confirm();
        }
        return true;
    }
    if (ui_key_theme_picker_list_prev(key)) {
        picker.move_selection(-1);
        return true;
    }
    if (ui_key_theme_picker_list_next(key)) {
        picker.move_selection(1);
        return true;
    }
    if (key.key == tuinator::Key::PageUp) {
        picker.move_page(-5);
        return true;
    }
    if (key.key == tuinator::Key::PageDown) {
        picker.move_page(5);
        return true;
    }
    if (key.key == tuinator::Key::Home) {
        picker.jump_selection(false);
        return true;
    }
    if (key.key == tuinator::Key::End) {
        picker.jump_selection(true);
        return true;
    }
    return false;
}
