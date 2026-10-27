#pragma once

#include "composer_fuzzy_picker.hpp"

#include <functional>
#include <string>

class ComposerThemePicker {
  public:
    explicit ComposerThemePicker(tuinator::Theme theme);

    bool is_open() const { return picker_.is_open(); }
    void open(const std::string& active_theme_id, bool draft_message_saved = false);
    void close();

    void set_filter(const std::string& query);
    const std::string& selected_theme_id() const;

    tuinator::Size preferred_size(int width, int max_rows = 10) const;

    void layout(tuinator::Rect bounds);
    void paint(tuinator::PaintContext& ctx) const;

    bool move_selection(int delta);
    bool move_page(int rows);
    bool jump_selection(bool to_end);
    bool select_visible_row(int visible_index);
    bool hit_test_row(tuinator::Point local, int& visible_index) const;

    tuinator::Rect bounds() const { return picker_.bounds(); }

    void set_on_preview(std::function<void(const std::string& theme_id)> handler);
    void set_on_commit(std::function<void(const std::string& theme_id)> handler);
    void commit_selection();

  private:
    void rebuild_entries();
    void sync_view();

    tuinator::Theme theme_;
    ComposerFuzzyPicker picker_;
    bool draft_message_saved_ = false;
    std::vector<ComposerFuzzyPickerItem> catalog_;
    std::function<void(const std::string&)> on_preview_;
    std::function<void(const std::string&)> on_commit_;
};
