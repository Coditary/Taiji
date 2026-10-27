#include "composer_actor_picker.hpp"

#include "ui_locale.hpp"
#include "ui_theme.hpp"

ComposerActorPicker::ComposerActorPicker(tuinator::Theme theme) : picker_(std::move(theme)) {
    picker_.set_on_commit([this](const std::string& id) {
        if (on_commit_) {
            on_commit_(id);
        }
    });
    sync_view();
}

void ComposerActorPicker::set_on_commit(std::function<void(const std::string& id)> handler) {
    on_commit_ = std::move(handler);
}

void ComposerActorPicker::sync_view() {
    ComposerFuzzyPickerView view;
    const std::string picker_title = ui_tr("common", "actor_picker.title");
    view.title = picker_title.empty() ? "Context" : picker_title;
    const std::string no_matches = ui_tr("common", "actor_picker.no_matches");
    view.no_matches_text = no_matches.empty() ? "(no matches)" : no_matches;
    view.footer_suffix = "  Tab";
    view.format_row_label = [](const ComposerFuzzyPickerItem& item) {
        if (!item.label.empty()) {
            return item.label;
        }
        return item.line;
    };
    picker_.set_view(std::move(view));
}

void ComposerActorPicker::set_entries(std::vector<ComposerFuzzyPickerItem> items) {
    entries_ = std::move(items);
    picker_.set_items(entries_);
}

void ComposerActorPicker::open() {
    picker_.open();
    picker_.set_filter("");
}

void ComposerActorPicker::close() {
    picker_.close();
}

void ComposerActorPicker::set_filter(const std::string& query) {
    picker_.set_filter(query);
}

const std::string& ComposerActorPicker::selected_id() const {
    return picker_.selected_id();
}

tuinator::Size ComposerActorPicker::preferred_size(int width, int max_rows) const {
    return picker_.preferred_size(width, max_rows);
}

void ComposerActorPicker::layout(tuinator::Rect bounds) {
    picker_.layout(bounds);
}

void ComposerActorPicker::paint(tuinator::PaintContext& ctx) const {
    picker_.paint(ctx);
}

bool ComposerActorPicker::move_selection(int delta) {
    return picker_.move_selection(delta);
}

bool ComposerActorPicker::move_page(int rows) {
    return picker_.move_page(rows);
}

bool ComposerActorPicker::jump_selection(bool to_end) {
    return picker_.jump_selection(to_end);
}

bool ComposerActorPicker::select_visible_row(int visible_index) {
    return picker_.select_visible_row(visible_index);
}

bool ComposerActorPicker::hit_test_row(tuinator::Point local, int& visible_index) const {
    return picker_.hit_test_row(local, visible_index);
}

void ComposerActorPicker::commit_selection() {
    picker_.commit_selection();
}

bool ComposerActorPicker::route_key(const tuinator::KeyPress& key, const std::function<void()>& on_escape,
                                    const std::function<void()>& on_confirm) {
    return composer_fuzzy_picker_route_key(key, picker_, on_escape, on_confirm);
}
