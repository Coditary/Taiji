#include "composer_theme_picker.hpp"

#include "skim_fuzzy.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/glyphs.hpp>
#include <tuinator/render/text.hpp>

#include <algorithm>

namespace {

int visible_row_count(int panel_h) {
    return std::max(1, panel_h - 3);
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
            entry.line = id + "  ·  " + ui_theme_display_name(id);
        } else {
            entry.line = id + "  ·  user";
        }
        entry.meta = id + " " + ui_theme_display_name(id) + " theme color palette stylesheet";
        entries_.push_back(std::move(entry));
    }
}

void ComposerThemePicker::rebuild_filter() {
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
    sync_scroll(visible_row_count(panel_height()));
    preview_selection();
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
    rebuild_filter();
    for (int row = 0; row < static_cast<int>(filtered_.size()); ++row) {
        const Entry& entry = entries_[static_cast<std::size_t>(filtered_[row])];
        if (entry.id == active_theme_id) {
            selected_ = row;
            break;
        }
    }
    sync_scroll(visible_row_count(panel_height()));
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

int ComposerThemePicker::panel_height(int max_rows) const {
    if (!open_) {
        return 0;
    }
    const int rows = std::max(1, static_cast<int>(filtered_.empty() ? 1 : filtered_.size()));
    int h = std::min(max_rows + 3, rows + 3);
    if (draft_message_saved_) {
        h += 1;
    }
    return h;
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

    const tuinator::Rect panel{0, 0, bounds_.width, bounds_.height};
    const tuinator::BorderGlyphs border = tuinator::unicode_rounded_border_glyphs();
    tuinator::Style frame = theme_.label;
    frame.background_rgb = ui_input_chrome_fill();
    frame.foreground_rgb = ui_border_chrome();
    ctx.canvas.draw_box(panel, frame, border);

    tuinator::Style title = theme_.heading;
    title.bold = true;
    title.background_rgb = ui_input_chrome_fill();
    ctx.canvas.draw_text({2, 0}, " Color themes ", title);

    tuinator::Style row_style = ui_resolve_main_text(theme_.label);
    row_style.background_rgb = ui_input_chrome_fill();

    int list_y = 2;
    if (draft_message_saved_) {
        tuinator::Style draft = row_style;
        draft.dim = true;
        draft.foreground_rgb = ui_secondary_text();
        ctx.canvas.draw_text({2, 1}, " Draft message saved — Esc restores it ", draft);
        list_y = 3;
    }
    const int max_rows = std::max(1, visible_row_count(panel.height - (list_y - 1)));

    if (filtered_.empty()) {
        tuinator::Style empty = row_style;
        empty.dim = true;
        ctx.canvas.draw_text({2, list_y}, " (no matches)", empty);
        return;
    }

    for (int row = 0; row < max_rows; ++row) {
        const int idx = list_scroll_ + row;
        if (idx >= static_cast<int>(filtered_.size())) {
            break;
        }
        const Entry& entry = entries_[static_cast<std::size_t>(filtered_[idx])];
        tuinator::Style line_style = idx == selected_ ? ui_list_selection_style(true) : row_style;
        std::string prefix = idx == selected_ ? "> " : "  ";
        std::string line = prefix + entry.line;
        const std::size_t bytes = tuinator::text_byte_length_for_width(line, panel.width - 4);
        ctx.canvas.draw_text({2, list_y + row}, line.substr(0, bytes), line_style);
    }

    if (static_cast<int>(filtered_.size()) > max_rows) {
        tuinator::Style hint = row_style;
        hint.dim = true;
        const std::string scroll_hint =
            std::to_string(selected_ + 1) + "/" + std::to_string(filtered_.size()) + "  ↑↓";
        ctx.canvas.draw_text({panel.width - static_cast<int>(scroll_hint.size()) - 2, panel.height - 1}, scroll_hint,
                             hint);
    }
}

bool ComposerThemePicker::move_selection(int delta) {
    if (!open_ || filtered_.empty()) {
        return false;
    }
    const int n = static_cast<int>(filtered_.size());
    selected_ = std::clamp(selected_ + delta, 0, n - 1);
    sync_scroll(visible_row_count(panel_height()));
    preview_selection();
    return true;
}

bool ComposerThemePicker::move_page(int rows) {
    return move_selection(rows);
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
    const int list_y = 2;
    const int max_rows = visible_row_count(bounds_.height);
    if (local.y < list_y || local.y >= list_y + max_rows || local.x < 0 || local.x >= bounds_.width) {
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
