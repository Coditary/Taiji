#pragma once

#include <tuinator/tuinator.hpp>

#include <functional>
#include <string>
#include <vector>

/// One row in a composer fuzzy list (themes, @-mentions, etc.).
struct ComposerFuzzyPickerItem {
    std::string id;
    std::string line;
    std::string meta;
    /// If empty, `line` is shown in the list.
    std::string label;
};

struct ComposerFuzzyPickerView {
    std::string title;
    std::string no_matches_text = "(no matches)";
    /// Extra hint on the bottom border (e.g. "  Tab").
    std::string footer_suffix;
    int header_rows = 0;
    std::function<void(tuinator::PaintContext& ctx, const tuinator::Rect& inner, int y, int inner_w,
                       const tuinator::Style& panel_fill)>
        paint_header;
    std::function<std::string(const ComposerFuzzyPickerItem& item)> format_row_label;
};

/// Shared fuzzy list overlay used by theme picker, @-mention picker, and similar UIs.
class ComposerFuzzyPicker {
  public:
    explicit ComposerFuzzyPicker(tuinator::Theme theme);

    bool is_open() const { return open_; }
    void open();
    void close();

    void set_view(ComposerFuzzyPickerView view);
    void set_items(std::vector<ComposerFuzzyPickerItem> items);
    void set_filter(const std::string& query);
    void set_initial_selection_id(const std::string& id);

    const std::string& selected_id() const;

    tuinator::Size preferred_size(int width, int max_rows = 10) const;
    void layout(tuinator::Rect bounds);
    void paint(tuinator::PaintContext& ctx) const;

    bool move_selection(int delta);
    bool move_page(int rows);
    bool jump_selection(bool to_end);
    bool select_visible_row(int visible_index);
    bool hit_test_row(tuinator::Point local, int& visible_index) const;

    tuinator::Rect bounds() const { return bounds_; }

    void set_on_preview(std::function<void(const std::string& id)> handler) { on_preview_ = std::move(handler); }
    void set_on_commit(std::function<void(const std::string& id)> handler) { on_commit_ = std::move(handler); }
    void commit_selection();
    void preview_selection();

  private:
    tuinator::Rect panel_rect() const;
    int panel_height(int max_rows = 10) const;
    int visible_list_rows(int panel_h) const;
    int list_start_y(const tuinator::Rect& panel) const;

    void rebuild_filter(bool preview = true);
    void sync_scroll(int max_visible);

    tuinator::Theme theme_;
    ComposerFuzzyPickerView view_;
    bool open_ = false;
    std::vector<ComposerFuzzyPickerItem> items_;
    std::vector<int> filtered_;
    std::string filter_;
    int selected_ = 0;
    int list_scroll_ = 0;
    tuinator::Rect bounds_{};
    std::function<void(const std::string& id)> on_preview_;
    std::function<void(const std::string& id)> on_commit_;
};

/// Key routing shared by composer fuzzy overlays (theme, @-mention, …).
bool composer_fuzzy_picker_route_key(const tuinator::KeyPress& key, ComposerFuzzyPicker& picker,
                                   const std::function<void()>& on_escape,
                                   const std::function<void()>& on_confirm);
