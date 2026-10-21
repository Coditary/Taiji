#include "composer_theme_picker.hpp"

#include "skim_fuzzy.hpp"
#include "ui_locale.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/glyphs.hpp>
#include <tuinator/render/text.hpp>

#include <algorithm>

namespace {

constexpr int kPanelMaxWidth = 52;
constexpr int kPanelMaxListRows = 10;

int header_row_count(bool draft_saved) {
    return draft_saved ? 1 : 0;
}

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

ComposerThemePicker::ComposerThemePicker(tuinator::Theme theme) : theme_(std::move(theme)) {}

void ComposerThemePicker::rebuild_entries() {
    entries_.clear();
    const std::vector<std::string> ids = ui_list_theme_ids();
    for (const std::string& id : ids) {
        Entry entry;
        entry.id = id;
        if (ui_is_bundled_theme(id)) {
            entry.subtitle = ui_theme_display_name(id);
        } else {
            entry.subtitle = "user theme";
        }
        entry.line = id + "  ·  " + entry.subtitle;
        entry.meta = id + " " + entry.subtitle + " theme color palette stylesheet";
        entries_.push_back(std::move(entry));
    }
}

void ComposerThemePicker::rebuild_filter(bool preview) {
    filtered_.clear();
    std::vector<std::string> lines;
    std::vector<std::string> meta;
    lines.reserve(entries_.size());
    meta.reserve(entries_.size());
    for (const Entry& e : entries_) {
        lines.push_back(e.line);
        meta.push_back(e.meta);
    }
    filtered_ = skim_fuzzy_filter_indices(filter_, lines, meta, meta);
    if (selected_ >= static_cast<int>(filtered_.size())) {
        selected_ = std::max(0, static_cast<int>(filtered_.size()) - 1);
    }
    sync_scroll(visible_list_rows(panel_height()));
    if (preview) {
        preview_selection();
    }
}

void ComposerThemePicker::sync_scroll(int max_visible) {
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

void ComposerThemePicker::preview_selection() {
    if (!open_ || filtered_.empty() || selected_ < 0 || selected_ >= static_cast<int>(filtered_.size())) {
        return;
    }
    const Entry& entry = entries_[static_cast<std::size_t>(filtered_[selected_])];
    if (on_preview_) {
        on_preview_(entry.id);
    }
}

void ComposerThemePicker::open(const std::string& active_theme_id, bool draft_message_saved) {
    open_ = true;
    draft_message_saved_ = draft_message_saved;
    filter_.clear();
    selected_ = 0;
    list_scroll_ = 0;
    rebuild_entries();
    rebuild_filter(false);
    for (int row = 0; row < static_cast<int>(filtered_.size()); ++row) {
        const Entry& entry = entries_[static_cast<std::size_t>(filtered_[row])];
        if (entry.id == active_theme_id) {
            selected_ = row;
            break;
        }
    }
    sync_scroll(visible_list_rows(panel_height()));
    preview_selection();
}

void ComposerThemePicker::close() {
    open_ = false;
    draft_message_saved_ = false;
    filter_.clear();
    filtered_.clear();
    selected_ = 0;
    list_scroll_ = 0;
}

void ComposerThemePicker::set_filter(const std::string& query) {
    if (filter_ == query) {
        return;
    }
    filter_ = query;
    selected_ = 0;
    list_scroll_ = 0;
    rebuild_filter();
}

const std::string& ComposerThemePicker::selected_theme_id() const {
    static const std::string kEmpty;
    if (!open_ || filtered_.empty() || selected_ < 0 || selected_ >= static_cast<int>(filtered_.size())) {
        return kEmpty;
    }
    return entries_[static_cast<std::size_t>(filtered_[selected_])].id;
}

int ComposerThemePicker::visible_list_rows(int panel_h) const {
    if (panel_h <= 0) {
        return 1;
    }
    return std::max(1, panel_h - 2 - header_row_count(draft_message_saved_));
}

int ComposerThemePicker::panel_height(int max_rows) const {
    if (!open_) {
        return 0;
    }
    const int row_count = std::max(1, static_cast<int>(filtered_.empty() ? 1 : filtered_.size()));
    const int visible = std::min(std::min(max_rows, kPanelMaxListRows), row_count);
    const int inner = header_row_count(draft_message_saved_) + visible;
    return std::min(kPanelMaxListRows + header_row_count(draft_message_saved_) + 2, inner + 2);
}

tuinator::Rect ComposerThemePicker::panel_rect() const {
    const int h = panel_height();
    const int w = bounds_.width <= 0 ? kPanelMaxWidth : std::min(bounds_.width, kPanelMaxWidth);
    const int x = bounds_.width <= 0 ? 0 : std::max(0, (bounds_.width - w) / 2);
    return {x, 0, w, h};
}

int ComposerThemePicker::list_start_y(const tuinator::Rect& panel) const {
    return panel.y + 1 + header_row_count(draft_message_saved_);
}

tuinator::Size ComposerThemePicker::preferred_size(int width, int max_rows) const {
    if (!open_) {
        return {0, 0};
    }
    return {width, panel_height(max_rows)};
}

void ComposerThemePicker::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
}

void ComposerThemePicker::paint(tuinator::PaintContext& ctx) const {
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
    const int inner_h = panel.height - 2;

    tuinator::Style panel_fill = ui_overlay_panel_style();
    panel_fill.background_rgb = ui_input_chrome_fill();
    ctx.canvas.fill_rect({inner_x, inner_y, inner_w, inner_h}, ' ', panel_fill);

    tuinator::Style caption = ui_resolve_main_text(theme_.heading);
    caption.bold = true;
    caption.background_rgb = ui_input_chrome_fill();
    const std::string picker_title = ui_tr("common", "theme_picker.title");
    ctx.canvas.draw_text({panel.x + 2, panel.y},
                         " " + (picker_title.empty() ? "Color themes" : picker_title) + " ", caption);

    tuinator::Style row_style = ui_resolve_main_text(theme_.label);
    row_style.background_rgb = ui_input_chrome_fill();
    tuinator::Style row_active = row_style;
    row_active.background_rgb = ui_list_active_bg();
    row_active.foreground_rgb = ui_list_active_fg();
    tuinator::Style hint = ui_resolve_main_text(ui_secondary_style(theme_));
    hint.background_rgb = ui_input_chrome_fill();

    int y = panel.y + 1;

    if (draft_message_saved_) {
        tuinator::Style draft = hint;
        draft.dim = true;
        const std::string draft_line = ui_tr("common", "theme_picker.draft_saved");
        const std::string draft_display =
            draft_line.empty() ? "Draft saved — Esc restores your message" : draft_line;
        ctx.canvas.fill_rect({inner_x, y, inner_w, 1}, ' ', panel_fill);
        const std::size_t draft_bytes = tuinator::text_byte_length_for_width(draft_display, inner_w);
        ctx.canvas.draw_text({inner_x, y}, draft_display.substr(0, draft_bytes), draft);
        ++y;
    }

    const int max_rows = visible_list_rows(panel.height);
    const int list_y = y;

    if (filtered_.empty()) {
        tuinator::Style empty = row_style;
        empty.dim = true;
        ctx.canvas.fill_rect({inner_x, list_y, inner_w, 1}, ' ', panel_fill);
        const std::string no_matches = ui_tr("common", "theme_picker.no_matches");
        const std::string no_matches_display = no_matches.empty() ? "(no matches)" : no_matches;
        ctx.canvas.draw_text({inner_x, list_y}, " " + no_matches_display, empty);
        return;
    }

    for (int row = 0; row < max_rows; ++row) {
        const int idx = list_scroll_ + row;
        if (idx >= static_cast<int>(filtered_.size())) {
            break;
        }
        const Entry& entry = entries_[static_cast<std::size_t>(filtered_[idx])];
        const bool active = idx == selected_;
        const tuinator::Style row_bg = active ? row_active : panel_fill;
        ctx.canvas.fill_rect({inner_x, list_y + row, inner_w, 1}, ' ', row_bg);

        tuinator::Style label_style = active ? row_active : row_style;
        const std::string label = ui_is_bundled_theme(entry.id) ? entry.subtitle : entry.id;
        const std::size_t label_bytes = tuinator::text_byte_length_for_width(label, inner_w);
        ctx.canvas.draw_text({inner_x, list_y + row}, label.substr(0, label_bytes), label_style);
    }

    std::string scroll_text = std::to_string(selected_ + 1) + "/" + std::to_string(filtered_.size());
    if (static_cast<int>(filtered_.size()) > max_rows) {
        scroll_text += "  ↑↓";
    }
    draw_bottom_border_caption(ctx, panel, scroll_text, frame);
}

bool ComposerThemePicker::move_selection(int delta) {
    if (!open_ || filtered_.empty()) {
        return false;
    }
    const int n = static_cast<int>(filtered_.size());
    selected_ = std::clamp(selected_ + delta, 0, n - 1);
    sync_scroll(visible_list_rows(panel_height()));
    preview_selection();
    return true;
}

bool ComposerThemePicker::move_page(int rows) {
    return move_selection(rows);
}

bool ComposerThemePicker::jump_selection(bool to_end) {
    if (!open_ || filtered_.empty()) {
        return false;
    }
    selected_ = to_end ? static_cast<int>(filtered_.size()) - 1 : 0;
    sync_scroll(visible_list_rows(panel_height()));
    preview_selection();
    return true;
}

bool ComposerThemePicker::select_visible_row(int visible_index) {
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

bool ComposerThemePicker::hit_test_row(tuinator::Point local, int& visible_index) const {
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

void ComposerThemePicker::commit_selection() {
    const std::string& id = selected_theme_id();
    if (id.empty()) {
        return;
    }
    if (on_commit_) {
        on_commit_(id);
    }
}
