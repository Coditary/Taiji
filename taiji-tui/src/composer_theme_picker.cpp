#include "composer_theme_picker.hpp"

#include "ui_locale.hpp"
#include "ui_theme.hpp"

#include <tuinator/render/text.hpp>

ComposerThemePicker::ComposerThemePicker(tuinator::Theme theme)
    : theme_(theme), picker_(std::move(theme)) {
    picker_.set_on_preview([this](const std::string& id) {
        if (on_preview_) {
            on_preview_(id);
        }
    });
    picker_.set_on_commit([this](const std::string& id) {
        if (on_commit_) {
            on_commit_(id);
        }
    });
}

void ComposerThemePicker::set_on_preview(std::function<void(const std::string& theme_id)> handler) {
    on_preview_ = std::move(handler);
}

void ComposerThemePicker::set_on_commit(std::function<void(const std::string& theme_id)> handler) {
    on_commit_ = std::move(handler);
}

void ComposerThemePicker::rebuild_entries() {
    catalog_.clear();
    const std::vector<std::string> ids = ui_list_theme_ids();
    for (const std::string& id : ids) {
        ComposerFuzzyPickerItem entry;
        entry.id = id;
        std::string subtitle;
        if (ui_is_bundled_theme(id)) {
            subtitle = ui_theme_display_name(id);
        } else {
            subtitle = "user theme";
        }
        entry.line = id + "  ·  " + subtitle;
        entry.meta = id + " " + subtitle + " theme color palette stylesheet";
        entry.label = ui_is_bundled_theme(id) ? subtitle : id;
        catalog_.push_back(std::move(entry));
    }
    picker_.set_items(catalog_);
}

void ComposerThemePicker::sync_view() {
    ComposerFuzzyPickerView view;
    const std::string picker_title = ui_tr("common", "theme_picker.title");
    view.title = picker_title.empty() ? "Color themes" : picker_title;
    const std::string no_matches = ui_tr("common", "theme_picker.no_matches");
    view.no_matches_text = no_matches.empty() ? "(no matches)" : no_matches;
    view.header_rows = draft_message_saved_ ? 1 : 0;
    if (draft_message_saved_) {
        view.paint_header = [this](tuinator::PaintContext& ctx, const tuinator::Rect& inner, int y, int inner_w,
                                   const tuinator::Style& panel_fill) {
            tuinator::Style draft = ui_resolve_main_text(ui_secondary_style(theme_));
            draft.dim = true;
            draft.background_rgb = ui_input_chrome_fill();
            const std::string draft_line = ui_tr("common", "theme_picker.draft_saved");
            const std::string draft_display =
                draft_line.empty() ? "Draft saved — Esc restores your message" : draft_line;
            ctx.canvas.fill_rect({inner.x, y, inner_w, 1}, ' ', panel_fill);
            const std::size_t draft_bytes = tuinator::text_byte_length_for_width(draft_display, inner_w);
            ctx.canvas.draw_text({inner.x, y}, draft_display.substr(0, draft_bytes), draft);
        };
    }
    picker_.set_view(std::move(view));
}

void ComposerThemePicker::open(const std::string& active_theme_id, bool draft_message_saved) {
    draft_message_saved_ = draft_message_saved;
    sync_view();
    rebuild_entries();
    picker_.open();
    picker_.set_filter("");
    picker_.set_initial_selection_id(active_theme_id);
}

void ComposerThemePicker::close() {
    draft_message_saved_ = false;
    picker_.close();
}

void ComposerThemePicker::set_filter(const std::string& query) {
    picker_.set_filter(query);
}

const std::string& ComposerThemePicker::selected_theme_id() const {
    return picker_.selected_id();
}

tuinator::Size ComposerThemePicker::preferred_size(int width, int max_rows) const {
    return picker_.preferred_size(width, max_rows);
}

void ComposerThemePicker::layout(tuinator::Rect bounds) {
    picker_.layout(bounds);
}

void ComposerThemePicker::paint(tuinator::PaintContext& ctx) const {
    picker_.paint(ctx);
}

bool ComposerThemePicker::move_selection(int delta) {
    return picker_.move_selection(delta);
}

bool ComposerThemePicker::move_page(int rows) {
    return picker_.move_page(rows);
}

bool ComposerThemePicker::jump_selection(bool to_end) {
    return picker_.jump_selection(to_end);
}

bool ComposerThemePicker::select_visible_row(int visible_index) {
    return picker_.select_visible_row(visible_index);
}

bool ComposerThemePicker::hit_test_row(tuinator::Point local, int& visible_index) const {
    return picker_.hit_test_row(local, visible_index);
}

void ComposerThemePicker::commit_selection() {
    picker_.commit_selection();
}
