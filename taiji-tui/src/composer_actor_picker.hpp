#pragma once

#include "composer_fuzzy_picker.hpp"

#include <functional>
#include <string>
#include <vector>

class ComposerActorPicker {
  public:
    explicit ComposerActorPicker(tuinator::Theme theme);

    bool is_open() const { return picker_.is_open(); }
    void open();
    void close();

    void set_entries(std::vector<ComposerFuzzyPickerItem> items);
    void set_filter(const std::string& query);
    const std::string& selected_id() const;

    tuinator::Size preferred_size(int width, int max_rows = 10) const;

    void layout(tuinator::Rect bounds);
    void paint(tuinator::PaintContext& ctx) const;

    bool move_selection(int delta);
    bool move_page(int rows);
    bool jump_selection(bool to_end);
    bool select_visible_row(int visible_index);
    bool hit_test_row(tuinator::Point local, int& visible_index) const;

    tuinator::Rect bounds() const { return picker_.bounds(); }

    void set_on_commit(std::function<void(const std::string& id)> handler);
    void commit_selection();

    bool route_key(const tuinator::KeyPress& key, const std::function<void()>& on_escape,
                   const std::function<void()>& on_confirm);

  private:
    void sync_view();

    ComposerFuzzyPicker picker_;
    std::vector<ComposerFuzzyPickerItem> entries_;
    std::function<void(const std::string&)> on_commit_;
};
